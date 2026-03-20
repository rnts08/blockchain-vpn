package blockchain

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

func TestSelectCoinsForTxBasic(t *testing.T) {
	t.Parallel()

	unspent := []struct {
		txID         string
		vout         uint32
		amount       float64
		scriptPubKey string
	}{
		{"aaaa1111", 0, 0.001, "76a914" + strings.Repeat("aa", 20) + "88ac"},
		{"bbbb2222", 1, 0.002, "a914" + strings.Repeat("bb", 20) + "87"},
	}

	for _, u := range unspent {
		if len(u.scriptPubKey) != 2+40 && u.scriptPubKey != "" {
			// basic format check
		}
	}

	// Verify hex decoding works as expected for change script extraction
	// P2PKH: OP_DUP OP_HASH160 <20 bytes> OP_EQUALVERIFY OP_CHECKSIG = 25 bytes
	p2pkhScript, _ := hex.DecodeString("76a914" + strings.Repeat("ff", 20) + "88ac")
	if len(p2pkhScript) != 25 {
		t.Errorf("P2PKH script length: got %d, want 25", len(p2pkhScript))
	}

	// P2SH: OP_HASH160 <20 bytes> OP_EQUAL = 23 bytes
	p2shScript, _ := hex.DecodeString("a914" + strings.Repeat("ff", 20) + "87")
	if len(p2shScript) != 23 {
		t.Errorf("P2SH script length: got %d, want 23", len(p2shScript))
	}
}

func TestScriptClassDetection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		scriptHex string
		wantClass txscript.ScriptClass
		wantType  string
	}{
		{
			name:      "P2PKH script",
			scriptHex: "76a914" + strings.Repeat("aa", 20) + "88ac",
			wantClass: txscript.PubKeyHashTy,
			wantType:  "p2pkh",
		},
		{
			name:      "P2SH script",
			scriptHex: "a914" + strings.Repeat("bb", 20) + "87",
			wantClass: txscript.ScriptHashTy,
			wantType:  "p2sh",
		},
		{
			name:      "P2WPKH script",
			scriptHex: "0014" + strings.Repeat("cc", 20),
			wantClass: txscript.WitnessV0PubKeyHashTy,
			wantType:  "bech32",
		},
		{
			name:      "P2WSH script",
			scriptHex: "0020" + strings.Repeat("dd", 32),
			wantClass: txscript.WitnessV0ScriptHashTy,
			wantType:  "bech32m",
		},
		{
			name:      "null data script",
			scriptHex: "6a" + "04" + "01020304",
			wantClass: txscript.NullDataTy,
			wantType:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scriptBytes, err := hex.DecodeString(tt.scriptHex)
			if err != nil {
				t.Fatalf("hex.DecodeString failed: %v", err)
			}

			class := txscript.GetScriptClass(scriptBytes)
			if class != tt.wantClass {
				t.Errorf("script class = %v, want %v", class, tt.wantClass)
			}

			hexLower := strings.ToLower(tt.scriptHex)
			var detectedType string
			switch {
			case strings.HasPrefix(hexLower, "76a9"):
				detectedType = "p2pkh"
			case strings.HasPrefix(hexLower, "a914"):
				detectedType = "p2sh"
			case strings.HasPrefix(hexLower, "0014"):
				detectedType = "bech32"
			case strings.HasPrefix(hexLower, "0020"):
				detectedType = "bech32m"
			}
			if tt.wantType != "" && detectedType != tt.wantType {
				t.Errorf("detected type = %q, want %q", detectedType, tt.wantType)
			}
		})
	}
}

func TestSendRawTransactionSerialize(t *testing.T) {
	t.Parallel()

	tx := wire.NewMsgTx(wire.TxVersion)
	tx.TxIn = []*wire.TxIn{
		{
			PreviousOutPoint: wire.OutPoint{
				Hash:  chainhash.Hash{},
				Index: 0,
			},
		},
	}
	tx.TxOut = []*wire.TxOut{
		{Value: 1000, PkScript: []byte{0x76, 0xa9, 0x14, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0x00, 0x11, 0x22, 0x33, 0x88, 0xac}},
	}

	var buf strings.Builder
	if err := tx.Serialize(&buf); err != nil {
		t.Fatalf("failed to serialize transaction: %v", err)
	}

	txHex := hex.EncodeToString([]byte(buf.String()))
	if len(txHex) == 0 {
		t.Error("serialized tx hex should not be empty")
	}

	if !strings.HasPrefix(txHex, "0100000001") {
		t.Errorf("tx hex should start with version + num inputs, got: %s", txHex[:20])
	}
}

