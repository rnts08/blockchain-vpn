package history

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"
	"time"
)

func TestPaymentRecordJSON(t *testing.T) {
	record := PaymentRecord{
		TxID:      "abc123",
		Provider:  "bc1qxyz...",
		Amount:    1000,
		Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded PaymentRecord
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.TxID != record.TxID {
		t.Errorf("expected txid %q, got %q", record.TxID, decoded.TxID)
	}
	if decoded.Provider != record.Provider {
		t.Errorf("expected provider %q, got %q", record.Provider, decoded.Provider)
	}
	if decoded.Amount != record.Amount {
		t.Errorf("expected amount %d, got %d", record.Amount, decoded.Amount)
	}
}

func TestDecodeHistoryEmpty(t *testing.T) {
	empty := []byte("[]")
	records, err := decodeHistory(bytes.NewReader(empty))
	if err != nil {
		t.Fatalf("decodeHistory failed: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
}

func TestDecodeHistoryMultiple(t *testing.T) {
	data := []byte(`[
		{"txid": "tx1", "provider": "addr1", "amount": 1000, "timestamp": "2024-01-15T10:00:00Z"},
		{"txid": "tx2", "provider": "addr2", "amount": 2000, "timestamp": "2024-01-16T10:00:00Z"}
	]`)
	records, err := decodeHistory(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decodeHistory failed: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].TxID != "tx1" {
		t.Errorf("expected tx1, got %s", records[0].TxID)
	}
	if records[1].Amount != 2000 {
		t.Errorf("expected 2000, got %d", records[1].Amount)
	}
}

func TestDecodeHistoryEOF(t *testing.T) {
	records, err := decodeHistory(bytes.NewReader([]byte("")))
	if err != nil {
		t.Fatalf("decodeHistory failed: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
}

func TestDecodeHistoryInvalid(t *testing.T) {
	invalid := []byte(`{invalid json}`)
	_, err := decodeHistory(bytes.NewReader(invalid))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

type fakeReader struct {
	data []byte
	pos  int
}

func (r *fakeReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func TestHistoryFilePath(t *testing.T) {
	if historyFileName != "history.json" {
		t.Errorf("expected historyFileName 'history.json', got %q", historyFileName)
	}
	if legacyHistoryFileName != "payment_history.json" {
		t.Errorf("expected legacyHistoryFileName 'payment_history.json', got %q", legacyHistoryFileName)
	}
}

func TestPaymentRecordFields(t *testing.T) {
	record := PaymentRecord{
		TxID:      "tx123",
		Provider:  "provider_address",
		Amount:    500,
		Timestamp: time.Now(),
	}

	if record.TxID != "tx123" {
		t.Errorf("TxID mismatch")
	}
	if record.Provider != "provider_address" {
		t.Errorf("Provider mismatch")
	}
	if record.Amount != 500 {
		t.Errorf("Amount mismatch")
	}
	if record.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestPaymentRecordZeroAmount(t *testing.T) {
	record := PaymentRecord{
		TxID:      "tx_zero",
		Provider:  "provider",
		Amount:    0,
		Timestamp: time.Now(),
	}

	if record.Amount != 0 {
		t.Errorf("expected 0 amount, got %d", record.Amount)
	}
}

func TestPaymentRecordLargeAmount(t *testing.T) {
	record := PaymentRecord{
		TxID:      "tx_large",
		Provider:  "provider",
		Amount:    1_000_000_000,
		Timestamp: time.Now(),
	}

	if record.Amount != 1_000_000_000 {
		t.Errorf("expected 1_000_000_000, got %d", record.Amount)
	}
}

func TestDecodeHistoryPartial(t *testing.T) {
	partial := []byte(`[{"txid": "tx_partial", "provider": "addr", "amount": 100, "timestamp": "2024-01-15T10:00:00Z"}]`)
	records, err := decodeHistory(bytes.NewReader(partial))
	if err != nil {
		t.Fatalf("decodeHistory failed: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].TxID != "tx_partial" {
		t.Errorf("expected tx_partial, got %s", records[0].TxID)
	}
}

func TestDecodeHistoryMissingFields(t *testing.T) {
	missing := []byte(`[{"txid": "tx_missing"}]`)
	records, err := decodeHistory(bytes.NewReader(missing))
	if err != nil {
		t.Fatalf("decodeHistory failed: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].TxID != "tx_missing" {
		t.Errorf("expected tx_missing, got %s", records[0].TxID)
	}
}
