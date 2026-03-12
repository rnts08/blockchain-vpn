package tunnel

import (
	"crypto/tls"
	"fmt"
	"strings"
)

type TLSPolicy struct {
	MinVersion      uint16
	MinVersionLabel string
	Profile         string
	CipherSuites    []uint16
	CipherNames     []string
}

func ResolveTLSPolicy(minVersion, profile string, customCiphers []string) (TLSPolicy, error) {
	p := strings.ToLower(strings.TrimSpace(profile))
	if p == "" {
		p = "modern"
	}
	switch p {
	case "modern", "compat":
	default:
		return TLSPolicy{}, fmt.Errorf("invalid tls profile %q", profile)
	}

	mv := strings.TrimSpace(minVersion)
	if mv == "" {
		if p == "compat" {
			mv = "1.2"
		} else {
			mv = "1.3"
		}
	}

	out := TLSPolicy{Profile: p}
	switch mv {
	case "1.2":
		out.MinVersion = tls.VersionTLS12
		out.MinVersionLabel = "1.2"
	case "1.3":
		out.MinVersion = tls.VersionTLS13
		out.MinVersionLabel = "1.3"
	default:
		return TLSPolicy{}, fmt.Errorf("invalid tls min version %q", minVersion)
	}

	if len(customCiphers) > 0 {
		ids, names, err := resolveCipherNames(customCiphers)
		if err != nil {
			return TLSPolicy{}, err
		}
		out.CipherSuites = ids
		out.CipherNames = names
	} else if p == "compat" && out.MinVersion <= tls.VersionTLS12 {
		out.CipherSuites = []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		}
		out.CipherNames = cipherNames(out.CipherSuites)
	} else {
		out.CipherNames = []string{"tls13-default"}
	}
	return out, nil
}

func cipherNames(ids []uint16) []string {
	if len(ids) == 0 {
		return []string{"tls13-default"}
	}
	all := tls.CipherSuites()
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		name := fmt.Sprintf("0x%04x", id)
		for _, suite := range all {
			if suite.ID == id {
				name = suite.Name
				break
			}
		}
		names = append(names, name)
	}
	return names
}

func resolveCipherNames(names []string) ([]uint16, []string, error) {
	if len(names) == 0 {
		return nil, nil, nil
	}

	all := tls.CipherSuites()
	idMap := make(map[string]uint16)
	for _, suite := range all {
		idMap[strings.ToUpper(suite.Name)] = suite.ID
		idMap[fmt.Sprintf("0x%04x", suite.ID)] = suite.ID
	}

	ids := make([]uint16, 0, len(names))
	outNames := make([]string, 0, len(names))
	for _, name := range names {
		upper := strings.ToUpper(strings.TrimSpace(name))
		if id, ok := idMap[upper]; ok {
			ids = append(ids, id)
			for _, suite := range all {
				if suite.ID == id {
					outNames = append(outNames, suite.Name)
					break
				}
			}
		} else {
			return nil, nil, fmt.Errorf("unknown cipher suite: %s", name)
		}
	}
	return ids, outNames, nil
}
