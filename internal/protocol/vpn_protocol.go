package protocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"

	"github.com/btcsuite/btcd/btcec/v2"
)

const MagicBytes = 0x56504E01            // "VPN" + Version 1
const PriceUpdateMagicBytes = 0x50524943 // "PRIC"
const PaymentMagicBytes = 0x50415901     // "PAY" + Version 1

type VPNEndpoint struct {
	IP        net.IP
	Port      uint16
	Price     uint64           // Satoshis per session
	PublicKey *btcec.PublicKey // 33 bytes for compressed secp256k1
}

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
