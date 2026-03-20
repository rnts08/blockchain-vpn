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
	"math"
	"math/big"
	"net"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"

	"blockchain-vpn/internal/auth"
	"blockchain-vpn/internal/protocol"
)

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

type ecdsaSignature struct {
	R, S *big.Int
}

func verifyASN1Signature(pub *ecdsa.PublicKey, hash, sigASN1 []byte) bool {
	var sig ecdsaSignature
	if _, err := asn1.Unmarshal(sigASN1, &sig); err != nil {
		return false
	}
	return ecdsa.Verify(pub, hash, sig.R, sig.S)
}

func DetectAddressType(client *rpcclient.Client) (string, error) {
	candidateTypes := []string{"p2pkh", "p2sh", "bech32", "bech32m", "legacy"}

	unspent, err := client.ListUnspentMin(0)
	if err == nil && len(unspent) > 0 {
		for _, utxo := range unspent {
			if utxo.ScriptPubKey != "" {
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

	for _, addrType := range candidateTypes {
		_, err := client.GetRawChangeAddress(addrType)
		if err == nil {
			return addrType, nil
		}
	}

	for _, addrType := range candidateTypes {
		_, err := client.GetNewAddress(addrType)
		if err == nil {
			return addrType, nil
		}
	}

	return "", fmt.Errorf("could not detect address type: no usable UTXOs found and all address type probes failed")
}

func AnnounceService(client *rpcclient.Client, endpoint *protocol.VPNEndpoint, feeTargetBlocks int, feeMode string, addressType string, cfg FeeConfig) error {
	payload, err := endpoint.EncodePayloadV2()
	if err != nil {
		return fmt.Errorf("error encoding payload: %w", err)
	}
	log.Printf("Encoded service announcement payload (%d bytes)", len(payload))

	opReturnScript, err := txscript.NewScriptBuilder().AddOp(txscript.OP_RETURN).AddData(payload).Script()
	if err != nil {
		return fmt.Errorf("error creating OP_RETURN script: %w", err)
	}

	feePerKb, err := estimateDynamicFeePerKbWithMode(context.Background(), client, int64(feeTargetBlocks), FeeMode(feeMode), cfg)
	if err != nil {
		return fmt.Errorf("failed to determine dynamic fee: %w", err)
	}

	estimatedSize := 250
	requiredFee := feePerKb * uint64(estimatedSize) / 1000

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

	tx.AddTxOut(wire.NewTxOut(0, opReturnScript))
	tx.AddTxOut(wire.NewTxOut(int64(changeAmount), changeScript))

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

	log.Printf("Successfully broadcasted announcement transaction: %s\n", txHash)
	return nil
}

func AnnounceHeartbeat(client *rpcclient.Client, pubKey *btcec.PublicKey, flags uint8, addressType string, cfg FeeConfig) error {
	payload, err := protocol.EncodeHeartbeatPayload(pubKey, flags)
	if err != nil {
		return fmt.Errorf("error encoding heartbeat payload: %w", err)
	}
	opReturnScript, err := txscript.NewScriptBuilder().AddOp(txscript.OP_RETURN).AddData(payload).Script()
	if err != nil {
		return fmt.Errorf("error creating OP_RETURN script: %w", err)
	}

	feePerKb, err := estimateDynamicFeePerKb(context.Background(), client, 6, cfg)
	if err != nil {
		return fmt.Errorf("failed to determine dynamic fee: %w", err)
	}
	requiredFee := feePerKb * 250 / 1000
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
	log.Printf("Successfully broadcasted provider heartbeat transaction: %s\n", txHash)
	return nil
}

func AnnouncePriceUpdate(client *rpcclient.Client, pubKey *btcec.PublicKey, newPrice uint64, feeTargetBlocks int, feeMode string, addressType string, cfg FeeConfig) error {
	payload, err := protocol.EncodePriceUpdatePayload(pubKey, newPrice)
	if err != nil {
		return fmt.Errorf("error encoding price update payload: %w", err)
	}

	opReturnScript, err := txscript.NewScriptBuilder().AddOp(txscript.OP_RETURN).AddData(payload).Script()
	if err != nil {
		return fmt.Errorf("error creating OP_RETURN script: %w", err)
	}

	feePerKb, err := estimateDynamicFeePerKbWithMode(context.Background(), client, int64(feeTargetBlocks), FeeMode(feeMode), cfg)
	if err != nil {
		return fmt.Errorf("failed to determine dynamic fee: %w", err)
	}

	estimatedSize := 250
	requiredFee := feePerKb * uint64(estimatedSize) / 1000

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

	signedTx, complete, err := client.SignRawTransactionWithWallet(tx)
	if err != nil || !complete {
		return fmt.Errorf("error signing transaction: %w", err)
	}

	txHash, err := sendRawTransaction(client, signedTx)
	if err != nil {
		return fmt.Errorf("error sending transaction: %w", err)
	}

	log.Printf("Successfully broadcasted price update transaction: %s\n", txHash)
	return nil
}

func AnnounceRating(client *rpcclient.Client, providerPubKey *btcec.PublicKey, clientPrivKey *btcec.PrivateKey, score uint8, source string, feeTargetBlocks int, feeMode string, addressType string, cfg FeeConfig) error {
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

	feePerKb, err := estimateDynamicFeePerKbWithMode(context.Background(), client, int64(feeTargetBlocks), FeeMode(feeMode), cfg)
	if err != nil {
		return fmt.Errorf("failed to determine dynamic fee: %w", err)
	}

	requiredFee := feePerKb * 250 / 1000

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

	log.Printf("Successfully broadcasted rating transaction: %s (provider=%s, score=%d)\n", txHash, hex.EncodeToString(providerPubKey.SerializeCompressed()), score)
	return nil
}

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

	buf := make([]byte, 128)
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

type PaymentMonitorCfg struct {
	Price                 uint64
	PricingMethod         string
	TimeUnitSecs          uint32
	DataUnitBytes         uint32
	MaxSessionSecs        int
	RequiredConfirmations int
}

func MonitorPayments(ctx context.Context, client *rpcclient.Client, authManager *auth.AuthManager, pmCfg PaymentMonitorCfg, pollingInterval time.Duration) {
	var lastBlockHash string
	payments := newPaymentTracker()

	info, err := withRetry(ctx, "GetBlockChainInfo", 5, 1*time.Second, func() (*btcjson.GetBlockChainInfoResult, error) {
		return client.GetBlockChainInfo()
	})
	if err == nil {
		lastBlockHash = info.BestBlockHash
	} else {
		log.Printf("Warning: could not initialize monitor from best block: %v", err)
	}

	log.Println("Starting payment monitor (polling mode)...")
	runPaymentMonitorPolling(ctx, client, authManager, pmCfg, lastBlockHash, payments, pollingInterval)
}

func runPaymentMonitorPolling(ctx context.Context, client *rpcclient.Client, authManager *auth.AuthManager, pmCfg PaymentMonitorCfg, lastBlockHash string, payments *paymentTracker, interval time.Duration) {
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
				var lastBlock *chainhash.Hash
				if lastBlockHash != "" {
					lastBlock, _ = chainhash.NewHashFromStr(lastBlockHash)
				}
				return client.ListSinceBlock(lastBlock)
			})
			if err != nil {
				log.Printf("Error checking payments (polling): %v", err)
				continue
			}
			for _, tx := range result.Transactions {
				processTxForPayment(ctx, client, tx.TxID, tx.Amount, tx.Confirmations, authManager, pmCfg, payments)
			}
			lastBlockHash = result.LastBlock
		}
	}
}

