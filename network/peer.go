package network

import (
	"crypto/ed25519"
	"encoding/binary"
	"io"
	"log"
	"net"

	"github.com/nyx2d/ivy/rpc"
)

type Peer struct {
	ID   string
	Conn net.Conn

	isClient bool // if this peer is a client of ours
}

// HandleClient takes in an established connection with us as the server
func (m *Manager) HandleClient(c net.Conn) error {
	m.addConn(c) // we only add clients to the peers set once they've sent us their pubkey
	log.Printf("got connection from client at %s\n", c.RemoteAddr().String())

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

	p := &Peer{ID: peerID, Conn: conn}
	m.addConn(conn)
	m.addPeer(p)
	log.Printf("established connection to server at %s\n", conn.RemoteAddr().String())

	go m.HandlePeer(p)
	return nil
}

func (m *Manager) HandlePeer(p *Peer) {
	defer p.Conn.Close()
	defer m.removeConn(p.Conn)
	defer m.removePeer(p)

	if !p.isClient {
		p.sendHandshake(m.peerID, m.privateKey) // we are the client, let the server know who we are
	}

	readC, readErrC := p.read()
	for {
		select {
		case err := <-readErrC:
			if err == io.EOF {
				log.Printf("peer closed connection %s\n", p.Conn.RemoteAddr().String())
				return
			}
			log.Fatal(err)

		case in := <-readC:
			msg, err := rpc.Decode(in)
			if err != nil {
				log.Fatal(err)
			}

			if msg.Handshake != nil {
				// TODO: validate sig, set pubkey
				p.ID = msg.Handshake.PeerID
				m.addPeer(p)
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
	return p.sendMessage(b)
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
