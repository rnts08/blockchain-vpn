package blockchain

import (
	"context"
	"encoding/hex"
	"log"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"

	"blockchain-vpn/internal/protocol"
)

// ProviderAnnouncement holds the decoded endpoint info and the transaction ID
// where it was found.
type ProviderAnnouncement struct {
	Endpoint              *protocol.VPNEndpoint
	TxID                  string
	MetadataVersion       int
	AdvertisedBandwidthKB uint32
	MaxConsumers          uint16
	DeclaredCountry       string
	AvailabilityFlags     uint8
	LastHeartbeatSeen     bool
}

func (p *ProviderAnnouncement) AvailableSlots() int {
	if p.MaxConsumers == 0 {
		return 0
	}
	return int(p.MaxConsumers)
}

type heartbeatState struct {
	flags uint8
}

// ScanForVPNs scans the blockchain for VPN service announcements starting from
// the current tip and going backwards until startBlock or the cached max height.
func ScanForVPNs(client *rpcclient.Client, startBlock int64, cache *ScanCache) ([]*ProviderAnnouncement, map[string]uint64, error) {
	announcementByPubKey := make(map[string]*ProviderAnnouncement)
	priceUpdates := make(map[string]uint64) // Key: hex pubkey, Value: new price
	heartbeats := make(map[string]heartbeatState)

	// Get current block count
	count, err := withRetry(context.Background(), "GetBlockCount", 5, 500*time.Millisecond, func() (int64, error) {
		return client.GetBlockCount()
	})
	if err != nil {
		return nil, nil, err
	}

	effectiveStart := startBlock
	if cache != nil && cache.LastScannedHeight > effectiveStart {
		effectiveStart = cache.LastScannedHeight
	}

	// Iterate backwards from tip to effectiveStart
	for i := count; i > effectiveStart && i > 0; i-- {
		hash, err := withRetry(context.Background(), "GetBlockHash", 5, 500*time.Millisecond, func() (*chainhash.Hash, error) {
			return client.GetBlockHash(i)
		})
		if err != nil {
			log.Printf("Could not get block hash for height %d: %v", i, err)
			continue
		}
		block, err := withRetry(context.Background(), "GetBlockVerboseTx", 5, 500*time.Millisecond, func() (*btcjson.GetBlockVerboseTxResult, error) {
			return client.GetBlockVerboseTx(hash)
		})
		if err != nil {
			log.Printf("Could not get verbose tx block for hash %s: %v", hash, err)
			continue
		}

		for _, tx := range block.Tx {
			for _, vout := range tx.Vout {
				pkScript, err := hex.DecodeString(vout.ScriptPubKey.Hex)
				if err != nil {
					continue
				}

				if payload, err := protocol.ExtractScriptPayload(pkScript); err == nil {
					// Try to decode as a v2 service announcement first.
					if endpoint, err := protocol.DecodePayloadV2(payload); err == nil {
						pubKeyHex := hex.EncodeToString(endpoint.PublicKey.SerializeCompressed())
						if _, exists := announcementByPubKey[pubKeyHex]; !exists {
							announcementByPubKey[pubKeyHex] = &ProviderAnnouncement{
								Endpoint:              endpoint,
								TxID:                  tx.Txid,
								MetadataVersion:       2,
								AdvertisedBandwidthKB: endpoint.AdvertisedBandwidthKB,
								MaxConsumers:          endpoint.MaxConsumers,
								DeclaredCountry:       strings.ToUpper(endpoint.CountryCode),
								AvailabilityFlags:     endpoint.AvailabilityFlags,
							}
						}
						continue
					}

					// Try to decode as a v1 service announcement.
					if endpoint, err := protocol.DecodePayload(payload); err == nil {
						pubKeyHex := hex.EncodeToString(endpoint.PublicKey.SerializeCompressed())
						if _, exists := announcementByPubKey[pubKeyHex]; !exists {
							announcementByPubKey[pubKeyHex] = &ProviderAnnouncement{
								Endpoint:        endpoint,
								TxID:            tx.Txid,
								MetadataVersion: 1,
							}
						}
						continue // Move to next vout
					}

					// Try to decode as a price update
					if priceUpdate, err := protocol.DecodePriceUpdatePayload(payload); err == nil {
						pubKeyHex := hex.EncodeToString(priceUpdate.PublicKey.SerializeCompressed())
						// Since we are scanning backwards, the first price update we see is the most recent.
						// So we only add it if it's not already in the map.
						if _, exists := priceUpdates[pubKeyHex]; !exists {
							priceUpdates[pubKeyHex] = priceUpdate.NewPrice
						}
						continue
					}

					// Try to decode heartbeat availability update.
					if hb, err := protocol.DecodeHeartbeatPayload(payload); err == nil {
						pubKeyHex := hex.EncodeToString(hb.PublicKey.SerializeCompressed())
						if _, exists := heartbeats[pubKeyHex]; !exists {
							heartbeats[pubKeyHex] = heartbeatState{flags: hb.Flags}
						}
					}
				}
			}
		}
	}

	if cache != nil && count > effectiveStart {
		cache.Update(count, announcementByPubKey, priceUpdates, heartbeats)
		_ = cache.Save()
	}

	// Merge the delta we just scanned with whatever is in the cache (if provided).
	// If no cache, we just merge the results of this scan run.
	var finalAnns map[string]*ProviderAnnouncement
	var finalPrices map[string]uint64
	var finalHBs map[string]heartbeatState

	if cache != nil {
		cache.mu.RLock()
		finalAnns = make(map[string]*ProviderAnnouncement, len(cache.Announcements))
		for pk, ep := range cache.Announcements {
			finalAnns[pk] = &ProviderAnnouncement{Endpoint: ep}
		}
		finalPrices = make(map[string]uint64, len(cache.PriceUpdates))
		for pk, p := range cache.PriceUpdates {
			finalPrices[pk] = p
		}
		finalHBs = make(map[string]heartbeatState, len(cache.Heartbeats))
		for pk, f := range cache.Heartbeats {
			finalHBs[pk] = heartbeatState{flags: f}
		}
		cache.mu.RUnlock()

		// Overwrite with any new findings from this run
		for pk, a := range announcementByPubKey {
			finalAnns[pk] = a
		}
		for pk, p := range priceUpdates {
			finalPrices[pk] = p
		}
		for pk, h := range heartbeats {
			finalHBs[pk] = h
		}
	} else {
		finalAnns = announcementByPubKey
		finalPrices = priceUpdates
		finalHBs = heartbeats
	}

	announcements := mergeProviderState(finalAnns, finalPrices, finalHBs)
	return announcements, finalPrices, nil
}

func mergeProviderState(announcementByPubKey map[string]*ProviderAnnouncement, priceUpdates map[string]uint64, heartbeats map[string]heartbeatState) []*ProviderAnnouncement {
	announcements := make([]*ProviderAnnouncement, 0, len(announcementByPubKey))
	for pubKeyHex, ann := range announcementByPubKey {
		if ann == nil || ann.Endpoint == nil {
			continue
		}
		if newPrice, ok := priceUpdates[pubKeyHex]; ok {
			ann.Endpoint.Price = newPrice
		}
		if hb, ok := heartbeats[pubKeyHex]; ok {
			ann.AvailabilityFlags = hb.flags
			ann.LastHeartbeatSeen = true
		}
		if ann.DeclaredCountry == "" {
			ann.DeclaredCountry = strings.ToUpper(strings.TrimSpace(ann.Endpoint.CountryCode))
		}
		announcements = append(announcements, ann)
	}
	return announcements
}
