package blockchain

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"net"
	"strings"
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

// signWithSecp256k1 creates an ECDSA signature using the secp256k1 curve.
func signWithSecp256k1(privKey *btcec.PrivateKey, message []byte) ([]byte, error) {
	ecdsaPriv := &ecdsa.PrivateKey{
		D: new(big.Int).SetBytes(privKey.Serialize()),
	}
	ecdsaPriv.PublicKey.Curve = elliptic.P256()
	ecdsaPriv.PublicKey.X, ecdsaPriv.PublicKey.Y = elliptic.P256().ScalarBaseMult(ecdsaPriv.D.Bytes())

	hash := sha256.Sum256(message)
	sig, err := ecdsa.SignASN1(rand.Reader, ecdsaPriv, hash[:])
	if err != nil {
		return nil, err
	}
	return sig, nil
}

// ecdsaSignature represents an ECDSA signature with r and s values.
type ecdsaSignature struct {
	R, S *big.Int
}

// verifyASN1Signature verifies an ASN.1 encoded ECDSA signature.
func verifyASN1Signature(pub *ecdsa.PublicKey, hash, sigASN1 []byte) bool {
	var sig ecdsaSignature
	if _, err := asn1.Unmarshal(sigASN1, &sig); err != nil {
		return false
	}
	return ecdsa.Verify(pub, hash, sig.R, sig.S)
}

// DetectAddressType probes the node to determine the appropriate address type for wallet
// operations. It first checks existing UTXOs (from scriptPubKey hex), then falls back to
// probing with getrawchangeaddress and getnewaddress. Returns the address type string
// ("p2pkh", "bech32", etc.) or an error.
func DetectAddressType(client *rpcclient.Client) (string, error) {
	candidateTypes := []string{"p2pkh", "p2sh", "bech32", "bech32m", "legacy"}

	// Method 1: Inspect existing UTXOs from listunspent
	unspent, err := client.ListUnspentMin(0)
	if err == nil && len(unspent) > 0 {
		for _, utxo := range unspent {
			// Try scriptPubKey hex first
			if utxo.ScriptPubKey != "" {
				hexBytes, err := hex.DecodeString(utxo.ScriptPubKey)
				if err == nil && len(hexBytes) >= 2 {
					scriptClass := txscript.GetScriptClass(hexBytes)
					switch scriptClass {
					case txscript.PubKeyHashTy:
						return "p2pkh", nil
					case txscript.ScriptHashTy:
						return "p2sh", nil
					case txscript.WitnessV0PubKeyHashTy:
						return "bech32", nil
					case txscript.WitnessV0ScriptHashTy:
						return "bech32m", nil
					case txscript.WitnessV1TaprootTy:
						return "bech32m", nil
					}
					hexLower := strings.ToLower(utxo.ScriptPubKey)
					if strings.HasPrefix(hexLower, "76a9") {
						return "p2pkh", nil
					}
					if strings.HasPrefix(hexLower, "a914") {
						return "p2sh", nil
					}
					if strings.HasPrefix(hexLower, "0014") {
						return "bech32", nil
					}
					if strings.HasPrefix(hexLower, "0020") {
						return "bech32m", nil
					}
				}
			}
			// Try to determine from address string prefix (OrdexCoin addresses start with 'o')
			if utxo.Address != "" {
				addr := strings.TrimSpace(utxo.Address)
				if len(addr) > 0 {
					switch addr[0] {
					case 'o', 'O':
						return "p2pkh", nil
					case '1', '3':
						return "p2sh", nil
					case 'b', 'B', 't':
						return "bech32", nil
					}
				}
			}
		}
	}

	// Method 2: Probe getrawchangeaddress with common types
	for _, addrType := range candidateTypes {
		_, err := client.GetRawChangeAddress(addrType)
		if err == nil {
			return addrType, nil
		}
	}

	// Method 3: Probe getnewaddress as fallback
	for _, addrType := range candidateTypes {
		_, err := client.GetNewAddress(addrType)
		if err == nil {
			return addrType, nil
		}
	}

	return "", fmt.Errorf("could not detect address type: no usable UTXOs found and all address type probes failed")
}

