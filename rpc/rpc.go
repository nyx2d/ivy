package rpc

import (
	"github.com/fxamacker/cbor/v2"
)

type Message struct {
	*Handshake
	*Encrypted
}

type Encrypted struct {
	Payload []byte
}

type Handshake struct {
	PeerID    string
	PublicKey []byte
	Signature []byte
}

func (m Message) Encode() ([]byte, error) {
	return cbor.Marshal(m)
}

func Decode(raw []byte) (Message, error) {
	var m Message
	err := cbor.Unmarshal(raw, &m)
	return m, err
}
