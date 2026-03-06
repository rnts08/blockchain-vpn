package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

// This file provides a basic example of a VPN provider application that
// broadcasts its service information onto the blockchain.

// AnnounceService uses the provided RPC client to broadcast a transaction
// with an OP_RETURN output containing the VPN service details.
func AnnounceService(client *rpcclient.Client, endpoint *VPNEndpoint) error {
	// 4. Encode the endpoint data into the OP_RETURN payload.
	payload, err := endpoint.EncodePayload()
	if err != nil {
		return fmt.Errorf("error encoding payload: %w", err)
	}
	log.Printf("Created payload: %x\n", payload)

	// 5. Create the OP_RETURN script.
	opReturnScript, err := txscript.NewScriptBuilder().AddOp(txscript.OP_RETURN).AddData(payload).Script()
	if err != nil {
		return fmt.Errorf("error creating OP_RETURN script: %w", err)
	}

	// 6. Find a UTXO to spend to fund the transaction.
	// A real implementation would have more sophisticated coin selection.
	unspent, err := client.ListUnspent()
	if err != nil {
		return fmt.Errorf("error listing unspent outputs: %w", err)
	}
	if len(unspent) == 0 {
		return fmt.Errorf("no unspent outputs available to create transaction")
	}
	utxo := unspent[0] // Use the first available UTXO.

	txid, err := chainhash.NewHashFromStr(utxo.TxID)
	if err != nil {
		return fmt.Errorf("error getting hash from txid: %w", err)
	}

	// 7. Create the raw transaction.
	txInput := wire.NewTxIn(&wire.OutPoint{Hash: *txid, Index: utxo.Vout}, nil, nil)
	opReturnOutput := wire.NewTxOut(0, opReturnScript)

	// Calculate change and create a change output.
	// A real app would use fee estimation (e.g., `estimatesmartfee`).
	fee := btcutil.Amount(10000) // Hardcoded fee of 10000 satoshis.
	inputAmount, _ := btcutil.NewAmount(utxo.Amount)
	changeAmount := inputAmount - fee

	changeAddr, err := client.GetRawChangeAddress("")
	if err != nil {
		return fmt.Errorf("error getting change address: %w", err)
	}
	changeScript, err := txscript.PayToAddrScript(changeAddr)
	if err != nil {
		return fmt.Errorf("error creating change script: %w", err)
	}
	changeOutput := wire.NewTxOut(int64(changeAmount), changeScript)

	tx := wire.NewMsgTx(wire.TxVersion)
	tx.AddTxIn(txInput)
	tx.AddTxOut(opReturnOutput)
	tx.AddTxOut(changeOutput)

	// 8. Sign and broadcast the transaction.
	signedTx, complete, err := client.SignRawTransactionWithWallet(tx)
	if err != nil {
		return fmt.Errorf("error signing transaction: %w", err)
	}
	if !complete {
		return fmt.Errorf("transaction signing incomplete")
	}
	txHash, err := client.SendRawTransaction(signedTx, true)
	if err != nil {
		return fmt.Errorf("error sending transaction: %w", err)
	}

	log.Printf("Successfully broadcasted announcement transaction: %s\n", txHash.String())
	return nil
}

// StartEchoServer starts a simple UDP echo server on the given port.
// This is used by clients to measure latency. It's a blocking function.
func StartEchoServer(ctx context.Context, port int) {
	addr := net.UDPAddr{
		Port: port,
		IP:   net.ParseIP("0.0.0.0"),
	}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		log.Fatalf("Failed to start echo server on port %d: %v", port, err)
	}
	defer conn.Close()
	log.Printf("UDP Echo server listening for latency checks on port %d", port)

	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	buf := make([]byte, 128) // Small buffer is fine for echo
	for {
		n, remoteaddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			select {
			case <-ctx.Done():
				log.Println("Echo server shutting down.")
				return
			default:
				log.Printf("Error reading from UDP: %v", err)
				continue
			}
		}

		if _, err := conn.WriteToUDP(buf[0:n], remoteaddr); err != nil {
			log.Printf("Error writing echo to UDP: %v", err)
		}
	}
}

