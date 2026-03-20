package tunnel

import "testing"

func TestResolveTLSPolicyDefaults(t *testing.T) {
	p, err := ResolveTLSPolicy("", "", nil)
	if err != nil {
		t.Fatalf("resolve default policy: %v", err)
	}
	if p.MinVersionLabel != "1.3" {
		t.Fatalf("expected default min version 1.3, got %s", p.MinVersionLabel)
	}
	if p.Profile != "modern" {
		t.Fatalf("expected default profile modern, got %s", p.Profile)
	}
	if len(p.CipherNames) == 0 {
		t.Fatal("expected cipher profile names")
	}
}

func TestResolveTLSPolicyCompat(t *testing.T) {
	p, err := ResolveTLSPolicy("", "compat", nil)
	if err != nil {
		t.Fatalf("resolve compat policy: %v", err)
	}
	if p.MinVersionLabel != "1.2" {
		t.Fatalf("expected compat min version 1.2, got %s", p.MinVersionLabel)
	}
	if len(p.CipherSuites) == 0 {
		t.Fatal("expected explicit TLS 1.2 cipher suites in compat profile")
	}
}

func TestResolveTLSPolicyInvalid(t *testing.T) {
	if _, err := ResolveTLSPolicy("1.0", "modern", nil); err == nil {
		t.Fatal("expected invalid min version error")
	}
	if _, err := ResolveTLSPolicy("1.3", "legacy", nil); err == nil {
		t.Fatal("expected invalid profile error")
	}
}

func TestResolveTLSPolicyCustomCiphers(t *testing.T) {
	p, err := ResolveTLSPolicy("1.2", "compat", []string{"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256", "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384"})
	if err != nil {
		t.Fatalf("resolve custom cipher policy: %v", err)
	}
	if len(p.CipherSuites) != 2 {
		t.Fatalf("expected 2 custom ciphers, got %d", len(p.CipherSuites))
	}
	if p.CipherNames[0] != "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256" {
		t.Fatalf("expected first cipher TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256, got %s", p.CipherNames[0])
	}
}

func TestResolveTLSPolicyInvalidCipher(t *testing.T) {
	if _, err := ResolveTLSPolicy("1.2", "compat", []string{"INVALID-CIPHER-SUITE"}); err == nil {
		t.Fatal("expected invalid cipher error")
	}
}

func TestResolveTLSPolicyModernExplicit(t *testing.T) {
	p, err := ResolveTLSPolicy("1.3", "modern", nil)
	if err != nil {
		t.Fatalf("resolve modern policy: %v", err)
	}
	if p.MinVersionLabel != "1.3" {
		t.Errorf("expected min version 1.3, got %s", p.MinVersionLabel)
	}
	if p.Profile != "modern" {
		t.Errorf("expected profile modern, got %s", p.Profile)
	}
	if len(p.CipherSuites) != 0 {
		t.Errorf("expected no explicit cipher suites for modern profile (tls13), got %d", len(p.CipherSuites))
	}
}

func TestResolveTLSPolicyWhitespaceTrimming(t *testing.T) {
	p, err := ResolveTLSPolicy("  1.2  ", "  compat  ", nil)
	if err != nil {
		t.Fatalf("resolve policy with whitespace: %v", err)
	}
	if p.MinVersionLabel != "1.2" {
		t.Errorf("expected min version 1.2, got %s", p.MinVersionLabel)
	}
	if p.Profile != "compat" {
		t.Errorf("expected profile compat, got %s", p.Profile)
	}
}
