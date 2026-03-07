package blockchain

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"

	"blockchain-vpn/internal/auth"
	"blockchain-vpn/internal/protocol"
)

// This file provides a basic example of a VPN provider application that
// broadcasts its service information onto the blockchain.

// AnnounceService uses the provided RPC client to broadcast a transaction
// with an OP_RETURN output containing the VPN service details.
func AnnounceService(client *rpcclient.Client, endpoint *protocol.VPNEndpoint) error {
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

	// 6. Estimate Fee
	feePerKb, err := estimateDynamicFeePerKb(context.Background(), client, 6)
	if err != nil {
		return fmt.Errorf("failed to determine dynamic fee: %w", err)
	}

	// Estimate size: 1 input, 2 outputs (op_return, change) ~ 250 bytes
	estimatedSize := 250
	requiredFee := btcutil.Amount(float64(feePerKb) * float64(estimatedSize) / 1000.0)

	// 7. Coin Selection
	// We need enough for fee (since output is 0 value OP_RETURN)
	utxos, totalInput, err := selectCoins(client, requiredFee)
	if err != nil {
		return err
	}

	// 8. Create the raw transaction.
	tx := wire.NewMsgTx(wire.TxVersion)
	for _, utxo := range utxos {
		txHash, _ := chainhash.NewHashFromStr(utxo.TxID)
		txIn := wire.NewTxIn(wire.NewOutPoint(txHash, utxo.Vout), nil, nil)
		tx.AddTxIn(txIn)
	}

	opReturnOutput := wire.NewTxOut(0, opReturnScript)
	changeAmount := totalInput - requiredFee

	changeAddr, err := client.GetRawChangeAddress("")
	if err != nil {
		return fmt.Errorf("error getting change address: %w", err)
	}
	changeScript, err := txscript.PayToAddrScript(changeAddr)
	if err != nil {
		return fmt.Errorf("error creating change script: %w", err)
	}
	changeOutput := wire.NewTxOut(int64(changeAmount), changeScript)

	tx.AddTxOut(opReturnOutput)
	tx.AddTxOut(changeOutput)

	// 9. Sign and broadcast the transaction.
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

// AnnouncePriceUpdate broadcasts a lightweight transaction to update the provider's price.
func AnnouncePriceUpdate(client *rpcclient.Client, pubKey *btcec.PublicKey, newPrice uint64) error {
	payload, err := protocol.EncodePriceUpdatePayload(pubKey, newPrice)
	if err != nil {
		return fmt.Errorf("error encoding price update payload: %w", err)
	}

	opReturnScript, err := txscript.NewScriptBuilder().AddOp(txscript.OP_RETURN).AddData(payload).Script()
	if err != nil {
		return fmt.Errorf("error creating OP_RETURN script: %w", err)
	}

	// Estimate Fee
	feePerKb, err := estimateDynamicFeePerKb(context.Background(), client, 6)
	if err != nil {
		return fmt.Errorf("failed to determine dynamic fee: %w", err)
	}

	estimatedSize := 250
	requiredFee := btcutil.Amount(float64(feePerKb) * float64(estimatedSize) / 1000.0)

	utxos, totalInput, err := selectCoins(client, requiredFee)
	if err != nil {
		return err
	}

	tx := wire.NewMsgTx(wire.TxVersion)
	for _, utxo := range utxos {
		txHash, _ := chainhash.NewHashFromStr(utxo.TxID)
		txIn := wire.NewTxIn(wire.NewOutPoint(txHash, utxo.Vout), nil, nil)
		tx.AddTxIn(txIn)
	}
	changeAmount := totalInput - requiredFee

	changeAddr, err := client.GetRawChangeAddress("")
	if err != nil {
		return fmt.Errorf("error getting change address: %w", err)
	}
	changeScript, err := txscript.PayToAddrScript(changeAddr)
	if err != nil {
		return fmt.Errorf("error creating change script: %w", err)
	}
	changeOutput := wire.NewTxOut(int64(changeAmount), changeScript)

	opReturnOutput := wire.NewTxOut(0, opReturnScript)
	tx.AddTxOut(opReturnOutput)
	tx.AddTxOut(changeOutput)

	// Sign and broadcast.
	signedTx, complete, err := client.SignRawTransactionWithWallet(tx)
	if err != nil || !complete {
		return fmt.Errorf("error signing transaction: %w", err)
	}

	txHash, err := client.SendRawTransaction(signedTx, true)
	if err != nil {
		return fmt.Errorf("error sending transaction: %w", err)
	}

	log.Printf("Successfully broadcasted price update transaction: %s\n", txHash.String())
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
func MonitorPayments(ctx context.Context, client *rpcclient.Client, authManager *auth.AuthManager, servicePrice uint64) {
	// Start tracking from the current best block to avoid listing old transactions.
	var lastBlockHash *chainhash.Hash
	payments := newPaymentTracker()

	info, err := withRetry(ctx, "GetBlockChainInfo", 5, 1*time.Second, func() (*btcjson.GetBlockChainInfoResult, error) {
		return client.GetBlockChainInfo()
	})
	if err == nil {
		lastBlockHash, _ = chainhash.NewHashFromStr(info.BestBlockHash)
	} else {
		log.Printf("Warning: could not initialize monitor from best block: %v", err)
	}

	log.Println("Starting payment monitor (polling mode)...")
	runPaymentMonitorPolling(ctx, client, authManager, servicePrice, lastBlockHash, payments)
}

// runPaymentMonitorPolling is a fallback for when block notifications are not available.
func runPaymentMonitorPolling(ctx context.Context, client *rpcclient.Client, authManager *auth.AuthManager, servicePrice uint64, lastBlockHash *chainhash.Hash, payments *paymentTracker) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			result, err := withRetry(ctx, "ListSinceBlock", 5, 1*time.Second, func() (*btcjson.ListSinceBlockResult, error) {
				return client.ListSinceBlock(lastBlockHash)
			})
			if err != nil {
				log.Printf("Error checking payments (polling): %v", err)
				continue
			}
			for _, tx := range result.Transactions {
				processTxForPayment(ctx, client, tx.TxID, tx.Amount, tx.Confirmations, authManager, servicePrice, payments)
			}
			if hash, err := chainhash.NewHashFromStr(result.LastBlock); err == nil {
				lastBlockHash = hash
			}
		}
	}
}

