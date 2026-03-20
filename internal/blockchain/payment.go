package blockchain

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
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

// GetProviderPaymentAddress inspects the announcement transaction to find the
// address that funded it. This is considered the provider's payment address.
// If params is nil (unknown blockchain), returns the raw address string as a
// stringAddress wrapper that implements btcutil.Address.
func GetProviderPaymentAddress(client *rpcclient.Client, announcementTxID string, params *chaincfg.Params) (btcutil.Address, error) {
	txHash, err := chainhash.NewHashFromStr(announcementTxID)
	if err != nil {
		return nil, fmt.Errorf("invalid announcement txid: %w", err)
	}

	// Get the announcement transaction itself.
	txVerbose, err := withRetry(context.Background(), "GetRawTransactionVerbose(announcement)", 4, 500*time.Millisecond, func() (*btcjson.TxRawResult, error) {
		return client.GetRawTransactionVerbose(txHash)
	})
	if err != nil {
		return nil, fmt.Errorf("could not get raw announcement transaction: %w", err)
	}

	if len(txVerbose.Vin) == 0 {
		return nil, fmt.Errorf("announcement transaction has no inputs")
	}

	// Use the first input to find the source address.
	vin := txVerbose.Vin[0]
	if vin.IsCoinBase() {
		return nil, fmt.Errorf("announcement transaction input is a coinbase, cannot determine address")
	}

	// Get the transaction that the announcement is spending from.
	prevTxHash, err := chainhash.NewHashFromStr(vin.Txid)
	if err != nil {
		return nil, fmt.Errorf("invalid previous txid: %w", err)
	}
	prevTxVerbose, err := withRetry(context.Background(), "GetRawTransactionVerbose(previous)", 4, 500*time.Millisecond, func() (*btcjson.TxRawResult, error) {
		return client.GetRawTransactionVerbose(prevTxHash)
	})
	if err != nil {
		return nil, fmt.Errorf("could not get raw previous transaction: %w", err)
	}

	if int(vin.Vout) >= len(prevTxVerbose.Vout) {
		return nil, fmt.Errorf("announcement transaction vin refers to out-of-bounds vout")
	}

	// Get the output that was spent.
	spentVout := prevTxVerbose.Vout[vin.Vout]
	if len(spentVout.ScriptPubKey.Addresses) == 0 {
		return nil, fmt.Errorf("previous transaction output has no addresses")
	}

	// The first address is the provider's payment address.
	rawAddr := spentVout.ScriptPubKey.Addresses[0]
	if params == nil {
		// Unknown blockchain - return raw address string wrapped as stringAddress
		return newStringAddress(rawAddr), nil
	}
	addr, err := btcutil.DecodeAddress(rawAddr, params)
	if err != nil {
		return nil, fmt.Errorf("could not decode provider address: %w", err)
	}
	return addr, nil
}

// stringAddress wraps a raw address string to implement btcutil.Address interface.
// This is used when chain params are unknown (blockchain-agnostic mode).
type stringAddress string

func (s stringAddress) String() string                 { return string(s) }
func (s stringAddress) EncodeAddress() string          { return string(s) }
func (s stringAddress) ScriptAddress() []byte          { return nil }
func (s stringAddress) IsValid() bool                  { return len(s) > 0 }
func (s stringAddress) IsForNet(*chaincfg.Params) bool { return true }

// newStringAddress creates a stringAddress wrapper from a raw address string.
func newStringAddress(addr string) btcutil.Address {
	return stringAddress(addr)
}

// selectCoins selects a set of unspent transaction outputs (UTXOs) that sum up
// to at least the target amount. It returns the selected UTXOs, the total value,
// and an error if insufficient funds are found.
func selectCoins(client *rpcclient.Client, targetAmount btcutil.Amount) ([]btcjson.ListUnspentResult, btcutil.Amount, error) {
	// List all unspent outputs.
	// In a real application, you might want to filter by min/max confirmations.
	unspent, err := client.ListUnspent()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list unspent outputs: %w", err)
	}

	return deterministicSelectCoins(unspent, targetAmount)
}

// deterministicSelectCoins chooses coins with a deterministic strategy:
// 1) single-UTXO exact match
// 2) smallest single UTXO that covers target
// 3) ascending accumulation (smallest-first) until target
func deterministicSelectCoins(unspent []btcjson.ListUnspentResult, targetAmount btcutil.Amount) ([]btcjson.ListUnspentResult, btcutil.Amount, error) {
	type entry struct {
		utxo   btcjson.ListUnspentResult
		amount btcutil.Amount
	}
	entries := make([]entry, 0, len(unspent))
	for _, u := range unspent {
		amount, _ := btcutil.NewAmount(u.Amount)
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

	// Prefer exact match.
	for _, e := range entries {
		if e.amount == targetAmount {
			return []btcjson.ListUnspentResult{e.utxo}, e.amount, nil
		}
	}

	// Then smallest single coin over target.
	for _, e := range entries {
		if e.amount > targetAmount {
			return []btcjson.ListUnspentResult{e.utxo}, e.amount, nil
		}
	}

	// Finally accumulate largest first to minimize number of inputs and fee impact.
	var selected []btcjson.ListUnspentResult
	var total btcutil.Amount
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		selected = append(selected, e.utxo)
		total += e.amount
		if total >= targetAmount {
			return selected, total, nil
		}
	}

	return nil, 0, fmt.Errorf("insufficient funds: have %v, need %v", total, targetAmount)
}

