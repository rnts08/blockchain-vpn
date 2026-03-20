package crypto

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func TestRotateProviderKey_FileMode(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "keystore-rotate-test")
	defer os.RemoveAll(tmpDir)
	keyPath := filepath.Join(tmpDir, "provider.key")

	oldPass := []byte("oldpassword123")
	newPass := []byte("newpassword456")

	_, err := GenerateAndEncryptKey(keyPath, oldPass)
	if err != nil {
		t.Fatalf("failed to create initial key: %v", err)
	}

	err = RotateProviderKey(keyPath, oldPass, newPass, "file", "")
	if err != nil {
		t.Fatalf("RotateProviderKey failed: %v", err)
	}

	loadedKey, err := LoadAndDecryptKey(keyPath, newPass)
	if err != nil {
		t.Fatalf("failed to load rotated key with new password: %v", err)
	}
	if loadedKey == nil {
		t.Fatal("expected loaded key to be non-nil")
	}

	_, err = LoadAndDecryptKey(keyPath, oldPass)
	if err == nil {
		t.Error("old password should no longer work after rotation")
	}
}

func TestRotateProviderKey_FileModeBackupCreated(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "keystore-backup-test")
	defer os.RemoveAll(tmpDir)
	keyPath := filepath.Join(tmpDir, "provider.key")

	_, _ = GenerateAndEncryptKey(keyPath, []byte("oldpass"))

	err := RotateProviderKey(keyPath, []byte("oldpass"), []byte("newpass"), "file", "")
	if err != nil {
		t.Fatalf("RotateProviderKey failed: %v", err)
	}

	entries, _ := os.ReadDir(tmpDir)
	var backupFound bool
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "provider.key.bak-") {
			backupFound = true
			break
		}
	}
	if !backupFound {
		t.Error("expected backup file to be created with .bak- timestamp suffix")
	}
}

func TestRotateProviderKey_FileModeWrongPassword(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "keystore-wrongpass-test")
	defer os.RemoveAll(tmpDir)
	keyPath := filepath.Join(tmpDir, "provider.key")

	_, _ = GenerateAndEncryptKey(keyPath, []byte("correctold"))

	err := RotateProviderKey(keyPath, []byte("wrongold"), []byte("newpass"), "file", "")
	if err == nil {
		t.Error("expected error for wrong old password")
	}
}

func TestRotateProviderKey_FileModeEmptyPassword(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "keystore-emptypass-test")
	defer os.RemoveAll(tmpDir)
	keyPath := filepath.Join(tmpDir, "provider.key")

	_, _ = GenerateAndEncryptKey(keyPath, []byte("password"))

	err := RotateProviderKey(keyPath, []byte(""), []byte("newpass"), "file", "")
	if err == nil {
		t.Error("expected error for empty old password")
	}

	err = RotateProviderKey(keyPath, []byte("password"), []byte(""), "file", "")
	if err == nil {
		t.Error("expected error for empty new password")
	}
}

func TestRotateProviderKey_SecureStoreMock(t *testing.T) {
	origLookup := commandLookup
	origGOOS := currentGOOS
	origRun := runCommand
	defer func() {
		commandLookup = origLookup
		currentGOOS = origGOOS
		runCommand = origRun
	}()

	tmpDir, _ := os.MkdirTemp("", "keystore-sstore-rotate")
	defer os.RemoveAll(tmpDir)
	keyPath := filepath.Join(tmpDir, "provider.key")

	currentGOOS = "linux"
	commandLookup = func(name string) bool { return name == "secret-tool" }

	var storedSecret = strings.Repeat("11", 32)
	runCommand = func(name string, stdin string, args ...string) ([]byte, error) {
		if name == "secret-tool" {
			if args[0] == "store" {
				storedSecret = stdin
				return []byte(""), nil
			}
			if args[0] == "lookup" {
				return []byte(storedSecret), nil
			}
		}
		return nil, fmt.Errorf("unknown command")
	}

	err := RotateProviderKey(keyPath, nil, nil, "libsecret", "TestRotateSvc")
	if err != nil {
		t.Fatalf("RotateProviderKey (secure store) failed: %v", err)
	}

	if storedSecret == "" {
		t.Fatal("expected secret to be stored")
	}
	if len(storedSecret) != 64 {
		t.Errorf("expected 64-char hex key, got %d chars", len(storedSecret))
	}
}
