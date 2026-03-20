//go:build functional

package blockchain

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"math/big"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
)

func TestFunctional_HeartbeatSignatureVerification(t *testing.T) {
	t.Parallel()

	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("NewPrivateKey failed: %v", err)
	}

	message := []byte("heartbeat test message")
	sig, err := signWithSecp256k1(privKey, message)
	if err != nil {
		t.Fatalf("signWithSecp256k1 failed: %v", err)
	}

	ecdsaPriv := &ecdsa.PrivateKey{
		D: new(big.Int).SetBytes(privKey.Serialize()),
	}
	ecdsaPriv.PublicKey.Curve = elliptic.P256()
	ecdsaPriv.PublicKey.X, ecdsaPriv.PublicKey.Y = elliptic.P256().ScalarBaseMult(ecdsaPriv.D.Bytes())

	hash := sha256.Sum256(message)
	if !verifyASN1Signature(&ecdsaPriv.PublicKey, hash[:], sig) {
		t.Error("Signature verification failed")
	}

	t.Logf("Heartbeat signature verification works correctly")
}

func TestFunctional_HeartbeatSignatureDifferentPubKeyFails(t *testing.T) {
	t.Parallel()

	privKey1, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("NewPrivateKey failed: %v", err)
	}

	privKey2, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("NewPrivateKey failed: %v", err)
	}

	message := []byte("heartbeat test message")
	sig, err := signWithSecp256k1(privKey1, message)
	if err != nil {
		t.Fatalf("signWithSecp256k1 failed: %v", err)
	}

	ecdsaPriv2 := &ecdsa.PrivateKey{
		D: new(big.Int).SetBytes(privKey2.Serialize()),
	}
	ecdsaPriv2.PublicKey.Curve = elliptic.P256()
	ecdsaPriv2.PublicKey.X, ecdsaPriv2.PublicKey.Y = elliptic.P256().ScalarBaseMult(ecdsaPriv2.D.Bytes())

	hash := sha256.Sum256(message)
	if verifyASN1Signature(&ecdsaPriv2.PublicKey, hash[:], sig) {
		t.Error("Signature should not verify with different public key")
	}

	t.Log("Heartbeat signature correctly fails verification with wrong public key")
}

func TestFunctional_HeartbeatSignatureTamperedFails(t *testing.T) {
	t.Parallel()

	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("NewPrivateKey failed: %v", err)
	}

	message := []byte("heartbeat test message")
	sig, err := signWithSecp256k1(privKey, message)
	if err != nil {
		t.Fatalf("signWithSecp256k1 failed: %v", err)
	}

	sig[0] ^= 0xFF

	ecdsaPriv := &ecdsa.PrivateKey{
		D: new(big.Int).SetBytes(privKey.Serialize()),
	}
	ecdsaPriv.PublicKey.Curve = elliptic.P256()
	ecdsaPriv.PublicKey.X, ecdsaPriv.PublicKey.Y = elliptic.P256().ScalarBaseMult(ecdsaPriv.D.Bytes())

	hash := sha256.Sum256(message)
	if verifyASN1Signature(&ecdsaPriv.PublicKey, hash[:], sig) {
		t.Error("Tampered signature should not verify")
	}

	t.Log("Tampered heartbeat signature correctly fails verification")
}

func TestFunctional_HeartbeatSignatureEmptyMessageFails(t *testing.T) {
	t.Parallel()

	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("NewPrivateKey failed: %v", err)
	}

	message := []byte{}
	sig, err := signWithSecp256k1(privKey, message)
	if err != nil {
		t.Fatalf("signWithSecp256k1 failed: %v", err)
	}

	ecdsaPriv := &ecdsa.PrivateKey{
		D: new(big.Int).SetBytes(privKey.Serialize()),
	}
	ecdsaPriv.PublicKey.Curve = elliptic.P256()
	ecdsaPriv.PublicKey.X, ecdsaPriv.PublicKey.Y = elliptic.P256().ScalarBaseMult(ecdsaPriv.D.Bytes())

	hash := sha256.Sum256(message)
	if !verifyASN1Signature(&ecdsaPriv.PublicKey, hash[:], sig) {
		t.Error("Signature should verify for empty message")
	}

	t.Log("Heartbeat signature verification works for empty message")
}

func TestFunctional_VerifyASN1SignatureInvalidASN1(t *testing.T) {
	t.Parallel()

	pubKey := &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int),
		Y:     new(big.Int),
	}

	hash := sha256.Sum256([]byte("test"))

	invalidSig := []byte("not valid ASN.1")
	if verifyASN1Signature(pubKey, hash[:], invalidSig) {
		t.Error("Invalid ASN.1 signature should not verify")
	}

	t.Log("Invalid ASN.1 signature correctly fails verification")
}
