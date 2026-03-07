package blockchain

import (
	"context"
	"encoding/hex"
	"log"
	"time"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"

	"blockchain-vpn/internal/protocol"
)

// ProviderAnnouncement holds the decoded endpoint info and the transaction ID
// where it was found.
type ProviderAnnouncement struct {
	Endpoint *protocol.VPNEndpoint
	TxID     string
}

// ScanForVPNs scans the blockchain for VPN service announcements starting from
// the current tip and going backwards until startBlock.
func ScanForVPNs(client *rpcclient.Client, startBlock int64) ([]*ProviderAnnouncement, map[string]uint64, error) {
	var announcements []*ProviderAnnouncement
	priceUpdates := make(map[string]uint64) // Key: hex pubkey, Value: new price

	// Get current block count
	count, err := withRetry(context.Background(), "GetBlockCount", 5, 500*time.Millisecond, func() (int64, error) {
		return client.GetBlockCount()
	})
	if err != nil {
		return nil, nil, err
	}

	// Iterate backwards from tip to startBlock
	for i := count; i > startBlock && i > 0; i-- {
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
					// Try to decode as a service announcement
					if endpoint, err := protocol.DecodePayload(payload); err == nil {
						announcements = append(announcements, &ProviderAnnouncement{
							Endpoint: endpoint,
							TxID:     tx.Txid,
						})
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
					}
				}
			}
		}
	}
	return announcements, priceUpdates, nil
}