// This file provides a basic example of a VPN provider application that
// broadcasts its service information onto the blockchain.

// AnnounceService uses the provided RPC client to broadcast a transaction
// with an OP_RETURN output containing the VPN service details.
func AnnounceService(client *rpcclient.Client, endpoint *protocol.VPNEndpoint, feeTargetBlocks int, feeMode string, addressType string) error {
	// 4. Encode endpoint data into OP_RETURN payload (v2 metadata-first format).
	payload, err := endpoint.EncodePayloadV2()
	if err != nil {
		return fmt.Errorf("error encoding payload: %w", err)
	}
	log.Printf("Encoded service announcement payload (%d bytes)", len(payload))

	// 5. Create the OP_RETURN script.
	opReturnScript, err := txscript.NewScriptBuilder().AddOp(txscript.OP_RETURN).AddData(payload).Script()
	if err != nil {
		return fmt.Errorf("error creating OP_RETURN script: %w", err)
	}

	// 6. Estimate Fee
	feePerKb, err := estimateDynamicFeePerKbWithMode(context.Background(), client, int64(feeTargetBlocks), FeeMode(feeMode))
	if err != nil {
		return fmt.Errorf("failed to determine dynamic fee: %w", err)
	}

	// Estimate size: 1 input, 2 outputs (op_return, change) ~ 250 bytes
	estimatedSize := 250
	requiredFee := btcutil.Amount(float64(feePerKb) * float64(estimatedSize) / 1000.0)

	// 7. Coin Selection
	// We need enough for fee (since output is 0 value OP_RETURN)
	utxos, totalInput, changeScript, err := selectCoinsForTx(client, requiredFee)
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
	txHash, err := sendRawTransaction(client, signedTx)
	if err != nil {
		return fmt.Errorf("error sending transaction: %w", err)
	}

	log.Printf("Successfully broadcasted announcement transaction: %s\n", txHash.String())
	return nil
}

// AnnounceHeartbeat broadcasts provider availability flags for discovery freshness.
func AnnounceHeartbeat(client *rpcclient.Client, pubKey *btcec.PublicKey, flags uint8, addressType string) error {
	payload, err := protocol.EncodeHeartbeatPayload(pubKey, flags)
	if err != nil {
		return fmt.Errorf("error encoding heartbeat payload: %w", err)
	}
	opReturnScript, err := txscript.NewScriptBuilder().AddOp(txscript.OP_RETURN).AddData(payload).Script()
	if err != nil {
		return fmt.Errorf("error creating OP_RETURN script: %w", err)
	}

	feePerKb, err := estimateDynamicFeePerKb(context.Background(), client, 6)
	if err != nil {
		return fmt.Errorf("failed to determine dynamic fee: %w", err)
	}
	requiredFee := btcutil.Amount(float64(feePerKb) * 250.0 / 1000.0)
	utxos, totalInput, changeScript, err := selectCoinsForTx(client, requiredFee)
	if err != nil {
		return err
	}

	tx := wire.NewMsgTx(wire.TxVersion)
	for _, utxo := range utxos {
		txHash, _ := chainhash.NewHashFromStr(utxo.TxID)
		tx.AddTxIn(wire.NewTxIn(wire.NewOutPoint(txHash, utxo.Vout), nil, nil))
	}
	changeAmount := totalInput - requiredFee
	tx.AddTxOut(wire.NewTxOut(0, opReturnScript))
	tx.AddTxOut(wire.NewTxOut(int64(changeAmount), changeScript))

	signedTx, complete, err := client.SignRawTransactionWithWallet(tx)
	if err != nil || !complete {
		return fmt.Errorf("error signing transaction: %w", err)
	}
	txHash, err := sendRawTransaction(client, signedTx)
	if err != nil {
		return fmt.Errorf("error sending transaction: %w", err)
	}
	log.Printf("Successfully broadcasted provider heartbeat transaction: %s\n", txHash.String())
	return nil
}

