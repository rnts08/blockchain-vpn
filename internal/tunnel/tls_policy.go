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

func ResolveTLSPolicy(minVersion, profile string) (TLSPolicy, error) {
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

	if p == "compat" && out.MinVersion <= tls.VersionTLS12 {
		out.CipherSuites = []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		}
	}
	out.CipherNames = cipherNames(out.CipherSuites)
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