// MonitorPayments checks for incoming transactions to the wallet.
// It polls the blockchain periodically for new payments.
func MonitorPayments(ctx context.Context, client *rpcclient.Client, authManager *AuthManager, servicePrice uint64) {
	// Start tracking from the current best block to avoid listing old transactions.
	var lastBlockHash *chainhash.Hash
	if info, err := client.GetBlockChainInfo(); err == nil {
		lastBlockHash, _ = chainhash.NewHashFromStr(info.BestBlockHash)
	}

	// Use notifications for new blocks. This is more efficient than polling.
	if err := client.NotifyBlocks(); err != nil {
		log.Printf("Warning: could not register for block notifications: %v. Falling back to polling.", err)
		// Fallback to polling if notifications are not available
		runPaymentMonitorPolling(ctx, client, authManager, servicePrice, lastBlockHash)
		return
	}

	log.Println("Starting payment monitor...")

	for {
		select {
		case <-ctx.Done():
			log.Println("Payment monitor shutting down.")
			client.Shutdown()
			return
		case hash, ok := <-client.Notifications():
			if !ok {
				log.Println("RPC client notifications channel closed. Shutting down payment monitor.")
				return
			}
			blockHash := hash.(*chainhash.Hash)
			block, err := client.GetBlockVerbose(blockHash)
			if err != nil {
				log.Printf("Could not get block for hash %s: %v", blockHash, err)
				continue
			}

			// Handle reorg: if the new block's parent is not our last known block,
			// we need to walk backwards to find the common ancestor.
			prevHash, _ := chainhash.NewHashFromStr(block.PreviousHash)
			if lastBlockHash != nil && *prevHash != *lastBlockHash {
				log.Printf("Reorg detected! New block %s does not connect to last known block %s", blockHash, lastBlockHash)
				// In a real implementation, you would de-authorize payments from detached blocks.
				// For this example, we'll just reset our position.
				log.Println("De-authorizing peers from potentially orphaned blocks...")
				// This is a simplified reorg handling. A robust implementation would
				// walk back to find the common ancestor and invalidate transactions in detached blocks.
				authManager.DeauthorizeAllPeers()
			}

			// Process transactions in the new block
			processBlockForPayments(client, block, authManager, servicePrice)
			lastBlockHash = blockHash
		}
	}
}

// runPaymentMonitorPolling is a fallback for when block notifications are not available.
func runPaymentMonitorPolling(ctx context.Context, client *rpcclient.Client, authManager *AuthManager, servicePrice uint64, lastBlockHash *chainhash.Hash) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			result, err := client.ListSinceBlock(lastBlockHash, 1)
			if err != nil {
				log.Printf("Error checking payments (polling): %v", err)
				continue
			}
			for _, tx := range result.Transactions {
				if tx.Removed {
					log.Printf("Transaction %s was removed (reorg). De-authorizing...", tx.TxID)
					// Find the public key associated with this tx and de-authorize
					// This requires storing a mapping from txid to pubkey, which we don't do yet.
					// As a simpler approach, we could de-authorize all peers on any reorg.
					authManager.DeauthorizeAllPeers()
					continue
				}
				processTxForPayment(client, tx.TxID, tx.Amount, authManager, servicePrice)
			}
			if hash, err := chainhash.NewHashFromStr(result.LastBlock); err == nil {
				lastBlockHash = hash
			}
		}
	}
}

// processBlockForPayments iterates through transactions in a block and processes them.
func processBlockForPayments(client *rpcclient.Client, block *btcjson.GetBlockVerboseResult, authManager *AuthManager, servicePrice uint64) {
	for _, tx := range block.Tx {
		// We need to check if the transaction involves our wallet.
		// GetRawTransactionVerbose doesn't tell us this directly.
		// A more robust way is to use `listsinceblock` as in the polling method,
		// or to use `walletnotify` if available.
		// For simplicity, we'll just re-fetch the tx to see if it's a wallet tx.
		txDetails, err := client.GetTransaction(tx.Txid)
		if err == nil && txDetails.Amount > 0 { // A simple check for received payments
			processTxForPayment(client, tx.Txid, txDetails.Amount, authManager, servicePrice)
		}
	}
}

// processTxForPayment checks a single transaction for a valid payment payload.
func processTxForPayment(client *rpcclient.Client, txid string, amount float64, authManager *AuthManager, servicePrice uint64) {
	paidAmount, _ := btcutil.NewAmount(amount)
	if uint64(paidAmount) < servicePrice {
		return // Not a valid payment amount
	}

	txHash, _ := chainhash.NewHashFromStr(txid)
	txVerbose, err := client.GetRawTransactionVerbose(txHash)
	if err != nil {
		log.Printf("Could not get raw transaction for payment %s: %v", txid, err)
		return
	}

	for _, vout := range txVerbose.Vout {
		pkScript, err := hex.DecodeString(vout.ScriptPubKey.Hex)
		if err != nil {
			continue
		}
		if clientKey, err := DecodePaymentPayload(pkScript); err == nil {
			authManager.AuthorizePeer(clientKey, 24*time.Hour) // Authorize for 24 hours
		}
	}
}
