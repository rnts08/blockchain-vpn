package protocol

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"net"
	"strings"

	"github.com/btcsuite/btcd/btcec/v2"
)

const MagicBytes = 0x56504E01                // "VPN" + Version 1
const MagicBytesV2 = 0x56504E02              // "VPN" + Version 2 (metadata)
const MagicBytesV3 = 0x56504E03              // "VPN" + Version 3 (flexible pricing)
const PriceUpdateMagicBytes = 0x50524943     // "PRIC"
const PaymentMagicBytes = 0x50415901         // "PAY" + Version 1
const HeartbeatMagicBytes = 0x56484254       // "VHBT"
const CertFingerprintMagicBytes = 0x43455254 // "CERT" - for certificate fingerprint announcements

// Pricing method types
const (
	PricingMethodSession = 0 // Flat fee per session
	PricingMethodTime    = 1 // Price per time unit
	PricingMethodData    = 2 // Price per data unit
)

type VPNEndpoint struct {
	IP                    net.IP
	Port                  uint16
	Price                 uint64           // Satoshis per session (v1/v2) or per billing unit (v3)
	PublicKey             *btcec.PublicKey // 33 bytes for compressed secp256k1
	AdvertisedBandwidthKB uint32           // optional metadata (v2 payload)
	MaxConsumers          uint16           // optional metadata (v2 payload), 0 = unknown/unlimited
	CountryCode           string           // optional metadata (v2 payload), ISO alpha2 upper-case
	AvailabilityFlags     uint8            // optional metadata (v2 payload), bit0=available
	ThroughputProbePort   uint16           // optional metadata (v2 payload)
	CertFingerprint       []byte           // SHA256 hash of current TLS certificate (33 bytes prefix)

	// New fields for V3 (flexible pricing)
	PricingMethod      uint8  // 0=session, 1=time, 2=data
	TimeUnitSecs       uint32 // For time-based: seconds per billing cycle (e.g., 60 for per-minute)
	DataUnitBytes      uint32 // For data-based: bytes per billing cycle (e.g., 1_000_000 for MB)
	SessionTimeoutSecs uint32 // Max session duration in seconds (0 = unlimited)
}

const AvailabilityFlagAvailable = 0x01

// EncodePayload creates the OP_RETURN data
func (v *VPNEndpoint) EncodePayload() ([]byte, error) {
	buf := new(bytes.Buffer)

	// 1. Magic Bytes
	if err := binary.Write(buf, binary.BigEndian, uint32(MagicBytes)); err != nil {
		return nil, err
	}

	// 2. IP Address
	ip4 := v.IP.To4()
	if ip4 != nil {
		// It's an IPv4 address
		buf.WriteByte(0x04)
		buf.Write(ip4)
	} else {
		// It's an IPv6 address
		ip16 := v.IP.To16()
		if ip16 == nil {
			return nil, fmt.Errorf("invalid IP address format: not IPv4 or IPv6")
		}
		buf.WriteByte(0x06)
		buf.Write(ip16)
	}

	// 3. Port
	if err := binary.Write(buf, binary.BigEndian, v.Port); err != nil {
		return nil, err
	}

	// 4. Price
	if err := binary.Write(buf, binary.BigEndian, v.Price); err != nil {
		return nil, err
	}

	// 5. Public Key
	if v.PublicKey == nil {
		return nil, fmt.Errorf("public key cannot be nil")
	}
	buf.Write(v.PublicKey.SerializeCompressed())

	return buf.Bytes(), nil
}

