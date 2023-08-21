package network

import (
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/binary"
	"io"
	"net"

	"github.com/nyx2d/ivy/rpc"
	log "github.com/sirupsen/logrus"
)

type Peer struct {
	ID   string
	Conn net.Conn

	isClient bool // if this peer is a client of ours

	// our temp keys for transport encryption
	transportPublicKey  *ecdh.PublicKey
	transportPrivateKey *ecdh.PrivateKey

	// their temp public key for transport encryption
	peerTransportPublicKey *ecdh.PublicKey

	// our shared secret
	transportSharedKey []byte
	transportCipher    cipher.AEAD
}

// HandleClient takes in an established connection with us as the server
func (m *Manager) HandleClient(c net.Conn) error {
	p := &Peer{
		Conn:     c,
		isClient: true,
	}
	go m.HandlePeer(p)
	return nil
}

// ConnectToServer establishes a connection to a new peer with us as the client
func (m *Manager) ConnectToServer(peerID string, a net.Addr) error {
	conn, err := net.Dial(a.Network(), a.String())
	if err != nil {
		return err
	}

	p := &Peer{ID: peerID, Conn: conn, isClient: false}
	go m.HandlePeer(p)
	return nil
}

func (m *Manager) HandlePeer(p *Peer) {
	m.addConn(p.Conn)
	defer m.removeConn(p.Conn)
	defer p.Conn.Close()
	defer m.removePeer(p) // no-op for non-added peer

	// generate temp public key
	transportPrivateKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		log.Errorf("⛔ generate key err: %v\n", err)
		return
	}
	p.transportPrivateKey = transportPrivateKey
	p.transportPublicKey = transportPrivateKey.PublicKey()

	if !p.isClient {
		err := m.addPeer(p)
		if err != nil {
			log.Errorf("⛔ add peer err: %v\n", err)
			return
		}

		err = m.sendHandshake(p) // we are the client, let the server know who we are
		if err != nil {
			log.Errorf("⛔ handshake send err: %v\n", err)
			return
		}
	}

	readC, readErrC := p.read()
	for {
		select {
		case err := <-readErrC:
			if err == io.EOF {
				log.Errorf("⛔ peer closed connection %s (%s)\n", p.Conn.RemoteAddr().String(), p.TypeIndicator())
				return
			}
			log.Errorf("⛔ read err: %v\n", err)
			return

		case in := <-readC:
			msg, err := rpc.Decode(in)
			if err != nil {
				log.Errorf("⛔ msg decode err: %v\n", err)
				return
			}

			log.Tracef("🔗 message %v from %s (%s)\n", msg, p.Conn.RemoteAddr().String(), p.TypeIndicator())

			err = m.handleMessage(p, msg)
			if err != nil {
				log.Errorf("⛔ handle message err: %v\n", err)
				return
			}
		}
	}
}

func (p *Peer) TypeIndicator() string {
	if p.isClient {
		return "client"
	}
	return "server"
}

func (p *Peer) read() (<-chan []byte, <-chan error) {
	c := make(chan []byte)
	errC := make(chan error)
	go func() {
		for {
			msg, err := p.readMessage()
			if err != nil {
				errC <- err
				return
			}
			c <- msg
		}
	}()
	return c, errC
}

func (p *Peer) sendMessage(m []byte) error {
	err := binary.Write(p.Conn, binary.LittleEndian, uint64(len(m)))
	if err != nil {
		return err
	}
	_, err = p.Conn.Write(m)
	return err
}

// blocks until read
func (p *Peer) readMessage() ([]byte, error) {
	sizeBuf := make([]byte, 8)
	_, err := io.ReadFull(p.Conn, sizeBuf)
	if err != nil {
		return nil, err
	}
	size := binary.LittleEndian.Uint64(sizeBuf)

	readBuf := make([]byte, size)
	_, err = io.ReadFull(p.Conn, readBuf)
	if err != nil {
		return nil, err
	}

	return readBuf, nil
}
