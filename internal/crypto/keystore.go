package crypto

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
)

var errKeyNotFound = errors.New("provider key not found")

func ResolveKeyStorageMode(mode string) (string, error) {
	m := strings.ToLower(strings.TrimSpace(mode))
	if m == "" || m == "file" {
		return "file", nil
	}
	if m == "auto" {
		switch runtime.GOOS {
		case "darwin":
			if commandExists("security") {
				return "keychain", nil
			}
		case "linux":
			if commandExists("secret-tool") {
				return "libsecret", nil
			}
		case "windows":
			if commandExists("powershell") || commandExists("pwsh") {
				return "dpapi", nil
			}
		}
		return "file", nil
	}
	switch m {
	case "keychain", "libsecret", "dpapi":
		return m, nil
	default:
		return "", fmt.Errorf("unknown key storage mode %q", mode)
	}
}

func SupportsKeyStorageMode(mode string) bool {
	m, err := ResolveKeyStorageMode(mode)
	if err != nil {
		return false
	}
	if m == "file" {
		return true
	}
	switch m {
	case "keychain":
		return runtime.GOOS == "darwin" && commandExists("security")
	case "libsecret":
		return runtime.GOOS == "linux" && commandExists("secret-tool")
	case "dpapi":
		return runtime.GOOS == "windows" && (commandExists("powershell") || commandExists("pwsh"))
	default:
		return false
	}
}

func KeyStorageStatus(mode string) (resolved string, supported bool, detail string) {
	resolved, err := ResolveKeyStorageMode(mode)
	if err != nil {
		return "", false, err.Error()
	}
	if resolved == "file" {
		return resolved, true, "file mode available"
	}
	if SupportsKeyStorageMode(resolved) {
		return resolved, true, resolved + " backend available"
	}
	return resolved, false, resolved + " backend unavailable on this host"
}

func LoadOrCreateProviderKey(path string, password []byte, mode, service string) (*btcec.PrivateKey, error) {
	requestedMode := strings.ToLower(strings.TrimSpace(mode))
	resolved, err := ResolveKeyStorageMode(mode)
	if err != nil {
		return nil, err
	}
	if resolved != "file" && !SupportsKeyStorageMode(resolved) {
		if requestedMode == "auto" {
			resolved = "file"
		} else {
			return nil, fmt.Errorf("key storage backend %q is unavailable; use mode=auto or file", resolved)
		}
	}
	if resolved == "file" {
		if len(password) == 0 {
			return nil, fmt.Errorf("provider key password cannot be empty in file mode")
		}
		if _, statErr := os.Stat(path); statErr == nil {
			return LoadAndDecryptKey(path, password)
		}
		return GenerateAndEncryptKey(path, password)
	}
	if service == "" {
		service = "BlockchainVPN"
	}
	account, dpapiPath, err := keyStoreIDs(path)
	if err != nil {
		return nil, err
	}
	key, err := loadProviderKeyFromSecureStore(resolved, service, account, dpapiPath)
	if err == nil {
		return key, nil
	}
	if requestedMode == "auto" && len(password) > 0 {
		return LoadOrCreateProviderKey(path, password, "file", service)
	}
	if !errors.Is(err, errKeyNotFound) {
		return nil, err
	}
	key, err = btcec.NewPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate provider key: %w", err)
	}
	if err := saveProviderKeyToSecureStore(resolved, service, account, dpapiPath, key); err != nil {
		if requestedMode == "auto" && len(password) > 0 {
			return LoadOrCreateProviderKey(path, password, "file", service)
		}
		return nil, err
	}
	return key, nil
}

func RotateProviderKey(path string, oldPassword, newPassword []byte, mode, service string) error {
	requestedMode := strings.ToLower(strings.TrimSpace(mode))
	resolved, err := ResolveKeyStorageMode(mode)
	if err != nil {
		return err
	}
	if resolved != "file" && !SupportsKeyStorageMode(resolved) {
		if requestedMode == "auto" {
			resolved = "file"
		} else {
			return fmt.Errorf("key storage backend %q is unavailable; use mode=auto or file", resolved)
		}
	}
	if resolved == "file" {
		if len(oldPassword) == 0 || len(newPassword) == 0 {
			return fmt.Errorf("old/new provider key passwords are required in file mode")
		}
		if _, err := LoadAndDecryptKey(path, oldPassword); err != nil {
			return fmt.Errorf("failed to decrypt existing key with provided password: %w", err)
		}
		backupPath := fmt.Sprintf("%s.bak-%s", path, nowUTCCompact())
		if err := os.Rename(path, backupPath); err != nil {
			return fmt.Errorf("failed to create backup before rotation: %w", err)
		}
		if _, err := GenerateAndEncryptKey(path, newPassword); err != nil {
			_ = os.Rename(backupPath, path)
			return fmt.Errorf("failed to write new rotated key (restored backup): %w", err)
		}
		return nil
	}
	if service == "" {
		service = "BlockchainVPN"
	}
	account, dpapiPath, err := keyStoreIDs(path)
	if err != nil {
		return err
	}
	if _, err := loadProviderKeyFromSecureStore(resolved, service, account, dpapiPath); err != nil {
		if errors.Is(err, errKeyNotFound) {
			return fmt.Errorf("provider key does not exist in secure store")
		}
		return err
	}
	key, err := btcec.NewPrivateKey()
	if err != nil {
		return fmt.Errorf("failed to generate rotated provider key: %w", err)
	}
	if requestedMode == "auto" && len(newPassword) > 0 {
		if err := saveProviderKeyToSecureStore(resolved, service, account, dpapiPath, key); err != nil {
			return RotateProviderKey(path, oldPassword, newPassword, "file", service)
		}
		return nil
	}
	return saveProviderKeyToSecureStore(resolved, service, account, dpapiPath, key)
}