// EncodePayloadV2 creates the OP_RETURN data for the v2 announcement payload.
func (v *VPNEndpoint) EncodePayloadV2() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.BigEndian, uint32(MagicBytesV2)); err != nil {
		return nil, err
	}

	ip4 := v.IP.To4()
	if ip4 != nil {
		buf.WriteByte(0x04)
		buf.Write(ip4)
	} else {
		ip16 := v.IP.To16()
		if ip16 == nil {
			return nil, fmt.Errorf("invalid IP address format: not IPv4 or IPv6")
		}
		buf.WriteByte(0x06)
		buf.Write(ip16)
	}

	if err := binary.Write(buf, binary.BigEndian, v.Port); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, v.Price); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, v.AdvertisedBandwidthKB); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, v.MaxConsumers); err != nil {
		return nil, err
	}

	country := normalizeCountryCode(v.CountryCode)
	buf.WriteString(country)
	buf.WriteByte(v.AvailabilityFlags)
	if err := binary.Write(buf, binary.BigEndian, v.ThroughputProbePort); err != nil {
		return nil, err
	}

	if v.PublicKey == nil {
		return nil, fmt.Errorf("public key cannot be nil")
	}
	buf.Write(v.PublicKey.SerializeCompressed())
	return buf.Bytes(), nil
}

// DecodePayload parses the OP_RETURN data
func DecodePayload(data []byte) (*VPNEndpoint, error) {
	buf := bytes.NewReader(data)

	// 1. Check Magic
	var magic uint32
	if err := binary.Read(buf, binary.BigEndian, &magic); err != nil {
		return nil, fmt.Errorf("could not read magic bytes: %w", err)
	}
	if magic != MagicBytes {
		return nil, fmt.Errorf("invalid magic bytes")
	}

	// 2. IP Type and Address
	ipType, err := buf.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("could not read ip type: %w", err)
	}

	var ip net.IP
	switch ipType {
	case 0x04:
		ipBytes := make([]byte, 4)
		if _, err := buf.Read(ipBytes); err != nil {
			return nil, fmt.Errorf("could not read ipv4 address: %w", err)
		}
		ip = net.IP(ipBytes)
	case 0x06:
		ipBytes := make([]byte, 16)
		if _, err := buf.Read(ipBytes); err != nil {
			return nil, fmt.Errorf("could not read ipv6 address: %w", err)
		}
		ip = net.IP(ipBytes)
	default:
		return nil, fmt.Errorf("unknown ip type: %d", ipType)
	}

	// 3. Port
	var port uint16
	if err := binary.Read(buf, binary.BigEndian, &port); err != nil {
		return nil, fmt.Errorf("could not read port: %w", err)
	}

	// 4. Price
	var price uint64
	if err := binary.Read(buf, binary.BigEndian, &price); err != nil {
		return nil, fmt.Errorf("could not read price: %w", err)
	}

	// 5. PubKey
	// The remainder of the payload should be the public key.
	expectedPubKeyLen := btcec.PubKeyBytesLenCompressed
	if buf.Len() != expectedPubKeyLen {
		return nil, fmt.Errorf("incorrect remaining payload length for public key, expected %d, got %d", expectedPubKeyLen, buf.Len())
	}
	pubKeyBytes := make([]byte, expectedPubKeyLen)
	if _, err := buf.Read(pubKeyBytes); err != nil {
		return nil, fmt.Errorf("could not read public key: %w", err)
	}
	pubKey, err := btcec.ParsePubKey(pubKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("invalid public key in payload: %w", err)
	}

	return &VPNEndpoint{
		IP:        ip,
		Port:      port,
		Price:     price,
		PublicKey: pubKey,
	}, nil
}

