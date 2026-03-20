package blockchain

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"math/big"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
)

func TestSignWithSecp256k1(t *testing.T) {
	t.Parallel()
	priv, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	message := []byte("test message for signing")
	sig, err := signWithSecp256k1(priv, message)
	if err != nil {
		t.Fatalf("signWithSecp256k1() error = %v", err)
	}
	if len(sig) == 0 {
		t.Error("expected non-empty signature")
	}
}

func TestSignWithSecp256k1_Randomized(t *testing.T) {
	t.Parallel()
	priv, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	message := []byte("randomized message")
	sig1, err := signWithSecp256k1(priv, message)
	if err != nil {
		t.Fatalf("signWithSecp256k1() error = %v", err)
	}
	sig2, err := signWithSecp256k1(priv, message)
	if err != nil {
		t.Fatalf("signWithSecp256k1() error = %v", err)
	}
	if len(sig1) == 0 || len(sig2) == 0 {
		t.Error("signatures should be non-empty")
	}
	// Check that signatures are different (due to randomization)
	if string(sig1) == string(sig2) {
		t.Error("signatures should differ due to randomized nonces (ECDSA)")
	}
	// Convert to ecdsa.PrivateKey for verification
	ecdsaPriv := &ecdsa.PrivateKey{
		D: new(big.Int).SetBytes(priv.Serialize()),
	}
	ecdsaPriv.PublicKey.Curve = elliptic.P256()
	ecdsaPriv.PublicKey.X, ecdsaPriv.PublicKey.Y = elliptic.P256().ScalarBaseMult(ecdsaPriv.D.Bytes())

	hash := sha256.Sum256(message)

	// Verify both signatures are valid using ASN.1 verification
	if !verifyASN1Signature(&ecdsaPriv.PublicKey, hash[:], sig1) {
		t.Error("sig1 failed verification")
	}
	if !verifyASN1Signature(&ecdsaPriv.PublicKey, hash[:], sig2) {
		t.Error("sig2 failed verification")
	}
}

func TestSignWithSecp256k1_DifferentMessages(t *testing.T) {
	t.Parallel()
	priv, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	sig1, _ := signWithSecp256k1(priv, []byte("message 1"))
	sig2, _ := signWithSecp256k1(priv, []byte("message 2"))
	if string(sig1) == string(sig2) {
		t.Error("signatures for different messages should differ")
	}
}

func TestSignWithSecp256k1_DifferentKeys(t *testing.T) {
	t.Parallel()
	priv1, _ := btcec.NewPrivateKey()
	priv2, _ := btcec.NewPrivateKey()
	message := []byte("same message")
	sig1, _ := signWithSecp256k1(priv1, message)
	sig2, _ := signWithSecp256k1(priv2, message)
	if string(sig1) == string(sig2) {
		t.Error("signatures from different keys should differ")
	}
}