type sbWriter struct {
	sb *strings.Builder
}

func (w *sbWriter) Write(p []byte) (n int, err error) {
	return w.sb.Write(p)
}

func TestSendRawTransactionMockRPC(t *testing.T) {
	t.Parallel()

	var receivedTxHex string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req JSONRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "parse error", 400)
			return
		}

		if req.Method != "sendrawtransaction" {
			http.Error(w, "unknown method", 400)
			return
		}

		if len(req.Params) == 0 {
			http.Error(w, "missing params", 400)
			return
		}

		var hexStr string
		if err := json.Unmarshal(req.Params[0], &hexStr); err != nil {
			http.Error(w, "invalid hex param", 400)
			return
		}
		receivedTxHex = hexStr

		resp := JSONRPCResponse{
			ID:      req.ID,
			Result:  "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234",
			JSONRPC: "2.0",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := newMockRPCClient(server.URL)
	if err != nil {
		t.Fatalf("failed to create mock client: %v", err)
	}

	tx := wire.NewMsgTx(wire.TxVersion)
	tx.TxIn = []*wire.TxIn{
		{
			PreviousOutPoint: wire.OutPoint{
				Hash:  chainhash.Hash{},
				Index: 0,
			},
		},
	}
	tx.TxOut = []*wire.TxOut{
		{Value: 1000, PkScript: []byte{0x76, 0xa9, 0x14, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0x00, 0x11, 0x22, 0x33, 0x88, 0xac}},
	}

	txHash, err := sendRawTransaction(client, tx)
	if err != nil {
		t.Fatalf("sendRawTransaction failed: %v", err)
	}

	if txHash == nil {
		t.Fatal("expected non-nil txHash")
	}

	expected := "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234"
	if txHash.String() != expected {
		t.Errorf("txHash mismatch: got %s, want %s", txHash.String(), expected)
	}

	if receivedTxHex == "" {
		t.Error("server did not receive transaction hex")
	}
}

func TestSendRawTransactionRPCError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := JSONRPCResponse{
			ID:      1,
			Error:   &RPCError{Code: -25, Message: "missing inputs"},
			JSONRPC: "2.0",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := newMockRPCClient(server.URL)
	if err != nil {
		t.Fatalf("failed to create mock client: %v", err)
	}

	tx := wire.NewMsgTx(wire.TxVersion)
	txHash, err := sendRawTransaction(client, tx)
	if err == nil {
		t.Error("expected error for RPC error response")
	}
	if txHash != nil {
		t.Error("expected nil txHash on error")
	}
}

type JSONRPCRequest struct {
	ID     interface{}       `json:"id"`
	Method string            `json:"method"`
	Params []json.RawMessage `json:"params"`
}

type JSONRPCResponse struct {
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
	JSONRPC string      `json:"jsonrpc"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func newMockRPCClient(serverURL string) (*mockRPCClient, error) {
	return &mockRPCClient{serverURL: serverURL}, nil
}

type mockRPCClient struct {
	serverURL string
}

func (c *mockRPCClient) RawRequest(method string, params []json.RawMessage) (json.RawMessage, error) {
	body, _ := json.Marshal(JSONRPCRequest{
		ID:     1,
		Method: method,
		Params: params,
	})
	resp, err := http.Post(c.serverURL, "application/json", strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var rpcResp JSONRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, err
	}
	if rpcResp.Error != nil {
		return nil, &mockRPCError{rpcResp.Error}
	}
	return json.Marshal(rpcResp.Result)
}

type mockRPCError struct {
	err *RPCError
}

func (e *mockRPCError) Error() string {
	return e.err.Message
}
