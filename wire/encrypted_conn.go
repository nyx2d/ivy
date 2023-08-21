package wire

import (
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net"

	"github.com/nyx2d/ivy/rpc"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/chacha20poly1305"
)

type EncryptedConn struct {
	*Conn

	// our temp keys for transport encryption
	transportPublicKey  *ecdh.PublicKey
	transportPrivateKey *ecdh.PrivateKey

	peerTransportPublicKey *ecdh.PublicKey    // their temp public key for transport encryption
	peerPublicSigningKey   *ed25519.PublicKey // their long term public signing key from the handshake

	// our shared secret
	transportSharedKey []byte
	transportCipher    cipher.AEAD
}

func NewEncryptedConn(conn net.Conn) (*EncryptedConn, error) {
	transportPrivateKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	return &EncryptedConn{
		Conn:                NewConn(conn),
		transportPrivateKey: transportPrivateKey,
		transportPublicKey:  transportPrivateKey.PublicKey(),
	}, nil
}

func (c *EncryptedConn) HandshakeAsClient(signingKey ed25519.PrivateKey) error {
	if err := c.Conn.SendMessage(c.buildHandshakeMessage(signingKey)); err != nil {
		return err
	}

	serverHandshake, err := c.Conn.ReadMessage()
	if err != nil {
		return err
	}

	if !c.verifyHandshake(serverHandshake) {
		return errors.New("handshake verification failed")
	}

	err = c.deriveSharedKey()
	if err != nil {
		return err
	}

	// send test message
	return c.SendMessage(&rpc.RPCMessage{Heartbeat: &rpc.Heartbeat{Message: "hello"}})
}

func (c *EncryptedConn) HandshakeAsServer(signingKey ed25519.PrivateKey) error {
	clientHandshake, err := c.Conn.ReadMessage()
	if err != nil {
		return err
	}

	if !c.verifyHandshake(clientHandshake) {
		return errors.New("handshake verification failed")
	}

	err = c.Conn.SendMessage(c.buildHandshakeMessage(signingKey))
	if err != nil {
		return err
	}

	err = c.deriveSharedKey()
	if err != nil {
		return err
	}

	// recv test message
	msg, err := c.ReadMessage()
	if err != nil {
		return err
	}
	log.Info(msg.Heartbeat.Message)

	return nil
}

func (c *EncryptedConn) buildHandshakeMessage(signingKey ed25519.PrivateKey) *rpc.RPCMessage {
	peerID := base64.StdEncoding.EncodeToString(signingKey.Public().(ed25519.PublicKey))
	sig := ed25519.Sign(signingKey, c.transportPublicKey.Bytes())
	return &rpc.RPCMessage{Handshake: &rpc.Handshake{
		PeerID:    peerID, // TODO: replace peerID with pubkey directly
		PublicKey: c.transportPublicKey.Bytes(),
		Signature: sig,
	}}
}

// verifyHandshake mutates the conn to include the peer's public keys
func (c *EncryptedConn) verifyHandshake(m *rpc.RPCMessage) bool {
	if m.Handshake == nil {
		return false
	}

	signingKey, err := base64.StdEncoding.DecodeString(m.Handshake.PeerID) // TODO: replace peerID with pubkey directly
	if err != nil {
		return false
	}
	peerTransportPublicKey, err := ecdh.X25519().NewPublicKey(m.Handshake.PublicKey)
	if err != nil {
		return false
	}

	ok := ed25519.Verify(signingKey, m.Handshake.PublicKey, m.Handshake.Signature)
	if ok {
		c.peerTransportPublicKey = peerTransportPublicKey
		c.peerPublicSigningKey = (*ed25519.PublicKey)(&signingKey)
	}
	return ok
}

func (c *EncryptedConn) deriveSharedKey() error {
	shared, err := c.transportPrivateKey.ECDH(c.peerTransportPublicKey)
	if err != nil {
		return err
	}
	cypher, err := chacha20poly1305.NewX(shared)
	if err != nil {
		return err
	}
	c.transportSharedKey = shared
	c.transportCipher = cypher

	return nil
}

func (c *EncryptedConn) ReadMessage() (*rpc.RPCMessage, error) {
	msg, err := c.Conn.ReadMessage()
	if err != nil {
		return nil, err
	}

	if msg.Encrypted == nil {
		return nil, errors.New("message not encrypted")
	}

	rawMsg, err := c.transportCipher.Open(nil, make([]byte, c.transportCipher.NonceSize()), msg.Encrypted.Payload, nil)
	if err != nil {
		return nil, err
	}

	decodedMsg, err := rpc.Decode(rawMsg)
	if err != nil {
		return nil, err
	}
	return &decodedMsg, nil
}

func (c *EncryptedConn) SendMessage(m *rpc.RPCMessage) error {
	rawMsg, err := m.Encode()
	if err != nil {
		return err
	}
	encryptedMsg := c.transportCipher.Seal(nil, make([]byte, c.transportCipher.NonceSize()), rawMsg, nil)

	return c.Conn.SendMessage(&rpc.RPCMessage{Encrypted: &rpc.Encrypted{
		Payload: encryptedMsg,
	}})
}
