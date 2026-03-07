package tunnel

import "testing"

func TestResolveTLSPolicyDefaults(t *testing.T) {
	p, err := ResolveTLSPolicy("", "")
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
	p, err := ResolveTLSPolicy("", "compat")
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
	if _, err := ResolveTLSPolicy("1.0", "modern"); err == nil {
		t.Fatal("expected invalid min version error")
	}
	if _, err := ResolveTLSPolicy("1.3", "legacy"); err == nil {
		t.Fatal("expected invalid profile error")
	}
}
