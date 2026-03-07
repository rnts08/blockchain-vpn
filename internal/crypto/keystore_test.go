package crypto

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveKeyStorageMode(t *testing.T) {
	origLookup := commandLookup
	origGOOS := currentGOOS
	defer func() {
		commandLookup = origLookup
		currentGOOS = origGOOS
	}()

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
			currentGOOS = tt.goos
			commandLookup = func(name string) bool {
				if tt.exists == nil {
					return false
				}
				v, ok := tt.exists[name]
				return ok && v
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
	origGOOS := currentGOOS
	defer func() {
		commandLookup = origLookup
		currentGOOS = origGOOS
	}()

	currentGOOS = "linux"
	commandLookup = func(name string) bool { return true }

	resolved, supported, detail := KeyStorageStatus("auto")
	if resolved != "libsecret" {
		t.Errorf("expected libsecret on linux with tools, got %q", resolved)
	}
	if !supported {
		t.Errorf("KeyStorageStatus returned unsupported for auto with tools present")
	}
	if detail == "" {
		t.Errorf("expected detail message")
	}
}

func TestKeyStorageStatus_File(t *testing.T) {
	resolved, ok, _ := KeyStorageStatus("file")
	if resolved != "file" || !ok {
		t.Errorf("file mode should always be supported")
	}
}

// Complete real-world use case coverage: mocking secure store interaction
func TestLoadOrCreateProviderKey_SecureStoreMock(t *testing.T) {
	origLookup := commandLookup
	origGOOS := currentGOOS
	origRun := runCommand
	defer func() {
		commandLookup = origLookup
		currentGOOS = origGOOS
		runCommand = origRun
	}()

	tmpDir, _ := os.MkdirTemp("", "keystore-test")
	defer os.RemoveAll(tmpDir)
	keyPath := filepath.Join(tmpDir, "provider.key")

	currentGOOS = "linux"
	commandLookup = func(name string) bool { return name == "secret-tool" }

	// Mock successful secret-tool storage and lookup
	var storedSecret string
	runCommand = func(name string, stdin string, args ...string) ([]byte, error) {
		if name == "secret-tool" {
			if args[0] == "store" {
				storedSecret = stdin
				return []byte(""), nil
			}
			if args[0] == "lookup" {
				if storedSecret == "" {
					return []byte(""), nil
				}
				return []byte(storedSecret), nil
			}
		}
		return nil, fmt.Errorf("unknown command")
	}

	// 1. First call creates the key in secure store
	key1, err := LoadOrCreateProviderKey(keyPath, nil, "libsecret", "TestService")
	if err != nil {
		t.Fatalf("LoadOrCreateProviderKey (create) failed: %v", err)
	}
	if key1 == nil {
		t.Fatal("expected key to be generated")
	}

	// 2. Second call loads it
	key2, err := LoadOrCreateProviderKey(keyPath, nil, "libsecret", "TestService")
	if err != nil {
		t.Fatalf("LoadOrCreateProviderKey (load) failed: %v", err)
	}
	if key2 == nil {
		t.Fatal("expected key to be loaded")
	}

	if !bytes.Equal(key1.Serialize(), key2.Serialize()) {
		t.Errorf("keys do not match")
	}
}
