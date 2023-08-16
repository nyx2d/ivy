package rpc

import "github.com/fxamacker/cbor/v2"

type RPCMessage struct {
	RequestID int64
	Error     bool
	End       bool

	*RPCRequest
	*RPCResponse
}

type RPCRequest struct {
	*HeartbeatRPCRequest
}

type HeartbeatRPCRequest struct {
	Message string
}

type RPCResponse struct {
}

func (m RPCMessage) Encode() ([]byte, error) {
	return cbor.Marshal(m)
}

func Decode(raw []byte) (RPCMessage, error) {
	var m RPCMessage
	err := cbor.Unmarshal(raw, &m)
	return m, err
}
