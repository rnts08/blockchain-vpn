package crypto

import "testing"

func TestResolveKeyStorageMode(t *testing.T) {
	tests := []struct {
		in      string
		wantErr bool
	}{
		{"", false},
		{"file", false},
		{"auto", false},
		{"keychain", false},
		{"libsecret", false},
		{"dpapi", false},
		{"bad-mode", true},
	}

	for _, tc := range tests {
		got, err := ResolveKeyStorageMode(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("expected error for mode %q", tc.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("unexpected error for mode %q: %v", tc.in, err)
		}
		if got == "" {
			t.Fatalf("resolved mode should not be empty for input %q", tc.in)
		}
	}
}

func TestKeyStorageStatus(t *testing.T) {
	resolved, ok, detail := KeyStorageStatus("file")
	if resolved != "file" {
		t.Fatalf("expected resolved file, got %q", resolved)
	}
	if !ok {
		t.Fatal("file mode should always be supported")
	}
	if detail == "" {
		t.Fatal("expected detail to be populated")
	}
}