// AnnouncePriceUpdate broadcasts a lightweight transaction to update the provider's price.
func AnnouncePriceUpdate(client *rpcclient.Client, pubKey *btcec.PublicKey, newPrice uint64, feeTargetBlocks int, feeMode string, addressType string) error {
	payload, err := protocol.EncodePriceUpdatePayload(pubKey, newPrice)
	if err != nil {
		return fmt.Errorf("error encoding price update payload: %w", err)
	}

	opReturnScript, err := txscript.NewScriptBuilder().AddOp(txscript.OP_RETURN).AddData(payload).Script()
	if err != nil {
		return fmt.Errorf("error creating OP_RETURN script: %w", err)
	}

	// Estimate Fee
	feePerKb, err := estimateDynamicFeePerKbWithMode(context.Background(), client, int64(feeTargetBlocks), FeeMode(feeMode))
	if err != nil {
		return fmt.Errorf("failed to determine dynamic fee: %w", err)
	}

	estimatedSize := 250
	requiredFee := btcutil.Amount(float64(feePerKb) * float64(estimatedSize) / 1000.0)

	utxos, totalInput, changeScript, err := selectCoinsForTx(client, requiredFee)
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

	changeOutput := wire.NewTxOut(int64(changeAmount), changeScript)

	opReturnOutput := wire.NewTxOut(0, opReturnScript)
	tx.AddTxOut(opReturnOutput)
	tx.AddTxOut(changeOutput)

	// Sign and broadcast.
	signedTx, complete, err := client.SignRawTransactionWithWallet(tx)
	if err != nil || !complete {
		return fmt.Errorf("error signing transaction: %w", err)
	}

	txHash, err := sendRawTransaction(client, signedTx)
	if err != nil {
		return fmt.Errorf("error sending transaction: %w", err)
	}

	log.Printf("Successfully broadcasted price update transaction: %s\n", txHash.String())
	return nil
}

// AnnounceRating broadcasts a reputation/rating for a provider on the blockchain.
func AnnounceRating(client *rpcclient.Client, providerPubKey *btcec.PublicKey, clientPrivKey *btcec.PrivateKey, score uint8, source string, feeTargetBlocks int, feeMode string, addressType string) error {
	repPayload := &protocol.ReputationPayload{
		SubjectPublicKey: providerPubKey,
		Score:            score,
		Source:           source,
		Signature:        nil,
	}

	payloadData, err := protocol.EncodeReputationPayloadWithoutSignature(repPayload)
	if err != nil {
		return fmt.Errorf("error encoding reputation payload: %w", err)
	}

	sigASN1, err := signWithSecp256k1(clientPrivKey, payloadData)
	if err != nil {
		return fmt.Errorf("error signing reputation payload: %w", err)
	}

	repPayload.Signature = sigASN1
	payload, err := protocol.EncodeReputationPayload(repPayload)
	if err != nil {
		return fmt.Errorf("error encoding final reputation payload: %w", err)
	}

	opReturnScript, err := txscript.NewScriptBuilder().AddOp(txscript.OP_RETURN).AddData(payload).Script()
	if err != nil {
		return fmt.Errorf("error creating OP_RETURN script: %w", err)
	}

	feePerKb, err := estimateDynamicFeePerKbWithMode(context.Background(), client, int64(feeTargetBlocks), FeeMode(feeMode))
	if err != nil {
		return fmt.Errorf("failed to determine dynamic fee: %w", err)
	}

	requiredFee := btcutil.Amount(float64(feePerKb) * 250.0 / 1000.0)

	utxos, totalInput, changeScript, err := selectCoinsForTx(client, requiredFee)
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

	opReturnOutput := wire.NewTxOut(0, opReturnScript)
	tx.AddTxOut(opReturnOutput)
	tx.AddTxOut(wire.NewTxOut(int64(changeAmount), changeScript))

	signedTx, complete, err := client.SignRawTransactionWithWallet(tx)
	if err != nil || !complete {
		return fmt.Errorf("error signing transaction: %w", err)
	}

	txHash, err := sendRawTransaction(client, signedTx)
	if err != nil {
		return fmt.Errorf("error sending transaction: %w", err)
	}

	log.Printf("Successfully broadcasted rating transaction: %s (provider=%s, score=%d)\n", txHash.String(), hex.EncodeToString(providerPubKey.SerializeCompressed()), score)
	return nil
}

