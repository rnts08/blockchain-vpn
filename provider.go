package main

import (
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

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
func StartEchoServer(port int) {
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

	buf := make([]byte, 128) // Small buffer is fine for echo
	for {
		n, remoteaddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("Error reading from UDP: %v", err)
			continue
		}

		if _, err := conn.WriteToUDP(buf[0:n], remoteaddr); err != nil {
			log.Printf("Error writing echo to UDP: %v", err)
		}
	}
}

// GetPublicIP attempts to determine the public IP address of the provider
// by querying external IP echo services.
func GetPublicIP() (net.IP, error) {
	services := []string{
		"https://api.ipify.org",
		"https://ifconfig.me/ip",
		"https://icanhazip.com",
		"https://checkip.amazonaws.com",
	}

	for _, service := range services {
		client := http.Client{
			Timeout: 5 * time.Second,
		}
		resp, err := client.Get(service)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			continue
		}

		ipStr := strings.TrimSpace(string(body))
		ip := net.ParseIP(ipStr)
		if ip != nil {
			return ip, nil
		}
	}
	return nil, fmt.Errorf("failed to determine public IP from any service")
}

// MonitorPayments checks for incoming transactions to the wallet.
// It polls the blockchain periodically for new payments.
func MonitorPayments(client *rpcclient.Client, authManager *AuthManager, servicePrice uint64) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	// Start tracking from the current best block to avoid listing old transactions.
	var lastBlockHash *chainhash.Hash
	if info, err := client.GetBlockChainInfo(); err == nil {
		lastBlockHash, _ = chainhash.NewHashFromStr(info.BestBlockHash)
	}

	log.Println("Starting payment monitor...")

	for range ticker.C {
		// List transactions since the last checked block.
		result, err := client.ListSinceBlock(lastBlockHash, 1)
		if err != nil {
			log.Printf("Error checking payments: %v", err)
			continue
		}

		if hash, err := chainhash.NewHashFromStr(result.LastBlock); err == nil {
			lastBlockHash = hash
		}

		for _, tx := range result.Transactions {
			if tx.Category == "receive" {
				// Verify the amount is sufficient.
				paidAmount, _ := btcutil.NewAmount(tx.Amount)
				if uint64(paidAmount) < servicePrice {
					log.Printf("Payment received (%s) but amount %d is less than service price %d", tx.TxID, paidAmount, servicePrice)
					continue
				}

				// Get the full transaction to inspect for an OP_RETURN.
				txHash, _ := chainhash.NewHashFromStr(tx.TxID)
				txVerbose, err := client.GetRawTransactionVerbose(txHash)
				if err != nil {
					log.Printf("Could not get raw transaction for payment %s: %v", tx.TxID, err)
					continue
				}

				for _, vout := range txVerbose.Vout {
					pkScript, err := hex.DecodeString(vout.ScriptPubKey.Hex)
					if err != nil {
						continue
					}
					if clientKey, err := DecodePaymentPayload(pkScript); err == nil {
						authManager.AuthorizePeer(*clientKey, 24*time.Hour) // Authorize for 24 hours
					}
				}
			}
		}
	}
}
