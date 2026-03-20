package blockchain

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"

	"blockchain-vpn/internal/history"
	"blockchain-vpn/internal/protocol"
)

type PaymentDetails struct {
	ProviderAddress string
	AmountSats      uint64
	ClientPubKey    []byte
	TxHash          string
}

func VerifyPaymentInput(advertisedPrice uint64, amountSatoshis uint64) error {
	if amountSatoshis < advertisedPrice {
		return fmt.Errorf("payment amount %d sats is less than advertised price %d sats", amountSatoshis, advertisedPrice)
	}
	return nil
}

func GetPaymentVerification(advertisedPrice uint64, amountSatoshis uint64) (uint64, error) {
	if err := VerifyPaymentInput(advertisedPrice, amountSatoshis); err != nil {
		return 0, err
	}
	return amountSatoshis, nil
}

func GetProviderPaymentAddress(client *rpcclient.Client, announcementTxID string) (string, error) {
	txHash, err := chainhash.NewHashFromStr(announcementTxID)
	if err != nil {
		return "", fmt.Errorf("invalid announcement txid: %w", err)
	}

	txVerbose, err := withRetry(context.Background(), "GetRawTransactionVerbose(announcement)", 4, 500*time.Millisecond, func() (*btcjson.TxRawResult, error) {
		return client.GetRawTransactionVerbose(txHash)
	})
	if err != nil {
		return "", fmt.Errorf("could not get raw announcement transaction: %w", err)
	}

	if len(txVerbose.Vin) == 0 {
		return "", fmt.Errorf("announcement transaction has no inputs")
	}

	vin := txVerbose.Vin[0]
	if vin.IsCoinBase() {
		return "", fmt.Errorf("announcement transaction input is a coinbase, cannot determine address")
	}

	prevTxHash, err := chainhash.NewHashFromStr(vin.Txid)
	if err != nil {
		return "", fmt.Errorf("invalid previous txid: %w", err)
	}
	prevTxVerbose, err := withRetry(context.Background(), "GetRawTransactionVerbose(previous)", 4, 500*time.Millisecond, func() (*btcjson.TxRawResult, error) {
		return client.GetRawTransactionVerbose(prevTxHash)
	})
	if err != nil {
		return "", fmt.Errorf("could not get raw previous transaction: %w", err)
	}

	if int(vin.Vout) >= len(prevTxVerbose.Vout) {
		return "", fmt.Errorf("announcement transaction vin refers to out-of-bounds vout")
	}

	spentVout := prevTxVerbose.Vout[vin.Vout]
	if len(spentVout.ScriptPubKey.Addresses) == 0 {
		return "", fmt.Errorf("previous transaction output has no addresses")
	}

	return spentVout.ScriptPubKey.Addresses[0], nil
}

type txBroadcaster interface {
	RawRequest(method string, params []json.RawMessage) (json.RawMessage, error)
}

func sendRawTransaction(client txBroadcaster, tx *wire.MsgTx) (string, error) {
	var buf bytes.Buffer
	if err := tx.Serialize(&buf); err != nil {
		return "", fmt.Errorf("failed to serialize transaction: %w", err)
	}
	txHex := hex.EncodeToString(buf.Bytes())

	params := []json.RawMessage{json.RawMessage(`"` + txHex + `"`)}
	result, err := client.RawRequest("sendrawtransaction", params)
	if err != nil {
		return "", fmt.Errorf("sendrawtransaction failed: %w", err)
	}

	var txidStr string
	if err := json.Unmarshal(result, &txidStr); err != nil {
		return "", fmt.Errorf("failed to parse txid from sendrawtransaction response: %w", err)
	}
	return txidStr, nil
}

func selectCoins(client *rpcclient.Client, targetAmount uint64) ([]btcjson.ListUnspentResult, uint64, error) {
	unspent, err := client.ListUnspent()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list unspent outputs: %w", err)
	}

	return deterministicSelectCoins(unspent, targetAmount)
}

func deterministicSelectCoins(unspent []btcjson.ListUnspentResult, targetAmount uint64) ([]btcjson.ListUnspentResult, uint64, error) {
	type entry struct {
		utxo   btcjson.ListUnspentResult
		amount uint64
	}
	entries := make([]entry, 0, len(unspent))
	for _, u := range unspent {
		amount := uint64(math.Round(u.Amount * 1e8))
		entries = append(entries, entry{utxo: u, amount: amount})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].amount != entries[j].amount {
			return entries[i].amount < entries[j].amount
		}
		if entries[i].utxo.TxID != entries[j].utxo.TxID {
			return entries[i].utxo.TxID < entries[j].utxo.TxID
		}
		return entries[i].utxo.Vout < entries[j].utxo.Vout
	})

	for _, e := range entries {
		if e.amount == targetAmount {
			return []btcjson.ListUnspentResult{e.utxo}, e.amount, nil
		}
	}

	for _, e := range entries {
		if e.amount > targetAmount {
			return []btcjson.ListUnspentResult{e.utxo}, e.amount, nil
		}
	}

	var selected []btcjson.ListUnspentResult
	var total uint64
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		selected = append(selected, e.utxo)
		total += e.amount
		if total >= targetAmount {
			return selected, total, nil
		}
	}

	return nil, 0, fmt.Errorf("insufficient funds: have %d, need %d", total, targetAmount)
}

