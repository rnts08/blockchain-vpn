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

func TestEncodeDecodePayloadV3(t *testing.T) {
	t.Parallel()
	endpoint := &VPNEndpoint{
		IP:                    net.ParseIP("198.51.100.42"),
		Port:                  51820,
		Price:                 2000,
		PublicKey:             mustTestPubKey(t),
		AdvertisedBandwidthKB: 100000,
		MaxConsumers:          50,
		CountryCode:           "DE",
		AvailabilityFlags:     AvailabilityFlagAvailable,
		ThroughputProbePort:   51821,
		PricingMethod:         PricingMethodTime,
		TimeUnitSecs:          60,
		DataUnitBytes:         1_000_000,
		SessionTimeoutSecs:    3600,
	}
	encoded, err := endpoint.EncodePayloadV3()
	if err != nil {
		t.Fatalf("EncodePayloadV3() error = %v", err)
	}
	decoded, err := DecodePayloadV3(encoded)
	if err != nil {
		t.Fatalf("DecodePayloadV3() error = %v", err)
	}
	if !endpoint.IP.Equal(decoded.IP) {
		t.Errorf("IP mismatch: got %v want %v", decoded.IP, endpoint.IP)
	}
	if decoded.Port != endpoint.Port || decoded.Price != endpoint.Price {
		t.Errorf("port/price mismatch: got %d/%d want %d/%d", decoded.Port, decoded.Price, endpoint.Port, endpoint.Price)
	}
	if decoded.AdvertisedBandwidthKB != endpoint.AdvertisedBandwidthKB {
		t.Errorf("bandwidth mismatch: got %d want %d", decoded.AdvertisedBandwidthKB, endpoint.AdvertisedBandwidthKB)
	}
	if decoded.MaxConsumers != endpoint.MaxConsumers {
		t.Errorf("max consumers mismatch: got %d want %d", decoded.MaxConsumers, endpoint.MaxConsumers)
	}
	if decoded.CountryCode != "DE" {
		t.Errorf("country mismatch: got %q want DE", decoded.CountryCode)
	}
	if decoded.PricingMethod != endpoint.PricingMethod {
		t.Errorf("pricing method mismatch: got %d want %d", decoded.PricingMethod, endpoint.PricingMethod)
	}
	if decoded.TimeUnitSecs != endpoint.TimeUnitSecs {
		t.Errorf("time unit mismatch: got %d want %d", decoded.TimeUnitSecs, endpoint.TimeUnitSecs)
	}
	if decoded.DataUnitBytes != endpoint.DataUnitBytes {
		t.Errorf("data unit mismatch: got %d want %d", decoded.DataUnitBytes, endpoint.DataUnitBytes)
	}
	if decoded.SessionTimeoutSecs != endpoint.SessionTimeoutSecs {
		t.Errorf("session timeout mismatch: got %d want %d", decoded.SessionTimeoutSecs, endpoint.SessionTimeoutSecs)
	}
	if decoded.ThroughputProbePort != endpoint.ThroughputProbePort {
		t.Errorf("probe port mismatch: got %d want %d", decoded.ThroughputProbePort, endpoint.ThroughputProbePort)
	}
	if !decoded.PublicKey.IsEqual(endpoint.PublicKey) {
		t.Errorf("pubkey mismatch")
	}
}

func TestEncodeDecodePayloadV3_DataPricing(t *testing.T) {
	t.Parallel()
	endpoint := &VPNEndpoint{
		IP:                 net.ParseIP("10.0.0.1"),
		Port:               51820,
		Price:              500,
		PublicKey:          mustTestPubKey(t),
		PricingMethod:      PricingMethodData,
		DataUnitBytes:      100_000_000,
		SessionTimeoutSecs: 7200,
	}
	encoded, err := endpoint.EncodePayloadV3()
	if err != nil {
		t.Fatalf("EncodePayloadV3() error = %v", err)
	}
	decoded, err := DecodePayloadV3(encoded)
	if err != nil {
		t.Fatalf("DecodePayloadV3() error = %v", err)
	}
	if decoded.PricingMethod != PricingMethodData {
		t.Errorf("pricing method mismatch")
	}
	if decoded.DataUnitBytes != endpoint.DataUnitBytes {
		t.Errorf("data unit mismatch")
	}
}