func nowUTCCompact() string {
	return time.Now().UTC().Format("20060102-150405")
}

func keyStoreIDs(path string) (account string, dpapiPath string, err error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", "", err
	}
	encoded := hex.EncodeToString([]byte(absPath))
	return "provider-key:" + encoded, absPath + ".dpapi", nil
}

func loadProviderKeyFromSecureStore(mode, service, account, dpapiPath string) (*btcec.PrivateKey, error) {
	raw, err := loadSecret(mode, service, account, dpapiPath)
	if err != nil {
		return nil, err
	}
	b, err := hex.DecodeString(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("invalid secure-store key encoding: %w", err)
	}
	if len(b) != btcec.PrivKeyBytesLen {
		return nil, fmt.Errorf("invalid provider key length from secure store: %d", len(b))
	}
	key, _ := btcec.PrivKeyFromBytes(b)
	return key, nil
}

func saveProviderKeyToSecureStore(mode, service, account, dpapiPath string, key *btcec.PrivateKey) error {
	if key == nil {
		return fmt.Errorf("nil provider key")
	}
	return saveSecret(mode, service, account, dpapiPath, hex.EncodeToString(key.Serialize()))
}

func loadSecret(mode, service, account, dpapiPath string) (string, error) {
	switch mode {
	case "keychain":
		return loadSecretKeychain(service, account)
	case "libsecret":
		return loadSecretLibsecret(service, account)
	case "dpapi":
		return loadSecretDPAPI(dpapiPath)
	default:
		return "", fmt.Errorf("unsupported secure-store mode %q", mode)
	}
}

func saveSecret(mode, service, account, dpapiPath, secret string) error {
	switch mode {
	case "keychain":
		return saveSecretKeychain(service, account, secret)
	case "libsecret":
		return saveSecretLibsecret(service, account, secret)
	case "dpapi":
		return saveSecretDPAPI(dpapiPath, secret)
	default:
		return fmt.Errorf("unsupported secure-store mode %q", mode)
	}
}

var commandLookup = func(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func commandExists(name string) bool {
	return commandLookup(name)
}

func loadSecretKeychain(service, account string) (string, error) {
	out, err := exec.Command("security", "find-generic-password", "-s", service, "-a", account, "-w").CombinedOutput()
	if err != nil {
		msg := strings.ToLower(string(out))
		if strings.Contains(msg, "could not be found") {
			return "", errKeyNotFound
		}
		return "", fmt.Errorf("keychain lookup failed: %s", strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func saveSecretKeychain(service, account, secret string) error {
	out, err := exec.Command("security", "add-generic-password", "-U", "-s", service, "-a", account, "-w", secret).CombinedOutput()
	if err != nil {
		return fmt.Errorf("keychain store failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func loadSecretLibsecret(service, account string) (string, error) {
	out, err := exec.Command("secret-tool", "lookup", "service", service, "account", account).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("libsecret lookup failed: %s", strings.TrimSpace(string(out)))
	}
	val := strings.TrimSpace(string(out))
	if val == "" {
		return "", errKeyNotFound
	}
	return val, nil
}

func saveSecretLibsecret(service, account, secret string) error {
	cmd := exec.Command("secret-tool", "store", "--label=BlockchainVPN Provider Key", "service", service, "account", account)
	cmd.Stdin = strings.NewReader(secret)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("libsecret store failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func loadSecretDPAPI(dpapiPath string) (string, error) {
	b, err := os.ReadFile(dpapiPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", errKeyNotFound
		}
		return "", err
	}
	return unprotectDPAPI(strings.TrimSpace(string(b)))
}

func saveSecretDPAPI(dpapiPath, secret string) error {
	protected, err := protectDPAPI(secret)
	if err != nil {
		return err
	}
	return os.WriteFile(dpapiPath, []byte(protected), 0o600)
}

func protectDPAPI(secret string) (string, error) {
	ps, err := pickPowerShell()
	if err != nil {
		return "", err
	}
	script := "$b=[Text.Encoding]::UTF8.GetBytes($env:BCVPN_SECRET);$p=[Security.Cryptography.ProtectedData]::Protect($b,$null,[Security.Cryptography.DataProtectionScope]::CurrentUser);[Convert]::ToBase64String($p)"
	return runPowerShell(ps, script, map[string]string{"BCVPN_SECRET": secret})
}

func unprotectDPAPI(blob string) (string, error) {
	ps, err := pickPowerShell()
	if err != nil {
		return "", err
	}
	script := "$b=[Convert]::FromBase64String($env:BCVPN_BLOB);$p=[Security.Cryptography.ProtectedData]::Unprotect($b,$null,[Security.Cryptography.DataProtectionScope]::CurrentUser);[Text.Encoding]::UTF8.GetString($p)"
	return runPowerShell(ps, script, map[string]string{"BCVPN_BLOB": blob})
}

func pickPowerShell() (string, error) {
	if commandExists("powershell") {
		return "powershell", nil
	}
	if commandExists("pwsh") {
		return "pwsh", nil
	}
	return "", fmt.Errorf("powershell not available for dpapi mode")
}

func runPowerShell(bin, script string, env map[string]string) (string, error) {
	cmd := exec.Command(bin, "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.Env = append([]string{}, os.Environ()...)
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(out.String())
		}
		return "", fmt.Errorf("powershell execution failed: %w (%s)", err, msg)
	}
	return strings.TrimSpace(out.String()), nil
}
