package crypto

import (
	"runtime"
	"testing"
)

func TestResolveKeyStorageMode(t *testing.T) {
	origLookup := commandLookup
	defer func() { commandLookup = origLookup }()

	tests := []struct {
		name    string
		mode    string
		goos    string
		exists  map[string]bool
		want    string
		wantErr bool
	}{
		{"EmptyMode", "", "linux", nil, "file", false},
		{"FileMode", "file", "linux", nil, "file", false},
		{"AutoLinuxNoSecretTool", "auto", "linux", map[string]bool{"secret-tool": false}, "file", false},
		{"AutoLinuxWithSecretTool", "auto", "linux", map[string]bool{"secret-tool": true}, "libsecret", false},
		{"AutoDarwinNoSecurity", "auto", "darwin", map[string]bool{"security": false}, "file", false},
		{"AutoDarwinWithSecurity", "auto", "darwin", map[string]bool{"security": true}, "keychain", false},
		{"AutoWindowsNoPS", "auto", "windows", map[string]bool{"powershell": false, "pwsh": false}, "file", false},
		{"AutoWindowsWithPS", "auto", "windows", map[string]bool{"powershell": true}, "dpapi", false},
		{"ExplicitLibsecret", "libsecret", "linux", nil, "libsecret", false},
		{"UnknownMode", "invalid", "linux", nil, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock command exists
			commandLookup = func(name string) bool {
				if tt.exists == nil {
					return false
				}
				v, ok := tt.exists[name]
				if !ok {
					return false
				}
				return v
			}

			// Note: We can't easily mock runtime.GOOS, so we can only fully test the current OS logic
			// unless we refactor ResolveKeyStorageMode to accept OS as param.
			// But for now, we'll only run cases matching current OS.
			if tt.goos != runtime.GOOS && tt.mode == "auto" {
				t.Skipf("Skipping %s test on %s", tt.goos, runtime.GOOS)
			}

			got, err := ResolveKeyStorageMode(tt.mode)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveKeyStorageMode(%q) error = %v, wantErr %v", tt.mode, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ResolveKeyStorageMode(%q) = %q, want %q", tt.mode, got, tt.want)
			}
		})
	}
}

func TestKeyStorageStatus(t *testing.T) {
	origLookup := commandLookup
	defer func() { commandLookup = origLookup }()

	commandLookup = func(name string) bool { return true } // assume all exist

	resolved, supported, detail := KeyStorageStatus("auto")
	if resolved == "" {
		t.Errorf("KeyStorageStatus returned empty resolved mode")
	}
	if !supported {
		t.Errorf("KeyStorageStatus returned unsupported for auto with tools present")
	}
	if detail == "" {
		t.Errorf("KeyStorageStatus returned empty detail")
	}
}

func TestKeyStorageStatus_File(t *testing.T) {
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
