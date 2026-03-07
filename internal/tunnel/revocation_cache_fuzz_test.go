package tunnel

import (
	"strings"
	"testing"
)

func FuzzParseRevocationEntries(f *testing.F) {
	f.Add("")
	f.Add("# comment\n")
	f.Add("deadbeef\n")
	f.Add("02ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff\n")

	f.Fuzz(func(t *testing.T, content string) {
		_, _ = parseRevocationEntries(strings.NewReader(content))
	})
}
