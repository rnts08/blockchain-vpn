package protocol

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

	// Parse pushed data from the script.
	pushes, err := txscript.PushedData(pkScript)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pkscript: %w", err)
	}

	if len(pushes) > 0 {
		return pushes[0], nil
	}

	// If it's just OP_RETURN with no data push, return an empty payload.
	return []byte{}, nil
}
