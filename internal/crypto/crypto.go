package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"os"

	"github.com/btcsuite/btcd/btcec/v2"
	"golang.org/x/crypto/pbkdf2"
)

const (
	saltBytes        = 32
	keyBytes         = 32
	pbkdf2Iterations = 4096
)

// Encrypt encrypts data using AES-256-GCM with a key derived from a password.
// The output format is: [salt (32 bytes)][nonce][ciphertext][tag]
func Encrypt(data, password []byte) ([]byte, error) {
	// 1. Generate a random salt.
	salt := make([]byte, saltBytes)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}

	// 2. Derive a key from the password and salt.
	key := pbkdf2.Key(password, salt, pbkdf2Iterations, keyBytes, sha256.New)

	// 3. Create AES-GCM cipher.
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// 4. Generate a random nonce.
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// 5. Encrypt the data.
	ciphertext := gcm.Seal(nil, nonce, data, nil)

	// 6. Prepend salt and nonce to the ciphertext.
	return append(salt, append(nonce, ciphertext...)...), nil
}

// Decrypt decrypts data using AES-256-GCM.
func Decrypt(data, password []byte) ([]byte, error) {
	if len(data) < saltBytes {
		return nil, fmt.Errorf("encrypted data is too short")
	}

	// 1. Derive key from password and salt.
	salt := data[:saltBytes]
	key := pbkdf2.Key(password, salt, pbkdf2Iterations, keyBytes, sha256.New)

	// 2. Create AES-GCM cipher.
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// 3. Extract nonce and ciphertext.
	nonceSize := gcm.NonceSize()
	if len(data) < saltBytes+nonceSize {
		return nil, fmt.Errorf("encrypted data is too short")
	}
	nonce, ciphertext := data[saltBytes:saltBytes+nonceSize], data[saltBytes+nonceSize:]

	// 4. Decrypt data.
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed (check password): %w", err)
	}

	return plaintext, nil
}

// GenerateAndEncryptKey creates a new secp256k1 private key and writes it encrypted to disk.
func GenerateAndEncryptKey(path string, password []byte) (*btcec.PrivateKey, error) {
	key, err := btcec.NewPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	encrypted, err := Encrypt(key.Serialize(), password)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt private key: %w", err)
	}

	if err := os.WriteFile(path, encrypted, 0600); err != nil {
		return nil, fmt.Errorf("failed to write encrypted key file: %w", err)
	}

	return key, nil
}

// LoadAndDecryptKey reads an encrypted key file and returns the decrypted secp256k1 private key.
func LoadAndDecryptKey(path string, password []byte) (*btcec.PrivateKey, error) {
	encrypted, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	decrypted, err := Decrypt(encrypted, password)
	if err != nil {
		return nil, err
	}
	if len(decrypted) != btcec.PrivKeyBytesLen {
		return nil, fmt.Errorf("invalid decrypted key length: got %d bytes", len(decrypted))
	}

	key, keyErr := btcec.PrivKeyFromBytes(decrypted)
	if key == nil || keyErr != nil {
		return nil, fmt.Errorf("invalid private key bytes: decryption may have yielded invalid data")
	}
	return key, nil
}
