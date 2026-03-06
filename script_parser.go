package main

import (
	"fmt"

	"github.com/btcsuite/btcd/txscript"
)

// ExtractScriptPayload safely parses a PkScript and extracts the data payload
// from a standard OP_RETURN output. It returns the data payload or an error
// if the script is not a valid, standard OP_RETURN script.
//
// The function relies on txscript.GetScriptClass to ensure the script adheres to
// standardness rules for null data outputs (starts with OP_RETURN, one data push,
// respects max size).
func ExtractScriptPayload(pkScript []byte) ([]byte, error) {
	// First, ensure the script is a standard null data script.
	if txscript.GetScriptClass(pkScript) != txscript.NullDataTy {
		return nil, fmt.Errorf("script is not a standard null data (OP_RETURN) script")
	}

	// Parse the script to access the pushed data.
	parsedOpcodes, err := txscript.ParsePkScript(pkScript)
	if err != nil {
		// This should not happen if GetScriptClass passed, but check for safety.
		return nil, fmt.Errorf("failed to parse pkscript: %w", err)
	}

	// If there's more than just the OP_RETURN, the second opcode contains our data.
	if len(parsedOpcodes) > 1 {
		return parsedOpcodes[1].GetData(), nil
	}

	// If it's just OP_RETURN with no data push, return an empty payload.
	return []byte{}, nil
}