// StartEchoServer starts a simple UDP echo server on the given port.
// This is used by clients to measure latency. It's a blocking function.
func StartEchoServer(ctx context.Context, port int) error {
	addr := net.UDPAddr{
		Port: port,
		IP:   net.ParseIP("0.0.0.0"),
	}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		return fmt.Errorf("failed to start echo server on port %d: %w", port, err)
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
				return nil
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

// PaymentMonitorCfg holds configuration for the payment monitor to interpret payments.
type PaymentMonitorCfg struct {
	Price                 uint64
	PricingMethod         string
	TimeUnitSecs          uint32
	DataUnitBytes         uint32
	MaxSessionSecs        int
	RequiredConfirmations int // Minimum confirmations before authorizing peer (default: 1)
}

// MonitorPayments checks for incoming transactions to the wallet.
// It polls the blockchain periodically for new payments.
func MonitorPayments(ctx context.Context, client *rpcclient.Client, authManager *auth.AuthManager, pmCfg PaymentMonitorCfg, pollingInterval time.Duration) {
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
	runPaymentMonitorPolling(ctx, client, authManager, pmCfg, lastBlockHash, payments, pollingInterval)
}

// runPaymentMonitorPolling is a fallback for when block notifications are not available.
func runPaymentMonitorPolling(ctx context.Context, client *rpcclient.Client, authManager *auth.AuthManager, pmCfg PaymentMonitorCfg, lastBlockHash *chainhash.Hash, payments *paymentTracker, interval time.Duration) {
	if interval <= 0 {
		interval = 1 * time.Minute
	}
	ticker := time.NewTicker(interval)
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
				processTxForPayment(ctx, client, tx.TxID, tx.Amount, tx.Confirmations, authManager, pmCfg, payments)
			}
			if hash, err := chainhash.NewHashFromStr(result.LastBlock); err == nil {
				lastBlockHash = hash
			}
		}
	}
}

// processBlockForPayments iterates through transactions in a block and processes them.
func processBlockForPayments(ctx context.Context, client *rpcclient.Client, block *btcjson.GetBlockVerboseTxResult, authManager *auth.AuthManager, pmCfg PaymentMonitorCfg, payments *paymentTracker) {
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
			processTxForPayment(ctx, client, tx.Txid, txDetails.Amount, int64(txDetails.Confirmations), authManager, pmCfg, payments)
		}
	}
}

// GetProviderAddresses returns all addresses controlled by the wallet that have UTXOs.
func GetProviderAddresses(client *rpcclient.Client) ([]string, error) {
	unspent, err := client.ListUnspent()
	if err != nil {
		return nil, err
	}
	var addresses []string
	seen := make(map[string]bool)
	for _, u := range unspent {
		if !seen[u.Address] {
			addresses = append(addresses, u.Address)
			seen[u.Address] = true
		}
	}
	// Fallback to change address if list is empty
	if len(addresses) == 0 {
		addr, err := client.GetRawChangeAddress("")
		if err == nil {
			addresses = append(addresses, addr.String())
		}
	}
	return addresses, nil
}

