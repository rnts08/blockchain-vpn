package history

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"blockchain-vpn/internal/config"
)

const historyFileName = "history.json"
const legacyHistoryFileName = "payment_history.json"

var historyMu sync.Mutex

type PaymentRecord struct {
	TxID      string    `json:"txid"`
	Provider  string    `json:"provider"`
	Amount    uint64    `json:"amount"`
	Timestamp time.Time `json:"timestamp"`
}

func getHistoryPath() (string, error) {
	configDir, err := config.AppConfigDir()
	if err != nil {
		return "", fmt.Errorf("could not get app config dir: %w", err)
	}
	return filepath.Join(configDir, historyFileName), nil
}

func SavePaymentRecord(record PaymentRecord) error {
	historyMu.Lock()
	defer historyMu.Unlock()

	records, err := loadHistoryInternal()
	// It's okay if the file doesn't exist yet.
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	records = append(records, record)

	path, err := getHistoryPath()
	if err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(records)
}

func LoadHistory() ([]PaymentRecord, error) {
	historyMu.Lock()
	defer historyMu.Unlock()
	return loadHistoryInternal()
}

// loadHistoryInternal is the non-locking version for internal use.
func loadHistoryInternal() ([]PaymentRecord, error) {
	path, err := getHistoryPath()
	if err != nil {
		return nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			if legacy, legacyErr := getLegacyHistoryPath(); legacyErr == nil {
				if lf, openLegacyErr := os.Open(legacy); openLegacyErr == nil {
					defer lf.Close()
					return decodeHistory(lf)
				}
			}
		}
		return nil, err
	}
	defer file.Close()
	return decodeHistory(file)
}

func decodeHistory(r io.Reader) ([]PaymentRecord, error) {
	var records []PaymentRecord
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&records); err != nil {
		// An empty file will cause an EOF error, which is fine.
		if err == io.EOF {
			return []PaymentRecord{}, nil
		}
		return nil, err
	}
	return records, nil
}

func getLegacyHistoryPath() (string, error) {
	configDir, err := config.AppConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, legacyHistoryFileName), nil
}
