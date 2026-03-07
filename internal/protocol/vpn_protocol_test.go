package protocol

import (
	"encoding/binary"
	"net"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
)

func TestEncodeDecodePayload(t *testing.T) {
	tests := []struct {
		name     string
		endpoint *VPNEndpoint
	}{
		{
			name: "IPv4 Address",
			endpoint: &VPNEndpoint{
				IP:        net.ParseIP("192.168.1.1"),
				Port:      8080,
				Price:     1000,
				PublicKey: mustTestPubKey(t),
			},
		},
		{
			name: "IPv6 Address",
			endpoint: &VPNEndpoint{
				IP:        net.ParseIP("2001:0db8:85a3:0000:0000:8a2e:0370:7334"),
				Port:      51820,
				Price:     500,
				PublicKey: mustTestPubKey(t),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := tt.endpoint.EncodePayload()
			if err != nil {
				t.Fatalf("EncodePayload() error = %v", err)
			}

			decoded, err := DecodePayload(encoded)
			if err != nil {
				t.Fatalf("DecodePayload() error = %v", err)
			}

			if !tt.endpoint.IP.Equal(decoded.IP) {
				t.Errorf("IP mismatch: got %v, want %v", decoded.IP, tt.endpoint.IP)
			}
			if tt.endpoint.Port != decoded.Port {
				t.Errorf("Port mismatch: got %v, want %v", decoded.Port, tt.endpoint.Port)
			}
			if tt.endpoint.Price != decoded.Price {
				t.Errorf("Price mismatch: got %v, want %v", decoded.Price, tt.endpoint.Price)
			}
			if !tt.endpoint.PublicKey.IsEqual(decoded.PublicKey) {
				t.Errorf("PublicKey mismatch: got %x, want %x", decoded.PublicKey.SerializeCompressed(), tt.endpoint.PublicKey.SerializeCompressed())
			}
		})
	}
}

func TestDecodePayload_Errors(t *testing.T) {
	// Test Invalid Magic Bytes
	invalidMagic := []byte{0x00, 0x00, 0x00, 0x00}
	if _, err := DecodePayload(invalidMagic); err == nil {
		t.Error("DecodePayload() expected error for invalid magic bytes, got nil")
	}

	// Test Short Payload (Magic only)
	shortPayload := make([]byte, 4)
	binary.BigEndian.PutUint32(shortPayload, MagicBytes)
	if _, err := DecodePayload(shortPayload); err == nil {
		t.Error("DecodePayload() expected error for short payload, got nil")
	}

	// Test Unknown IP Type
	unknownIPType := make([]byte, 5)
	binary.BigEndian.PutUint32(unknownIPType, MagicBytes)
	unknownIPType[4] = 0x99 // Unknown type
	if _, err := DecodePayload(unknownIPType); err == nil {
		t.Error("DecodePayload() expected error for unknown IP type, got nil")
	}
}

func TestEncodeDecodePayloadV2(t *testing.T) {
	endpoint := &VPNEndpoint{
		IP:                    net.ParseIP("203.0.113.10"),
		Port:                  51820,
		Price:                 1200,
		PublicKey:             mustTestPubKey(t),
		AdvertisedBandwidthKB: 50000,
		MaxConsumers:          42,
		CountryCode:           "us",
		AvailabilityFlags:     AvailabilityFlagAvailable,
	}
	encoded, err := endpoint.EncodePayloadV2()
	if err != nil {
		t.Fatalf("EncodePayloadV2() error = %v", err)
	}
	decoded, err := DecodePayloadV2(encoded)
	if err != nil {
		t.Fatalf("DecodePayloadV2() error = %v", err)
	}
	if !endpoint.IP.Equal(decoded.IP) {
		t.Fatalf("IP mismatch: got %v want %v", decoded.IP, endpoint.IP)
	}
	if decoded.Port != endpoint.Port || decoded.Price != endpoint.Price {
		t.Fatalf("port/price mismatch: got %d/%d want %d/%d", decoded.Port, decoded.Price, endpoint.Port, endpoint.Price)
	}
	if decoded.AdvertisedBandwidthKB != endpoint.AdvertisedBandwidthKB {
		t.Fatalf("bandwidth mismatch: got %d want %d", decoded.AdvertisedBandwidthKB, endpoint.AdvertisedBandwidthKB)
	}
	if decoded.MaxConsumers != endpoint.MaxConsumers {
		t.Fatalf("max consumers mismatch: got %d want %d", decoded.MaxConsumers, endpoint.MaxConsumers)
	}
	if decoded.CountryCode != "US" {
		t.Fatalf("country normalization mismatch: got %q want %q", decoded.CountryCode, "US")
	}
	if decoded.AvailabilityFlags != AvailabilityFlagAvailable {
		t.Fatalf("availability mismatch: got %d", decoded.AvailabilityFlags)
	}
	if !decoded.PublicKey.IsEqual(endpoint.PublicKey) {
		t.Fatalf("pubkey mismatch")
	}
}

func TestEncodeDecodeHeartbeatPayload(t *testing.T) {
	pub := mustTestPubKey(t)
	encoded, err := EncodeHeartbeatPayload(pub, AvailabilityFlagAvailable)
	if err != nil {
		t.Fatalf("EncodeHeartbeatPayload() error = %v", err)
	}
	decoded, err := DecodeHeartbeatPayload(encoded)
	if err != nil {
		t.Fatalf("DecodeHeartbeatPayload() error = %v", err)
	}
	if !decoded.PublicKey.IsEqual(pub) {
		t.Fatalf("pubkey mismatch")
	}
	if decoded.Flags != AvailabilityFlagAvailable {
		t.Fatalf("flags mismatch: got %d", decoded.Flags)
	}
}

func mustTestPubKey(t *testing.T) *btcec.PublicKey {
	t.Helper()
	priv, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("failed to create test private key: %v", err)
	}
	return priv.PubKey()
}
