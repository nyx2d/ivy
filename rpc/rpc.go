package rpc

import (
	"crypto/ed25519"

	"github.com/fxamacker/cbor/v2"
)

type RPCMessage struct {
	RequestID int64
	Error     bool
	End       bool

	*Handshake
	*Close
}

type Heartbeat struct {
	Message string
}

type Close struct {
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

func NewHandshake(peerID string, publicKey []byte, handshakePrivateKey ed25519.PrivateKey) ([]byte, error) {
	sig := ed25519.Sign(handshakePrivateKey, publicKey)
	m := RPCMessage{Handshake: &Handshake{
		PeerID:    peerID,
		PublicKey: publicKey,
		Signature: sig,
	}}
	return m.Encode()
}
