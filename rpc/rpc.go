package rpc

import "github.com/fxamacker/cbor/v2"

type RPCMessage struct {
	RequestID int64
	Error     bool
	End       bool

	*Handshake
}

type Heartbeat struct {
	Message string
}

type Handshake struct {
	PeerID    string
	PublicKey []byte
	Signature []byte
}

func (m RPCMessage) Encode() ([]byte, error) {
	return cbor.Marshal(m)
}

func Decode(raw []byte) (RPCMessage, error) {
	var m RPCMessage
	err := cbor.Unmarshal(raw, &m)
	return m, err
}