// processBlockForPayments iterates through transactions in a block and processes them.
func processBlockForPayments(ctx context.Context, client *rpcclient.Client, block *btcjson.GetBlockVerboseTxResult, authManager *auth.AuthManager, servicePrice uint64, payments *paymentTracker) {
	for _, tx := range block.Tx {
		// We need to check if the transaction involves our wallet.
		// GetRawTransactionVerbose doesn't tell us this directly.
		// A more robust way is to use `listsinceblock` as in the polling method,
		// or to use `walletnotify` if available.
		// For simplicity, we'll just re-fetch the tx to see if it's a wallet tx.
		txHash, err := chainhash.NewHashFromStr(tx.Txid)
		if err != nil {
			continue
		}
		txDetails, err := client.GetTransaction(txHash)
		if err == nil && txDetails.Amount > 0 { // A simple check for received payments
			processTxForPayment(ctx, client, tx.Txid, txDetails.Amount, int64(txDetails.Confirmations), authManager, servicePrice, payments)
		}
	}
}

// processTxForPayment checks a single transaction for a valid payment payload.
func processTxForPayment(ctx context.Context, client *rpcclient.Client, txid string, amount float64, confirmations int64, authManager *auth.AuthManager, servicePrice uint64, payments *paymentTracker) {
	if confirmations < 0 {
		payments.handleRemovedTx(txid, authManager)
		return
	}

	paidAmount, _ := btcutil.NewAmount(amount)
	if uint64(paidAmount) < servicePrice {
		return // Not a valid payment amount
	}

	txHash, _ := chainhash.NewHashFromStr(txid)
	txVerbose, err := withRetry(ctx, "GetRawTransactionVerbose(payment)", 5, 1*time.Second, func() (*btcjson.TxRawResult, error) {
		return client.GetRawTransactionVerbose(txHash)
	})
	if err != nil {
		log.Printf("Could not get raw transaction for payment %s: %v", txid, err)
		return
	}

	for _, vout := range txVerbose.Vout {
		pkScript, err := hex.DecodeString(vout.ScriptPubKey.Hex)
		if err != nil {
			continue
		}
		payload, err := protocol.ExtractScriptPayload(pkScript)
		if err != nil {
			continue
		}
		if clientKey, err := protocol.DecodePaymentPayload(payload); err == nil {
			authManager.AuthorizePeer(clientKey, 24*time.Hour)
			payments.trackPayment(txid, clientKey)
		}
	}
}