// DecodePayloadV2 parses the OP_RETURN data from a v2 service announcement.
func DecodePayloadV2(data []byte) (*VPNEndpoint, error) {
	buf := bytes.NewReader(data)
	var magic uint32
	if err := binary.Read(buf, binary.BigEndian, &magic); err != nil {
		return nil, fmt.Errorf("could not read magic bytes: %w", err)
	}
	if magic != MagicBytesV2 {
		return nil, fmt.Errorf("invalid v2 magic bytes")
	}

	ipType, err := buf.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("could not read ip type: %w", err)
	}
	var ip net.IP
	switch ipType {
	case 0x04:
		ipBytes := make([]byte, 4)
		if _, err := buf.Read(ipBytes); err != nil {
			return nil, fmt.Errorf("could not read ipv4 address: %w", err)
		}
		ip = net.IP(ipBytes)
	case 0x06:
		ipBytes := make([]byte, 16)
		if _, err := buf.Read(ipBytes); err != nil {
			return nil, fmt.Errorf("could not read ipv6 address: %w", err)
		}
		ip = net.IP(ipBytes)
	default:
		return nil, fmt.Errorf("unknown ip type: %d", ipType)
	}

	var port uint16
	if err := binary.Read(buf, binary.BigEndian, &port); err != nil {
		return nil, fmt.Errorf("could not read port: %w", err)
	}
	var price uint64
	if err := binary.Read(buf, binary.BigEndian, &price); err != nil {
		return nil, fmt.Errorf("could not read price: %w", err)
	}
	var bandwidthKB uint32
	if err := binary.Read(buf, binary.BigEndian, &bandwidthKB); err != nil {
		return nil, fmt.Errorf("could not read bandwidth: %w", err)
	}
	var maxConsumers uint16
	if err := binary.Read(buf, binary.BigEndian, &maxConsumers); err != nil {
		return nil, fmt.Errorf("could not read max consumers: %w", err)
	}
	country := make([]byte, 2)
	if _, err := buf.Read(country); err != nil {
		return nil, fmt.Errorf("could not read country code: %w", err)
	}
	flags, err := buf.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("could not read availability flags: %w", err)
	}
	var probePort uint16
	if err := binary.Read(buf, binary.BigEndian, &probePort); err != nil {
		return nil, fmt.Errorf("could not read throughput probe port: %w", err)
	}
	if buf.Len() != btcec.PubKeyBytesLenCompressed {
		return nil, fmt.Errorf("incorrect remaining payload length for public key, expected %d, got %d", btcec.PubKeyBytesLenCompressed, buf.Len())
	}
	pubKeyBytes := make([]byte, btcec.PubKeyBytesLenCompressed)
	if _, err := buf.Read(pubKeyBytes); err != nil {
		return nil, fmt.Errorf("could not read public key: %w", err)
	}
	pubKey, err := btcec.ParsePubKey(pubKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("invalid public key in payload: %w", err)
	}
	return &VPNEndpoint{
		IP:                    ip,
		Port:                  port,
		Price:                 price,
		PublicKey:             pubKey,
		AdvertisedBandwidthKB: bandwidthKB,
		MaxConsumers:          maxConsumers,
		CountryCode:           normalizeCountryCode(string(country)),
		AvailabilityFlags:     flags,
		ThroughputProbePort:   probePort,
	}, nil
}

// EncodePayloadV3 creates the OP_RETURN data for the v3 announcement payload
// with flexible pricing support.
func (v *VPNEndpoint) EncodePayloadV3() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.BigEndian, uint32(MagicBytesV3)); err != nil {
		return nil, err
	}

	ip4 := v.IP.To4()
	if ip4 != nil {
		buf.WriteByte(0x04)
		buf.Write(ip4)
	} else {
		ip16 := v.IP.To16()
		if ip16 == nil {
			return nil, fmt.Errorf("invalid IP address format: not IPv4 or IPv6")
		}
		buf.WriteByte(0x06)
		buf.Write(ip16)
	}

	if err := binary.Write(buf, binary.BigEndian, v.Port); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, v.Price); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, v.AdvertisedBandwidthKB); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, v.MaxConsumers); err != nil {
		return nil, err
	}

	country := normalizeCountryCode(v.CountryCode)
	buf.WriteString(country)
	buf.WriteByte(v.AvailabilityFlags)
	if err := binary.Write(buf, binary.BigEndian, v.ThroughputProbePort); err != nil {
		return nil, err
	}

	// New V3 fields
	buf.WriteByte(v.PricingMethod)
	if err := binary.Write(buf, binary.BigEndian, v.TimeUnitSecs); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, v.DataUnitBytes); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, v.SessionTimeoutSecs); err != nil {
		return nil, err
	}

	if v.PublicKey == nil {
		return nil, fmt.Errorf("public key cannot be nil")
	}
	buf.Write(v.PublicKey.SerializeCompressed())
	return buf.Bytes(), nil
}