// SendPayment sends the specified amount to the provider's address.
func SendPayment(client *rpcclient.Client, providerAddress btcutil.Address, amountSatoshis uint64, clientPubKey *btcec.PublicKey) (*chainhash.Hash, error) {
	// 1. Create the OP_RETURN script with the client's public key.
	paymentPayload, err := protocol.EncodePaymentPayload(clientPubKey)
	if err != nil {
		return nil, fmt.Errorf("could not encode payment payload: %w", err)
	}
	opReturnScript, err := txscript.NewScriptBuilder().AddOp(txscript.OP_RETURN).AddData(paymentPayload).Script()
	if err != nil {
		return nil, fmt.Errorf("could not create OP_RETURN script: %w", err)
	}

	// 2. Create the payment output to the provider.
	providerScript, err := txscript.PayToAddrScript(providerAddress)
	if err != nil {
		return nil, fmt.Errorf("could not create provider payment script: %w", err)
	}
	providerOutput := wire.NewTxOut(int64(amountSatoshis), providerScript)

	// 3. Estimate Fee
	// We estimate fee for a target of 6 blocks (approx 1 hour).
	feePerKb, err := estimateDynamicFeePerKb(context.Background(), client, 6)
	if err != nil {
		return nil, fmt.Errorf("failed to determine dynamic fee: %w", err)
	}

	// Estimated transaction size (P2PKH input ~148 bytes, P2PKH output ~34 bytes, OP_RETURN ~40-50 bytes)
	// We'll assume 1 input and 3 outputs (provider, op_return, change) for initial estimation.
	// 148 + 34 + 50 + 34 + 10 (overhead) ~= 276 bytes.
	// Let's be conservative and estimate 300 bytes per input/output set for coin selection.
	estimatedSize := 300
	requiredFee := btcutil.Amount(float64(feePerKb) * float64(estimatedSize) / 1000.0)

	// 4. Coin Selection
	targetAmount := btcutil.Amount(amountSatoshis) + requiredFee
	utxos, totalInput, err := selectCoins(client, targetAmount)
	if err != nil {
		return nil, err
	}

	// 5. Create Transaction Inputs
	tx := wire.NewMsgTx(wire.TxVersion)
	for _, utxo := range utxos {
		txHash, err := chainhash.NewHashFromStr(utxo.TxID)
		if err != nil {
			return nil, fmt.Errorf("invalid txid %s: %w", utxo.TxID, err)
		}
		outPoint := wire.NewOutPoint(txHash, utxo.Vout)
		txIn := wire.NewTxIn(outPoint, nil, nil)
		tx.AddTxIn(txIn)
	}

	// 6. Create Outputs
	opReturnOutput := wire.NewTxOut(0, opReturnScript)
	tx.AddTxOut(providerOutput)
	tx.AddTxOut(opReturnOutput)

	// 7. Calculate Change
	// Recalculate fee based on actual transaction size (signed size approximation)
	// We can't know exact signed size before signing, but we can approximate.
	// Or we can just use the conservative estimate from before.
	changeAmount := totalInput - btcutil.Amount(amountSatoshis) - requiredFee

	changeAddr, err := client.GetRawChangeAddress("")
	if err != nil {
		return nil, fmt.Errorf("failed to get change address: %w", err)
	}
	changeScript, err := txscript.PayToAddrScript(changeAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create change script: %w", err)
	}
	changeOutput := wire.NewTxOut(int64(changeAmount), changeScript)
	tx.AddTxOut(changeOutput)

	// 8. Sign and send.
	signedTx, complete, err := client.SignRawTransactionWithWallet(tx)
	if err != nil || !complete {
		return nil, fmt.Errorf("failed to sign payment transaction: %w", err)
	}

	txHash, err := client.SendRawTransaction(signedTx, true)
	if err != nil {
		return nil, err
	}

	// Record payment in history
	record := history.PaymentRecord{
		TxID:      txHash.String(),
		Provider:  providerAddress.String(),
		Amount:    amountSatoshis,
		Timestamp: time.Now(),
	}
	if err := history.SavePaymentRecord(record); err != nil {
		fmt.Printf("Warning: Failed to save payment history: %v\n", err)
	}

	return txHash, nil
}

// WaitForConfirmations waits for a transaction to reach the specified number of confirmations.
// It polls the blockchain and returns when the threshold is met or the context is cancelled.
func WaitForConfirmations(ctx context.Context, client *rpcclient.Client, txHash *chainhash.Hash, requiredConfirmations int, pollInterval time.Duration) (int64, error) {
	if client == nil {
		return 0, errors.New("client is nil")
	}
	if txHash == nil {
		return 0, errors.New("txHash is nil")
	}
	if requiredConfirmations <= 0 {
		requiredConfirmations = 1
	}
	if pollInterval <= 0 {
		pollInterval = 10 * time.Second
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-ticker.C:
			tx, err := client.GetTransaction(txHash)
			if err != nil {
				continue
			}
			if tx.Confirmations >= int64(requiredConfirmations) {
				return tx.Confirmations, nil
			}
		}
	}
}
