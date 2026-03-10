package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/btcsuite/btcd/rpcclient"
)

var verbose bool

func vlog(format string, args ...interface{}) {
	if verbose {
		log.Printf("[VERBOSE] "+format, args...)
	}
}

func main() {
	host := flag.String("host", "localhost:25174", "RPC server host:port")
	user := flag.String("user", "", "RPC username (required)")
	pass := flag.String("pass", "", "RPC password (required)")
	enableTLS := flag.Bool("tls", false, "Enable TLS for RPC connection")
	cmd := flag.String("cmd", "getblockcount", "RPC command to execute")
	verboseFlag := flag.Bool("v", false, "Enable ultra verbose output (logs all actions and raw responses)")
	flag.Parse()

	verbose = *verboseFlag

	if *user == "" || *pass == "" {
		fmt.Fprintln(os.Stderr, "Error: -user and -pass are required")
		flag.Usage()
		os.Exit(1)
	}

	vlog("Connecting to RPC server at %s", *host)
	vlog("TLS enabled: %v", *enableTLS)
	vlog("User: %s", *user)
	vlog("Password: %s", strings.Repeat("*", len(*pass)))

	connCfg := &rpcclient.ConnConfig{
		Host:         *host,
		User:         *user,
		Pass:         *pass,
		HTTPPostMode: true,
		DisableTLS:   !*enableTLS,
	}

	client, err := rpcclient.New(connCfg, nil)
	if err != nil {
		log.Fatalf("Failed to create RPC client: %v", err)
	}
	defer client.Shutdown()
	vlog("RPC client created successfully")

	// Build parameters for generic RPC call (for non-shortcut commands)
	params := make([]json.RawMessage, 0)
	for _, arg := range flag.Args() {
		var parsed json.RawMessage
		if strings.HasPrefix(arg, "{") || strings.HasPrefix(arg, "[") || strings.HasPrefix(arg, "\"") {
			parsed = json.RawMessage(arg)
		} else {
			parsed = json.RawMessage(fmt.Sprintf(`"%s"`, arg))
		}
		params = append(params, parsed)
	}

	// Execute RPC call
	var rawResult json.RawMessage
	var result interface{}

	vlog("Executing command: %s", *cmd)
	if len(params) > 0 {
		vlog("Parameters: %s", formatJSONOrString(params))
	}

	// Simple shortcuts for common commands
	switch *cmd {
	case "getblockcount":
		vlog("Using specialized GetBlockCount method")
		count, err := client.GetBlockCount()
		if err != nil {
			log.Fatalf("RPC failed: %v", err)
		}
		result = map[string]interface{}{"blockcount": count}
		vlog("Received result: %v", result)

	case "getnetworkinfo":
		vlog("Using specialized GetNetworkInfo method")
		info, err := client.GetNetworkInfo()
		if err != nil {
			log.Fatalf("RPC failed: %v", err)
		}
		result = info
		vlog("Received result: %+v", info)

	case "getrawmempool":
		vlog("Using specialized GetRawMempool method")
		txIDs, err := client.GetRawMempool()
		if err != nil {
			log.Fatalf("RPC failed: %v", err)
		}
		result = map[string]interface{}{"txids": txIDs}
		vlog("Received %d transaction(s) in mempool", len(txIDs))

	default:
		vlog("Using generic RawRequest method")
		rawResult, err = client.RawRequest(*cmd, params)
		if err != nil {
			log.Fatalf("RPC failed: %v", err)
		}
		vlog("Raw response received (%d bytes)", len(rawResult))
		vlog("Raw response: %s", string(rawResult))
	}

	// For shortcut commands, marshal result to JSON
	if result != nil {
		var err error
		rawResult, err = json.Marshal(result)
		if err != nil {
			log.Fatalf("Failed to marshal result: %v", err)
		}
	}

	// Pretty print final result
	vlog("Formatting output")
	var out bytes.Buffer
	if err := json.Indent(&out, rawResult, "", "  "); err != nil {
		// Fall back to raw output
		fmt.Println(string(rawResult))
	} else {
		fmt.Println(out.String())
	}
	vlog("Done")
}

// formatJSONOrString tries to format as JSON, falls back to string representation
func formatJSONOrString(v interface{}) string {
	if b, ok := v.([]byte); ok {
		return string(b)
	}
	if data, err := json.Marshal(v); err == nil {
		return string(data)
	}
	return fmt.Sprintf("%v", v)
}