// DecodePayloadV3 parses the OP_RETURN data from a v3 service announcement.
func DecodePayloadV3(data []byte) (*VPNEndpoint, error) {
	buf := bytes.NewReader(data)
	var magic uint32
	if err := binary.Read(buf, binary.BigEndian, &magic); err != nil {
		return nil, fmt.Errorf("could not read magic bytes: %w", err)
	}
	if magic != MagicBytesV3 {
		return nil, fmt.Errorf("invalid v3 magic bytes")
	}

	ipType, err := buf.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("could not read ip type: %w", err)
	}
	var ip net.IP
	switch ipType {
	case 0x04:
		ipBytes := make([]byte, 4)
		if _, err := buf.Read(ipBytes); err != nil {
			return nil, fmt.Errorf("could not read ipv4 address: %w", err)
		}
		ip = net.IP(ipBytes)
	case 0x06:
		ipBytes := make([]byte, 16)
		if _, err := buf.Read(ipBytes); err != nil {
			return nil, fmt.Errorf("could not read ipv6 address: %w", err)
		}
		ip = net.IP(ipBytes)
	default:
		return nil, fmt.Errorf("unknown ip type: %d", ipType)
	}

	var port uint16
	if err := binary.Read(buf, binary.BigEndian, &port); err != nil {
		return nil, fmt.Errorf("could not read port: %w", err)
	}
	var price uint64
	if err := binary.Read(buf, binary.BigEndian, &price); err != nil {
		return nil, fmt.Errorf("could not read price: %w", err)
	}
	var bandwidthKB uint32
	if err := binary.Read(buf, binary.BigEndian, &bandwidthKB); err != nil {
		return nil, fmt.Errorf("could not read bandwidth: %w", err)
	}
	var maxConsumers uint16
	if err := binary.Read(buf, binary.BigEndian, &maxConsumers); err != nil {
		return nil, fmt.Errorf("could not read max consumers: %w", err)
	}
	country := make([]byte, 2)
	if _, err := buf.Read(country); err != nil {
		return nil, fmt.Errorf("could not read country code: %w", err)
	}
	flags, err := buf.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("could not read availability flags: %w", err)
	}
	var probePort uint16
	if err := binary.Read(buf, binary.BigEndian, &probePort); err != nil {
		return nil, fmt.Errorf("could not read throughput probe port: %w", err)
	}

	// V3 specific fields
	pricingMethod, err := buf.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("could not read pricing method: %w", err)
	}
	var timeUnitSecs uint32
	if err := binary.Read(buf, binary.BigEndian, &timeUnitSecs); err != nil {
		return nil, fmt.Errorf("could not read time unit: %w", err)
	}
	var dataUnitBytes uint32
	if err := binary.Read(buf, binary.BigEndian, &dataUnitBytes); err != nil {
		return nil, fmt.Errorf("could not read data unit: %w", err)
	}
	var sessionTimeoutSecs uint32
	if err := binary.Read(buf, binary.BigEndian, &sessionTimeoutSecs); err != nil {
		return nil, fmt.Errorf("could not read session timeout: %w", err)
	}

	if buf.Len() != btcec.PubKeyBytesLenCompressed {
		return nil, fmt.Errorf("incorrect remaining payload length for public key, expected %d, got %d", btcec.PubKeyBytesLenCompressed, buf.Len())
	}
	pubKeyBytes := make([]byte, btcec.PubKeyBytesLenCompressed)
	if _, err := buf.Read(pubKeyBytes); err != nil {
		return nil, fmt.Errorf("could not read public key: %w", err)
	}
	pubKey, err := btcec.ParsePubKey(pubKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("invalid public key in payload: %w", err)
	}

	return &VPNEndpoint{
		IP:                    ip,
		Port:                  port,
		Price:                 price,
		PublicKey:             pubKey,
		AdvertisedBandwidthKB: bandwidthKB,
		MaxConsumers:          maxConsumers,
		CountryCode:           normalizeCountryCode(string(country)),
		AvailabilityFlags:     flags,
		ThroughputProbePort:   probePort,
		PricingMethod:         pricingMethod,
		TimeUnitSecs:          timeUnitSecs,
		DataUnitBytes:         dataUnitBytes,
		SessionTimeoutSecs:    sessionTimeoutSecs,
	}, nil
}

