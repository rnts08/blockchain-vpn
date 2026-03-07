package protocol

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/btcsuite/btcd/btcec/v2"
)

// ReputationPayloadMagic is the prefix for reputation metadata payloads.
var ReputationPayloadMagic = []byte{0x52, 0x45, 0x50, 0x01} // "REP\x01"

// ReputationPayload represents a signed reputation score for a provider.
type ReputationPayload struct {
	SubjectPublicKey *btcec.PublicKey // The provider being scored
	Score            uint8            // 0-100
	Source           string           // Identifier for the rater/source
	Signature        []byte           // Signature over the payload
}

// EncodeReputationPayload serializes a reputation record.
func EncodeReputationPayload(payload *ReputationPayload) ([]byte, error) {
	if payload == nil || payload.SubjectPublicKey == nil {
		return nil, fmt.Errorf("invalid reputation payload")
	}

	var buf bytes.Buffer
	buf.Write(ReputationPayloadMagic)

	// PubKey is 33 bytes compressed
	pubBytes := payload.SubjectPublicKey.SerializeCompressed()
	buf.Write(pubBytes)

	// 1 byte score
	buf.WriteByte(payload.Score)

	// Source string length (1 byte) + source bytes
	srcBytes := []byte(payload.Source)
	if len(srcBytes) > 255 {
		return nil, fmt.Errorf("source string too long")
	}
	buf.WriteByte(byte(len(srcBytes)))
	buf.Write(srcBytes)

	// Write signature length (2 bytes) + signature
	sigLen := uint16(len(payload.Signature))
	buf.WriteByte(byte(sigLen >> 8))
	buf.WriteByte(byte(sigLen & 0xFF))
	buf.Write(payload.Signature)

	return buf.Bytes(), nil
}

// DecodeReputationPayload deserializes a reputation record from an OP_RETURN payload.
func DecodeReputationPayload(data []byte) (*ReputationPayload, error) {
	if len(data) < len(ReputationPayloadMagic)+33+1+1+2 { // Minimum valid length
		return nil, fmt.Errorf("payload too short")
	}
	if !bytes.Equal(data[:len(ReputationPayloadMagic)], ReputationPayloadMagic) {
		return nil, fmt.Errorf("invalid magic bytes")
	}

	offset := len(ReputationPayloadMagic)

	pubKeyBytes := data[offset : offset+33]
	pubKey, err := btcec.ParsePubKey(pubKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("invalid public key: %w", err)
	}
	offset += 33

	score := data[offset]
	offset++

	srcLen := int(data[offset])
	offset++

	if len(data) < offset+srcLen+2 {
		return nil, fmt.Errorf("payload too short for source")
	}
	srcBytes := data[offset : offset+srcLen]
	source := string(srcBytes)
	offset += srcLen

	sigLen := (uint16(data[offset]) << 8) | uint16(data[offset+1])
	offset += 2

	if len(data) < offset+int(sigLen) {
		return nil, fmt.Errorf("payload too short for signature")
	}
	sigBytes := data[offset : offset+int(sigLen)]

	return &ReputationPayload{
		SubjectPublicKey: pubKey,
		Score:            score,
		Source:           source,
		Signature:        sigBytes,
	}, nil
}

// HexPubKey returns the hex encoding of the subject's compressed public key.
func (p *ReputationPayload) HexPubKey() string {
	if p.SubjectPublicKey == nil {
		return ""
	}
	return hex.EncodeToString(p.SubjectPublicKey.SerializeCompressed())
}