func TestDecodePayloadV3_Errors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{"invalid magic", []byte{0x00, 0x00, 0x00, 0x00}, true},
		{"short payload", []byte{0x56, 0x50, 0x4E, 0x03}, true},
		{"unknown IP type", func() []byte {
			b := []byte{0x56, 0x50, 0x4E, 0x03, 0x99}
			b = append(b, make([]byte, 90)...)
			return b
		}(), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodePayloadV3(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodePayloadV3() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDecodePayloadV2_Errors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{"invalid magic", []byte{0x00, 0x00, 0x00, 0x00}, true},
		{"short payload", []byte{0x56, 0x50, 0x4E, 0x02}, true},
		{"unknown IP type", func() []byte {
			b := []byte{0x56, 0x50, 0x4E, 0x02, 0x99}
			b = append(b, make([]byte, 64)...)
			return b
		}(), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodePayloadV2(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodePayloadV2() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEncodeDecodePaymentPayload(t *testing.T) {
	t.Parallel()
	pub := mustTestPubKey(t)
	encoded, err := EncodePaymentPayload(pub)
	if err != nil {
		t.Fatalf("EncodePaymentPayload() error = %v", err)
	}
	decoded, err := DecodePaymentPayload(encoded)
	if err != nil {
		t.Fatalf("DecodePaymentPayload() error = %v", err)
	}
	if !decoded.IsEqual(pub) {
		t.Errorf("pubkey mismatch")
	}
}

func TestEncodePaymentPayload_NilPubKey(t *testing.T) {
	_, err := EncodePaymentPayload(nil)
	if err == nil {
		t.Error("expected error for nil pubkey")
	}
}

func TestDecodePaymentPayload_Errors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{"invalid magic", []byte{0x00, 0x00, 0x00, 0x00}, true},
		{"too short", []byte{0x50, 0x41, 0x59, 0x01, 0x00}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodePaymentPayload(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodePaymentPayload() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEncodeDecodePriceUpdatePayload(t *testing.T) {
	t.Parallel()
	pub := mustTestPubKey(t)
	encoded, err := EncodePriceUpdatePayload(pub, 5000)
	if err != nil {
		t.Fatalf("EncodePriceUpdatePayload() error = %v", err)
	}
	decoded, err := DecodePriceUpdatePayload(encoded)
	if err != nil {
		t.Fatalf("DecodePriceUpdatePayload() error = %v", err)
	}
	if decoded.NewPrice != 5000 {
		t.Errorf("price mismatch: got %d want 5000", decoded.NewPrice)
	}
	if !decoded.PublicKey.IsEqual(pub) {
		t.Errorf("pubkey mismatch")
	}
}

func TestEncodePriceUpdatePayload_NilPubKey(t *testing.T) {
	_, err := EncodePriceUpdatePayload(nil, 1000)
	if err == nil {
		t.Error("expected error for nil pubkey")
	}
}

func TestDecodePriceUpdatePayload_Errors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{"invalid magic", []byte{0x00, 0x00, 0x00, 0x00}, true},
		{"short payload", []byte{0x50, 0x52, 0x49, 0x43}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodePriceUpdatePayload(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodePriceUpdatePayload() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDecodeHeartbeatPayload_Errors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{"invalid magic", []byte{0x00, 0x00, 0x00, 0x00}, true},
		{"too short", []byte{0x56, 0x48, 0x42, 0x54}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeHeartbeatPayload(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeHeartbeatPayload() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEncodeHeartbeatPayload_NilPubKey(t *testing.T) {
	_, err := EncodeHeartbeatPayload(nil, 0)
	if err == nil {
		t.Error("expected error for nil pubkey")
	}
}

func TestEncodeHeartbeatPayload_Flags(t *testing.T) {
	t.Parallel()
	pub := mustTestPubKey(t)
	for _, flags := range []uint8{0x00, 0x01, 0x02, 0xFF} {
		encoded, err := EncodeHeartbeatPayload(pub, flags)
		if err != nil {
			t.Fatalf("EncodeHeartbeatPayload(flags=%d) error = %v", flags, err)
		}
		decoded, err := DecodeHeartbeatPayload(encoded)
		if err != nil {
			t.Fatalf("DecodeHeartbeatPayload() error = %v", err)
		}
		if decoded.Flags != flags {
			t.Errorf("flags mismatch for 0x%02x: got 0x%02x", flags, decoded.Flags)
		}
	}
}

func TestNormalizeCountryCode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input    string
		expected string
	}{
		{"us", "US"},
		{"US", "US"},
		{"  de  ", "DE"},
		{"", "ZZ"},
		{"a", "ZZ"},
		{"ABC", "AB"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeCountryCode(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeCountryCode(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestComputeCertFingerprint(t *testing.T) {
	t.Parallel()
	input := []byte("test cert data")
	fp := ComputeCertFingerprint(input)
	if len(fp) != 16 {
		t.Errorf("expected 16-byte fingerprint, got %d bytes", len(fp))
	}
	fp2 := ComputeCertFingerprint([]byte("different data"))
	if len(fp2) != 16 {
		t.Errorf("expected 16-byte fingerprint, got %d bytes", len(fp2))
	}
	fp3 := ComputeCertFingerprint(input)
	if string(fp) != string(fp3) {
		t.Error("expected deterministic fingerprint")
	}
}

func TestEncodeDecodeCertFingerprintPayload(t *testing.T) {
	t.Parallel()
	pub := mustTestPubKey(t)
	fp := make([]byte, 32)
	for i := range fp {
		fp[i] = byte(i)
	}
	encoded, err := EncodeCertFingerprintPayload(pub, fp)
	if err != nil {
		t.Fatalf("EncodeCertFingerprintPayload() error = %v", err)
	}
	decoded, err := DecodeCertFingerprintPayload(encoded)
	if err != nil {
		t.Fatalf("DecodeCertFingerprintPayload() error = %v", err)
	}
	if !decoded.PublicKey.IsEqual(pub) {
		t.Errorf("pubkey mismatch")
	}
	if len(decoded.CertFingerprint) != 16 {
		t.Errorf("fingerprint length mismatch: got %d want 16", len(decoded.CertFingerprint))
	}
}

func TestEncodeCertFingerprintPayload_NilPubKey(t *testing.T) {
	_, err := EncodeCertFingerprintPayload(nil, make([]byte, 32))
	if err == nil {
		t.Error("expected error for nil pubkey")
	}
}

func TestEncodeCertFingerprintPayload_ShortFingerprint(t *testing.T) {
	t.Parallel()
	pub := mustTestPubKey(t)
	_, err := EncodeCertFingerprintPayload(pub, []byte{0x01, 0x02})
	if err == nil {
		t.Error("expected error for short fingerprint")
	}
}

func TestDecodeCertFingerprintPayload_Errors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{"invalid magic", []byte{0x00, 0x00, 0x00, 0x00}, true},
		{"too short", []byte{0x43, 0x45, 0x52, 0x54}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeCertFingerprintPayload(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeCertFingerprintPayload() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEncodePayload_NilPubKey(t *testing.T) {
	ep := &VPNEndpoint{IP: net.ParseIP("1.2.3.4"), Port: 51820, Price: 1000}
	_, err := ep.EncodePayload()
	if err == nil {
		t.Error("expected error for nil pubkey")
	}
}

func TestEncodePayloadV2_NilPubKey(t *testing.T) {
	ep := &VPNEndpoint{IP: net.ParseIP("1.2.3.4"), Port: 51820, Price: 1000}
	_, err := ep.EncodePayloadV2()
	if err == nil {
		t.Error("expected error for nil pubkey")
	}
}

func TestEncodePayloadV3_NilPubKey(t *testing.T) {
	ep := &VPNEndpoint{IP: net.ParseIP("1.2.3.4"), Port: 51820, Price: 1000}
	_, err := ep.EncodePayloadV3()
	if err == nil {
		t.Error("expected error for nil pubkey")
	}
}

func TestEncodePayloadV2_InvalidIP(t *testing.T) {
	ep := &VPNEndpoint{IP: net.IP{0xFF}, Port: 51820, Price: 1000, PublicKey: mustTestPubKey(t)}
	_, err := ep.EncodePayloadV2()
	if err == nil {
		t.Error("expected error for invalid IP")
	}
}

func TestEncodePayloadV3_InvalidIP(t *testing.T) {
	ep := &VPNEndpoint{IP: net.IP{0xFF}, Port: 51820, Price: 1000, PublicKey: mustTestPubKey(t)}
	_, err := ep.EncodePayloadV3()
	if err == nil {
		t.Error("expected error for invalid IP")
	}
}