func processBlockForPayments(ctx context.Context, client *rpcclient.Client, block *btcjson.GetBlockVerboseTxResult, authManager *auth.AuthManager, pmCfg PaymentMonitorCfg, payments *paymentTracker) {
	for _, tx := range block.Tx {
		txHash, err := chainhash.NewHashFromStr(tx.Txid)
		if err != nil {
			continue
		}
		txDetails, err := client.GetTransaction(txHash)
		if err == nil && txDetails.Amount > 0 {
			processTxForPayment(ctx, client, tx.Txid, txDetails.Amount, int64(txDetails.Confirmations), authManager, pmCfg, payments)
		}
	}
}

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
	if len(addresses) == 0 {
		addr, err := client.GetRawChangeAddress("")
		if err == nil {
			addresses = append(addresses, addr.String())
		}
	}
	return addresses, nil
}

func processTxForPayment(ctx context.Context, client *rpcclient.Client, txid string, amount float64, confirmations int64, authManager *auth.AuthManager, pmCfg PaymentMonitorCfg, payments *paymentTracker) {
	if confirmations < 0 {
		payments.handleRemovedTx(txid, authManager)
		return
	}

	minConfirmations := pmCfg.RequiredConfirmations
	if minConfirmations <= 0 {
		minConfirmations = 1
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
			totalToProvider += uint64(math.Round(vout.Value * 1e8))
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

	var duration time.Duration
	var dataQuota uint64

	switch strings.ToLower(pmCfg.PricingMethod) {
	case "session", "":
		duration = 24 * time.Hour
		if pmCfg.MaxSessionSecs > 0 {
			maxDur := time.Duration(pmCfg.MaxSessionSecs) * time.Second
			if maxDur < duration {
				duration = maxDur
			}
		}
	case "time":
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
		if pmCfg.Price == 0 {
			log.Printf("Payment %s: price is zero, cannot calculate data units", txid)
			return
		}
		units := totalToProvider / pmCfg.Price
		dataQuota = units * uint64(pmCfg.DataUnitBytes)
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
