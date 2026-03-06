package main

import (
	"fmt"
	"time"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// GetProviderPaymentAddress inspects the announcement transaction to find the
// address that funded it. This is considered the provider's payment address.
func GetProviderPaymentAddress(client *rpcclient.Client, announcementTxID string, params *chaincfg.Params) (btcutil.Address, error) {
	txHash, err := chainhash.NewHashFromStr(announcementTxID)
	if err != nil {
		return nil, fmt.Errorf("invalid announcement txid: %w", err)
	}

	// Get the announcement transaction itself.
	txVerbose, err := client.GetRawTransactionVerbose(txHash)
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
	prevTxVerbose, err := client.GetRawTransactionVerbose(prevTxHash)
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
	addr, err := btcutil.DecodeAddress(spentVout.ScriptPubKey.Addresses[0], params)
	if err != nil {
		return nil, fmt.Errorf("could not decode provider address: %w", err)
	}
	return addr, nil
}

// SendPayment sends the specified amount to the provider's address.
func SendPayment(client *rpcclient.Client, providerAddress btcutil.Address, amountSatoshis uint64, clientPubKey wgtypes.Key) (*chainhash.Hash, error) {
	// 1. Create the OP_RETURN script with the client's public key.
	paymentPayload, err := EncodePaymentPayload(clientPubKey)
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

	// 3. Create the full transaction.
	// This is a simplified example. A real implementation would perform more
	// sophisticated coin selection and fee estimation.
	unspent, err := client.ListUnspent()
	if err != nil || len(unspent) == 0 {
		return nil, fmt.Errorf("could not list unspent outputs or none available: %w", err)
	}
	utxo := unspent[0]
	txid, _ := chainhash.NewHashFromStr(utxo.TxID)

	txInput := wire.NewTxIn(&wire.OutPoint{Hash: *txid, Index: utxo.Vout}, nil, nil)
	opReturnOutput := wire.NewTxOut(0, opReturnScript)

	// 4. Create change output.
	fee := btcutil.Amount(10000) // Hardcoded fee
	inputAmount, _ := btcutil.NewAmount(utxo.Amount)
	changeAmount := inputAmount - btcutil.Amount(amountSatoshis) - fee

	changeAddr, _ := client.GetRawChangeAddress("")
	changeScript, _ := txscript.PayToAddrScript(changeAddr)
	changeOutput := wire.NewTxOut(int64(changeAmount), changeScript)

	tx := wire.NewMsgTx(wire.TxVersion)
	tx.AddTxIn(txInput)
	tx.AddTxOut(providerOutput)
	tx.AddTxOut(opReturnOutput)
	tx.AddTxOut(changeOutput)

	// 5. Sign and send.
	signedTx, complete, err := client.SignRawTransactionWithWallet(tx)
	if err != nil || !complete {
		return nil, fmt.Errorf("failed to sign payment transaction: %w", err)
	}

	txHash, err := client.SendRawTransaction(signedTx, true)
	if err != nil {
		return nil, err
	}

	// Record payment in history
	record := PaymentRecord{
		TxID:      txHash.String(),
		Provider:  providerAddress.String(),
		Amount:    amountSatoshis,
		Timestamp: time.Now(),
	}
	if err := SavePaymentRecord(record); err != nil {
		fmt.Printf("Warning: Failed to save payment history: %v\n", err)
	}

	return txHash, nil
}