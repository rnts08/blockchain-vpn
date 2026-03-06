package main

import (
	"encoding/hex"
	"log"

	"github.com/btcsuite/btcd/rpcclient"
)

// ProviderAnnouncement holds the decoded endpoint info and the transaction ID
// where it was found.
type ProviderAnnouncement struct {
	Endpoint *VPNEndpoint
	TxID     string
}

// ScanForVPNs scans the blockchain for VPN service announcements starting from
// the current tip and going backwards until startBlock.
func ScanForVPNs(client *rpcclient.Client, startBlock int64) ([]*ProviderAnnouncement, error) {
	var announcements []*ProviderAnnouncement

	// Get current block count
	count, err := client.GetBlockCount()
	if err != nil {
		return nil, err
	}

	// Iterate backwards from tip to startBlock
	for i := count; i > startBlock && i > 0; i-- {
		hash, err := client.GetBlockHash(i)
		if err != nil {
			log.Printf("Could not get block hash for height %d: %v", i, err)
			continue
		}
		block, err := client.GetBlockVerbose(hash)
		if err != nil {
			log.Printf("Could not get block for hash %s: %v", hash, err)
			continue
		}

		for _, tx := range block.Tx {
			for _, vout := range tx.Vout {
				pkScript, err := hex.DecodeString(vout.ScriptPubKey.Hex)
				if err != nil {
					continue
				}

				if payload, err := ExtractScriptPayload(pkScript); err == nil {
					if endpoint, err := DecodePayload(payload); err == nil {
						announcements = append(announcements, &ProviderAnnouncement{
							Endpoint: endpoint,
							TxID:     tx.Txid,
						})
					}
				}
			}
		}
	}
	return announcements, nil
}