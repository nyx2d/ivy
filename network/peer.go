package network

import (
	"crypto/ed25519"
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

	if !p.isClient {
		err := m.addPeer(p)
		if err != nil {
			log.Errorf("⛔ add peer err: %v\n", err)
			return
		}
		defer m.removePeer(p)

		err = p.sendHandshake(m.peerID, m.privateKey) // we are the client, let the server know who we are
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
				log.Fatal(err)
			}

			log.Tracef("🔗 message %v from %s (%s)\n", msg, p.Conn.RemoteAddr().String(), p.TypeIndicator())

			if msg.Handshake != nil {
				// TODO: validate sig, set pubkey
				p.ID = msg.Handshake.PeerID
				log.Tracef("📨 got handshake from %s, %s (%s)\n", p.ID, p.Conn.RemoteAddr().String(), p.TypeIndicator())

				if m.peerActive(p.ID) {
					// already have peer, drop them
					log.Errorf("⛔ already have peer %s, dropping %s\n", p.ID, p.Conn.RemoteAddr().String())
					return
				}

				err := m.addPeer(p)
				if err != nil {
					log.Errorf("⛔ add peer err: %v\n", err)
					return
				}
				defer m.removePeer(p)
			}
		}
	}
}

func (p *Peer) sendHandshake(peerID string, privateKey ed25519.PrivateKey) error {
	r := rpc.RPCMessage{Handshake: &rpc.Handshake{
		PeerID: peerID,
	}}
	b, err := r.Encode()
	if err != nil {
		return err
	}
	err = p.sendMessage(b)
	log.Tracef("✉️  sent handshake to %s %s (%s)\n", p.ID, p.Conn.RemoteAddr().String(), p.TypeIndicator())
	return err
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

func (p *Peer) TypeIndicator() string {
	if p.isClient {
		return "client"
	}
	return "server"
}
