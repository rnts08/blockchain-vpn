package protocol

import "testing"

func FuzzProtocolDecoders(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0x6a}) // OP_RETURN only
	f.Add([]byte{0, 1, 2, 3, 4, 5})

	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = DecodePayload(data)
		_, _ = DecodePayloadV2(data)
		_, _ = DecodePaymentPayload(data)
		_, _ = DecodePriceUpdatePayload(data)
		_, _ = DecodeHeartbeatPayload(data)
		_, _ = ExtractScriptPayload(data)
	})
}
