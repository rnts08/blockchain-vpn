package tunnel

import "strings"

var expectedSecureDNSServers = map[string]struct{}{
	"1.1.1.1": {},
	"8.8.8.8": {},
}

func hasExpectedSecureDNS(servers []string) bool {
	for _, s := range servers {
		if _, ok := expectedSecureDNSServers[strings.TrimSpace(s)]; ok {
			return true
		}
	}
	return false
}