// processTxForPayment checks a single transaction for a valid payment payload.
// It verifies the actual amount paid to the provider's address (not just wallet balance change)
// and ensures it meets or exceeds the service price. It then authorizes the peer based on pricing model.
func processTxForPayment(ctx context.Context, client *rpcclient.Client, txid string, amount float64, confirmations int64, authManager *auth.AuthManager, pmCfg PaymentMonitorCfg, payments *paymentTracker) {
	if confirmations < 0 {
		payments.handleRemovedTx(txid, authManager)
		return
	}

	minConfirmations := pmCfg.RequiredConfirmations
	if minConfirmations <= 0 {
		minConfirmations = 1 // Default to 1 confirmation for security
	}
	if confirmations < int64(minConfirmations) {
		log.Printf("Payment %s: waiting for confirmations (%d/%d)", txid, confirmations, minConfirmations)
		return
	}

	txHash, _ := chainhash.NewHashFromStr(txid)
	txVerbose, err := withRetry(ctx, "GetRawTransactionVerbose(payment)", 5, 1*time.Second, func() (*btcjson.TxRawResult, error) {
		return client.GetRawTransactionVerbose(txHash)
	})
	if err != nil {
		log.Printf("Could not get raw transaction for payment %s: %v", txid, err)
		return
	}

	providerAddresses, err := GetProviderAddresses(client)
	if err != nil {
		log.Printf("Could not get provider addresses: %v", err)
		return
	}
	addressSet := make(map[string]bool)
	for _, addr := range providerAddresses {
		addressSet[addr] = true
	}

	var totalToProvider uint64
	var clientPubKey *btcec.PublicKey

	for _, vout := range txVerbose.Vout {
		if len(vout.ScriptPubKey.Addresses) == 0 {
			continue
		}
		addr := vout.ScriptPubKey.Addresses[0]
		if addressSet[addr] {
			totalToProvider += uint64(vout.Value * 1e8)
		}

		pkScript, err := hex.DecodeString(vout.ScriptPubKey.Hex)
		if err != nil {
			continue
		}
		payload, err := protocol.ExtractScriptPayload(pkScript)
		if err != nil {
			continue
		}
		if clientKey, err := protocol.DecodePaymentPayload(payload); err == nil {
			clientPubKey = clientKey
		}
	}

	if totalToProvider < pmCfg.Price {
		log.Printf("Payment %s: received %d sats, expected at least %d sats - ignoring", txid, totalToProvider, pmCfg.Price)
		return
	}

	log.Printf("Payment %s: verified %d sats to provider (>= %d sats required)", txid, totalToProvider, pmCfg.Price)

	if clientPubKey == nil {
		return
	}

	// Compute authorization based on pricing method
	var duration time.Duration
	var dataQuota uint64

	switch strings.ToLower(pmCfg.PricingMethod) {
	case "session", "":
		// Default: 24 hours, capped by max session duration if set
		duration = 24 * time.Hour
		if pmCfg.MaxSessionSecs > 0 {
			maxDur := time.Duration(pmCfg.MaxSessionSecs) * time.Second
			if maxDur < duration {
				duration = maxDur
			}
		}
	case "time":
		// Payment amount buys time units
		if pmCfg.Price == 0 {
			log.Printf("Payment %s: price is zero, cannot calculate time units", txid)
			return
		}
		units := totalToProvider / pmCfg.Price
		secs := uint64(units) * uint64(pmCfg.TimeUnitSecs)
		if pmCfg.MaxSessionSecs > 0 && int(secs) > pmCfg.MaxSessionSecs {
			secs = uint64(pmCfg.MaxSessionSecs)
		}
		duration = time.Duration(secs) * time.Second
	case "data":
		// Payment amount buys data units
		if pmCfg.Price == 0 {
			log.Printf("Payment %s: price is zero, cannot calculate data units", txid)
			return
		}
		units := totalToProvider / pmCfg.Price
		dataQuota = units * uint64(pmCfg.DataUnitBytes)
		// For data, we may still set a time limit (max session duration) or default 24h
		if pmCfg.MaxSessionSecs > 0 {
			duration = time.Duration(pmCfg.MaxSessionSecs) * time.Second
		} else {
			duration = 24 * time.Hour
		}
	default:
		log.Printf("Payment %s: unknown pricing method %s, ignoring", txid, pmCfg.PricingMethod)
		return
	}

	authManager.AuthorizePeer(clientPubKey, duration, dataQuota)
	payments.trackPayment(txid, clientPubKey)
}
