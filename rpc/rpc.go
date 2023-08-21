package rpc

import (
	"crypto/ed25519"
	"encoding/base64"

	"github.com/fxamacker/cbor/v2"
	log "github.com/sirupsen/logrus"
)

type RPCMessage struct {
	RequestID int64
	Error     bool
	End       bool

	*Handshake
	*Encrypted
	*Heartbeat
}

type Heartbeat struct {
	Message string
}

type Encrypted struct {
	Payload []byte
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

func NewHandshake(peerID string, transportPublicKey []byte, signingKey ed25519.PrivateKey) ([]byte, error) {
	sig := ed25519.Sign(signingKey, transportPublicKey)
	m := RPCMessage{Handshake: &Handshake{
		PeerID:    peerID,
		PublicKey: transportPublicKey,
		Signature: sig,
	}}
	return m.Encode()
}

func VerifyHandshake(m RPCMessage) bool {
	if m.Handshake == nil {
		log.Error("⛔ handshake is nil")
		return false
	}
	signingKey, err := base64.StdEncoding.DecodeString(m.Handshake.PeerID)
	if err != nil {
		log.Error(err)
		return false
	}
	return ed25519.Verify(signingKey, m.Handshake.PublicKey, m.Handshake.Signature)
}
