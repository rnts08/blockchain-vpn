package blockchain

import (
	"encoding/hex"
	"testing"

	"blockchain-vpn/internal/protocol"

	"github.com/btcsuite/btcd/btcec/v2"
)

func TestMergeProviderStateAppliesPriceAndHeartbeat(t *testing.T) {
	priv, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("failed to create key: %v", err)
	}
	pub := priv.PubKey()
	pubHex := hex.EncodeToString(pub.SerializeCompressed())

	announcementByPubKey := map[string]*ProviderAnnouncement{
		pubHex: {
			Endpoint: &protocol.VPNEndpoint{
				Price:     1000,
				PublicKey: pub,
			},
			MetadataVersion: 2,
		},
	}
	priceUpdates := map[string]uint64{pubHex: 1500}
	heartbeats := map[string]heartbeatState{pubHex: {flags: protocol.AvailabilityFlagAvailable}}

	merged := mergeProviderState(announcementByPubKey, priceUpdates, heartbeats, nil)
	if len(merged) != 1 {
		t.Fatalf("expected one merged announcement, got %d", len(merged))
	}
	got := merged[0]
	if got.Endpoint.Price != 1500 {
		t.Fatalf("price update not applied: got %d", got.Endpoint.Price)
	}
	if got.AvailabilityFlags != protocol.AvailabilityFlagAvailable {
		t.Fatalf("heartbeat flags not applied")
	}
	if !got.LastHeartbeatSeen {
		t.Fatalf("expected LastHeartbeatSeen true")
	}
}