func normalizeCountryCode(v string) string {
	up := strings.ToUpper(strings.TrimSpace(v))
	if len(up) < 2 {
		return "ZZ"
	}
	return up[:2]
}

// EncodePaymentPayload creates the OP_RETURN data for a payment transaction.
func EncodePaymentPayload(clientPubKey *btcec.PublicKey) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.BigEndian, uint32(PaymentMagicBytes)); err != nil {
		return nil, err
	}
	buf.Write(clientPubKey.SerializeCompressed())
	return buf.Bytes(), nil
}

// DecodePaymentPayload parses the OP_RETURN data from a payment transaction.
func DecodePaymentPayload(data []byte) (*btcec.PublicKey, error) {
	buf := bytes.NewReader(data)
	var magic uint32
	if err := binary.Read(buf, binary.BigEndian, &magic); err != nil {
		return nil, fmt.Errorf("could not read magic bytes: %w", err)
	}
	if magic != PaymentMagicBytes {
		return nil, fmt.Errorf("invalid payment magic bytes")
	}
	if buf.Len() != btcec.PubKeyBytesLenCompressed {
		return nil, fmt.Errorf("invalid payload length for public key, expected %d, got %d", btcec.PubKeyBytesLenCompressed, buf.Len())
	}
	pubKeyBytes := make([]byte, btcec.PubKeyBytesLenCompressed)
	if _, err := buf.Read(pubKeyBytes); err != nil {
		return nil, fmt.Errorf("could not read public key bytes: %w", err)
	}
	return btcec.ParsePubKey(pubKeyBytes)
}

// PriceUpdatePayload holds the data for a price update message.
type PriceUpdatePayload struct {
	PublicKey *btcec.PublicKey
	NewPrice  uint64
}

// EncodePriceUpdatePayload creates the OP_RETURN data for a price update.
func EncodePriceUpdatePayload(pubKey *btcec.PublicKey, price uint64) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.BigEndian, uint32(PriceUpdateMagicBytes)); err != nil {
		return nil, err
	}
	if pubKey == nil {
		return nil, fmt.Errorf("public key cannot be nil")
	}
	buf.Write(pubKey.SerializeCompressed())
	if err := binary.Write(buf, binary.BigEndian, price); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// DecodePriceUpdatePayload parses the OP_RETURN data from a price update.
func DecodePriceUpdatePayload(data []byte) (*PriceUpdatePayload, error) {
	buf := bytes.NewReader(data)
	var magic uint32
	if err := binary.Read(buf, binary.BigEndian, &magic); err != nil {
		return nil, fmt.Errorf("could not read magic bytes: %w", err)
	}
	if magic != PriceUpdateMagicBytes {
		return nil, fmt.Errorf("invalid price update magic bytes")
	}

	pubKeyBytes := make([]byte, btcec.PubKeyBytesLenCompressed)
	if _, err := buf.Read(pubKeyBytes); err != nil {
		return nil, fmt.Errorf("could not read public key: %w", err)
	}
	pubKey, err := btcec.ParsePubKey(pubKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("invalid public key in price update payload: %w", err)
	}

	var price uint64
	if err := binary.Read(buf, binary.BigEndian, &price); err != nil {
		return nil, fmt.Errorf("could not read price from payload: %w", err)
	}

	return &PriceUpdatePayload{
		PublicKey: pubKey,
		NewPrice:  price,
	}, nil
}

