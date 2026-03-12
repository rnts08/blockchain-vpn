package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
)

var (
	listenAddr = flag.String("listen", "localhost:18443", "Listen address")
	network    = flag.String("network", "regtest", "Network: regtest, testnet, mainnet")
	verbose    = flag.Bool("v", false, "Verbose logging")
)

type RPCServer struct {
	blockHeight int64
	mempool     []string
	blocks      map[int64]*BlockData
}

type BlockData struct {
	Hash     string
	Tx       []TxData
	Previous string
}

type TxData struct {
	TxID     string
	Vout     []VoutData
	Vin      []VinData
	LockTime uint32
}

type VoutData struct {
	N            uint32
	Value        float64
	ScriptPubKey ScriptPubKey
}

type ScriptPubKey struct {
	Hex       string
	Addresses []string
}

type VinData struct {
	TxID string
	Vout uint32
}

func NewRPCServer() *RPCServer {
	return &RPCServer{
		blockHeight: 100,
		mempool:     []string{},
		blocks:      make(map[int64]*BlockData),
	}
}

func (s *RPCServer) handleRPC(w http.ResponseWriter, r *http.Request) {
	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, -32700, "Parse error")
		return
	}

	if *verbose {
		log.Printf("RPC: %s %v", req.Method, req.Params)
	}

	var result interface{}
	var errResp *RPCError

	switch req.Method {
	case "getblockcount":
		result = s.blockHeight
	case "getblockhash":
		height := req.Params[0].(float64)
		result = s.getBlockHash(int64(height))
	case "getblock":
		result = s.getBlock(req.Params[0].(string))
	case "getrawmempool":
		result = s.mempool
	case "getnetworkinfo":
		result = s.getNetworkInfo()
	case "getrawtransaction":
		result = s.getRawTransaction(req.Params[0].(string))
	case "sendrawtransaction":
		result = s.sendRawTransaction(req.Params[0].(string))
	case "createrawtransaction":
		result = s.createRawTransaction(req.Params)
	case "decoderawtransaction":
		result = s.decodeRawTransaction(req.Params[0].(string))
	case "signrawtransactionwithwallet":
		result = s.signRawTransaction(req.Params[0].(string))
	case "getnewaddress":
		result = s.getNewAddress()
	case "getrawchangeaddress":
		result = s.getRawChangeAddress()
	case "listunspent":
		result = s.listUnspent()
	case "estimatefee":
		result = 0.0001
	case "estimatesmartfee":
		result = map[string]interface{}{
			"feerate": 0.0001,
			"blocks":  6,
		}
	default:
		errResp = &RPCError{Code: -32601, Message: "Method not found"}
	}

	if errResp != nil {
		s.sendResponse(w, req.ID, nil, errResp)
	} else {
		s.sendResponse(w, req.ID, result, nil)
	}
}

func (s *RPCServer) getBlockHash(height int64) string {
	if height > s.blockHeight {
		return ""
	}
	hash := fmt.Sprintf("000000000000000000000000000000000000000000000000000000000000%04x", height)
	return hash
}

func (s *RPCServer) getBlock(hash string) interface{} {
	return map[string]interface{}{
		"hash":          hash,
		"height":        s.blockHeight,
		"tx":            []string{},
		"previoushash":  s.getBlockHash(s.blockHeight - 1),
		"nextblockhash": "",
	}
}

func (s *RPCServer) getNetworkInfo() map[string]interface{} {
	return map[string]interface{}{
		"version":         240000,
		"protocolversion": 70015,
		"localservices":   "0000000000000400",
		"relayfee":        0.00001,
	}
}

func (s *RPCServer) getRawTransaction(txid string) map[string]interface{} {
	return map[string]interface{}{
		"txid":     txid,
		"version":  2,
		"locktime": 0,
		"vin":      []VinData{},
		"vout": []VoutData{
			{
				N:     0,
				Value: 50.0,
				ScriptPubKey: ScriptPubKey{
					Hex:       "76a914" + strings.Repeat("ab", 20) + "88ac",
					Addresses: []string{"bcrt1qtest"},
				},
			},
		},
	}
}

func (s *RPCServer) sendRawTransaction(hexTx string) string {
	txHash := fmt.Sprintf("tx%064d", rand.Int63())
	s.mempool = append(s.mempool, txHash)
	return txHash
}

func (s *RPCServer) createRawTransaction(params []interface{}) string {
	return "02000000000100000000000000000000000000000000000000000000000000000000000000000000"
}

func (s *RPCServer) decodeRawTransaction(hexTx string) map[string]interface{} {
	return map[string]interface{}{
		"txid":     "decodedtx123",
		"version":  2,
		"locktime": 0,
		"vin":      []VinData{},
		"vout":     []VoutData{},
	}
}

func (s *RPCServer) signRawTransaction(hexTx string) map[string]interface{} {
	return map[string]interface{}{
		"hex":      hexTx,
		"complete": true,
	}
}

func (s *RPCServer) getNewAddress() string {
	return "bcrt1q" + strings.Repeat("test", 9)[:38]
}

func (s *RPCServer) getRawChangeAddress() string {
	return "bcrt1q" + strings.Repeat("change", 9)[:38]
}

func (s *RPCServer) listUnspent() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"txid":          "utxo0001",
			"vout":          0,
			"address":       "bcrt1qtest",
			"amount":        10.0,
			"confirmations": 6,
			"spendable":     true,
		},
		{
			"txid":          "utxo0002",
			"vout":          0,
			"address":       "bcrt1qtest2",
			"amount":        5.0,
			"confirmations": 10,
			"spendable":     true,
		},
	}
}

func (s *RPCServer) sendResponse(w http.ResponseWriter, id interface{}, result interface{}, err *RPCError) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
		Error:   err,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *RPCServer) sendError(w http.ResponseWriter, code int, message string) {
	w.WriteHeader(code)
	fmt.Fprintf(w, `{"error": {"code": %d, "message": "%s"}}`, code, message)
}

type JSONRPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      interface{}   `json:"id"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func main() {
	flag.Parse()

	server := NewRPCServer()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			server.handleRPC(w, r)
		} else {
			fmt.Fprintf(w, "Bitcoin-compatible RPC server\n")
			fmt.Fprintf(w, "Listening on %s\n", *listenAddr)
			fmt.Fprintf(w, "Network: %s\n", *network)
		}
	})

	log.Printf("Starting mock RPC server on %s (%s)", *listenAddr, *network)
	log.Fatal(http.ListenAndServe(*listenAddr, nil))
}
