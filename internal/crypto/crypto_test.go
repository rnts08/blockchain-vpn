package crypto

import (
	"bytes"
	"os"
	"testing"
)

func TestLoadAndDecryptKey_InvalidKeyBytes(t *testing.T) {
	t.Parallel()

	// Test: trying to load a non-existent file
	_, err := LoadAndDecryptKey("nonexistent_file_path_12345", []byte("password"))
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestEncrypt_DecryptRoundtrip(t *testing.T) {
	t.Parallel()

	password := []byte("testpassword123")
	originalData := []byte("hello world this is test data")

	encrypted, err := Encrypt(originalData, password)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	decrypted, err := Decrypt(encrypted, password)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if !bytes.Equal(decrypted, originalData) {
		t.Errorf("decrypted = %v, want %v", decrypted, originalData)
	}
}

func TestDecrypt_WrongPassword(t *testing.T) {
	t.Parallel()

	data := []byte("secret data")
	password := []byte("correctpassword")
	wrongPassword := []byte("wrongpassword")

	encrypted, err := Encrypt(data, password)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	_, err = Decrypt(encrypted, wrongPassword)
	if err == nil {
		t.Error("expected error with wrong password")
	}
}

func TestDecrypt_TruncatedData(t *testing.T) {
	t.Parallel()

	password := []byte("password")
	encrypted, err := Encrypt([]byte("data"), password)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Truncate the encrypted data to trigger "too short" error
	truncated := encrypted[:len(encrypted)/2]
	_, err = Decrypt(truncated, password)
	if err == nil {
		t.Error("expected error for truncated data")
	}
}

func TestLoadAndDecryptKey_InvalidDecryptedLength(t *testing.T) {
	t.Parallel()

	// Create encrypted data with wrong length plaintext
	password := []byte("testpassword")
	// Encrypt 10 bytes (not 32 bytes which is required for priv key)
	encrypted, err := Encrypt([]byte("shortdata"), password)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Write to temp file and try to load
	tmpFile := t.TempDir() + "/testkey"
	if err := os.WriteFile(tmpFile, encrypted, 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err = LoadAndDecryptKey(tmpFile, password)
	if err == nil {
		t.Error("expected error for invalid key length")
	}
	if err != nil && !contains(err.Error(), "invalid decrypted key length") {
		t.Errorf("expected 'invalid decrypted key length' error, got: %v", err)
	}
}

// Helper functions
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