func selectCoinsForTx(client *rpcclient.Client, targetAmount uint64) ([]btcjson.ListUnspentResult, uint64, []byte, error) {
	utxos, totalInput, err := selectCoins(client, targetAmount)
	if err != nil {
		return nil, 0, nil, err
	}
	if len(utxos) == 0 || utxos[0].ScriptPubKey == "" {
		return nil, 0, nil, fmt.Errorf("no usable UTXOs with scriptPubKey found")
	}
	changeScript, err := hex.DecodeString(utxos[0].ScriptPubKey)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("failed to decode change scriptPubKey: %w", err)
	}
	return utxos, totalInput, changeScript, nil
}

func SendPayment(client *rpcclient.Client, providerAddress string, amountSatoshis uint64, clientPubKey *btcec.PublicKey, addressType string, cfg FeeConfig) (string, error) {
	paymentPayload, err := protocol.EncodePaymentPayload(clientPubKey)
	if err != nil {
		return "", fmt.Errorf("could not encode payment payload: %w", err)
	}
	opReturnScript, err := txscript.NewScriptBuilder().AddOp(txscript.OP_RETURN).AddData(paymentPayload).Script()
	if err != nil {
		return "", fmt.Errorf("could not create OP_RETURN script: %w", err)
	}

	providerScript, err := hex.DecodeString(providerAddress)
	if err != nil {
		providerScript = []byte(providerAddress)
	}

	feePerKb, err := estimateDynamicFeePerKb(context.Background(), client, 6, cfg)
	if err != nil {
		return "", fmt.Errorf("failed to determine dynamic fee: %w", err)
	}

	estimatedSize := 300
	requiredFee := feePerKb * uint64(estimatedSize) / 1000

	targetAmount := amountSatoshis + requiredFee
	utxos, totalInput, changeScript, err := selectCoinsForTx(client, targetAmount)
	if err != nil {
		return "", err
	}

	tx := wire.NewMsgTx(wire.TxVersion)
	for _, utxo := range utxos {
		txHash, err := chainhash.NewHashFromStr(utxo.TxID)
		if err != nil {
			return "", fmt.Errorf("invalid txid %s: %w", utxo.TxID, err)
		}
		outPoint := wire.NewOutPoint(txHash, utxo.Vout)
		txIn := wire.NewTxIn(outPoint, nil, nil)
		tx.AddTxIn(txIn)
	}

	tx.AddTxOut(wire.NewTxOut(int64(amountSatoshis), providerScript))
	tx.AddTxOut(wire.NewTxOut(0, opReturnScript))

	changeAmount := totalInput - amountSatoshis - requiredFee
	changeOutput := wire.NewTxOut(int64(changeAmount), changeScript)
	tx.AddTxOut(changeOutput)

	signedTx, complete, err := client.SignRawTransactionWithWallet(tx)
	if err != nil || !complete {
		return "", fmt.Errorf("failed to sign payment transaction: %w", err)
	}

	txHash, err := sendRawTransaction(client, signedTx)
	if err != nil {
		return "", err
	}

	record := history.PaymentRecord{
		TxID:      txHash,
		Provider:  providerAddress,
		Amount:    amountSatoshis,
		Timestamp: time.Now(),
	}
	if err := history.SavePaymentRecord(record); err != nil {
		fmt.Printf("Warning: Failed to save payment history: %v\n", err)
	}

	return txHash, nil
}

func WaitForConfirmations(ctx context.Context, client *rpcclient.Client, txHash string, requiredConfirmations int, pollInterval time.Duration) (int64, error) {
	if client == nil {
		return 0, errors.New("client is nil")
	}
	if txHash == "" {
		return 0, errors.New("txHash is empty")
	}
	if requiredConfirmations <= 0 {
		requiredConfirmations = 1
	}
	if pollInterval <= 0 {
		pollInterval = 10 * time.Second
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	txHashObj, err := chainhash.NewHashFromStr(txHash)
	if err != nil {
		return 0, fmt.Errorf("invalid txHash: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-ticker.C:
			tx, err := client.GetTransaction(txHashObj)
			if err != nil {
				continue
			}
			if tx.Confirmations >= int64(requiredConfirmations) {
				return tx.Confirmations, nil
			}
		}
	}
}