type HeartbeatPayload struct {
	PublicKey *btcec.PublicKey
	Flags     uint8 // bit0=available
}

func EncodeHeartbeatPayload(pubKey *btcec.PublicKey, flags uint8) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.BigEndian, uint32(HeartbeatMagicBytes)); err != nil {
		return nil, err
	}
	if pubKey == nil {
		return nil, fmt.Errorf("public key cannot be nil")
	}
	buf.Write(pubKey.SerializeCompressed())
	buf.WriteByte(flags)
	return buf.Bytes(), nil
}

func DecodeHeartbeatPayload(data []byte) (*HeartbeatPayload, error) {
	buf := bytes.NewReader(data)
	var magic uint32
	if err := binary.Read(buf, binary.BigEndian, &magic); err != nil {
		return nil, fmt.Errorf("could not read magic bytes: %w", err)
	}
	if magic != HeartbeatMagicBytes {
		return nil, fmt.Errorf("invalid heartbeat magic bytes")
	}
	if buf.Len() != btcec.PubKeyBytesLenCompressed+1 {
		return nil, fmt.Errorf("invalid heartbeat payload length: got %d", buf.Len())
	}
	pubKeyBytes := make([]byte, btcec.PubKeyBytesLenCompressed)
	if _, err := buf.Read(pubKeyBytes); err != nil {
		return nil, fmt.Errorf("could not read heartbeat pubkey: %w", err)
	}
	pubKey, err := btcec.ParsePubKey(pubKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("invalid public key in heartbeat payload: %w", err)
	}
	flags, err := buf.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("could not read heartbeat flags: %w", err)
	}
	return &HeartbeatPayload{PublicKey: pubKey, Flags: flags}, nil
}

type CertFingerprintPayload struct {
	PublicKey       *btcec.PublicKey
	CertFingerprint []byte // First 16 bytes of SHA256 of the TLS certificate
}

func ComputeCertFingerprint(certHash []byte) []byte {
	hash := sha256.Sum256(certHash)
	return hash[:16]
}

func EncodeCertFingerprintPayload(pubKey *btcec.PublicKey, certFingerprint []byte) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.BigEndian, uint32(CertFingerprintMagicBytes)); err != nil {
		return nil, err
	}
	if pubKey == nil {
		return nil, fmt.Errorf("public key cannot be nil")
	}
	buf.Write(pubKey.SerializeCompressed())
	if len(certFingerprint) < 16 {
		certFingerprint = make([]byte, 16)
	}
	buf.Write(certFingerprint[:16])
	return buf.Bytes(), nil
}

func DecodeCertFingerprintPayload(data []byte) (*CertFingerprintPayload, error) {
	buf := bytes.NewReader(data)
	var magic uint32
	if err := binary.Read(buf, binary.BigEndian, &magic); err != nil {
		return nil, fmt.Errorf("could not read magic bytes: %w", err)
	}
	if magic != CertFingerprintMagicBytes {
		return nil, fmt.Errorf("invalid cert fingerprint magic bytes")
	}
	pubKeyBytes := make([]byte, btcec.PubKeyBytesLenCompressed)
	if _, err := buf.Read(pubKeyBytes); err != nil {
		return nil, fmt.Errorf("could not read public key: %w", err)
	}
	pubKey, err := btcec.ParsePubKey(pubKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("invalid public key in cert fingerprint payload: %w", err)
	}
	fingerprint := make([]byte, 16)
	if _, err := buf.Read(fingerprint); err != nil {
		return nil, fmt.Errorf("could not read cert fingerprint: %w", err)
	}
	return &CertFingerprintPayload{PublicKey: pubKey, CertFingerprint: fingerprint}, nil
}
