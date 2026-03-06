package main

import (
	"encoding/json"
	"os"
	"time"
)

const historyFilePath = "payment_history.json"

type PaymentRecord struct {
	TxID      string    `json:"txid"`
	Provider  string    `json:"provider"`
	Amount    uint64    `json:"amount"`
	Timestamp time.Time `json:"timestamp"`
}

func SavePaymentRecord(record PaymentRecord) error {
	history, _ := LoadHistory() // Ignore error, start fresh if fails
	history = append(history, record)

	file, err := os.Create(historyFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(history)
}

func LoadHistory() ([]PaymentRecord, error) {
	file, err := os.Open(historyFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []PaymentRecord{}, nil
		}
		return nil, err
	}
	defer file.Close()

	var history []PaymentRecord
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&history); err != nil {
		return nil, err
	}
	return history, nil
}