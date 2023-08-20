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

// NewPeer takes in an established peer connection with us as the server
func NewClientPeer(c net.Conn) (*Peer, error) {
	connManager.Add(c) // we only add clients to the peerManager once they've send us their pubkey
	log.Printf("got connection from client at %s\n", c.RemoteAddr().String())

	return &Peer{
		Conn: c,

		isClient: true,
	}, nil
}

// / NewServerPeer establishes a connection to a new peer with us as the client
func NewServerPeer(peerID string, a net.Addr) (*Peer, error) {
	conn, err := net.Dial(a.Network(), a.String())
	if err != nil {
		return nil, err
	}

	p := &Peer{ID: peerID, Conn: conn}
	connManager.Add(conn)
	peerManager.Add(p)
	log.Printf("established connection to server at %s\n", conn.RemoteAddr().String())

	return p, nil
}

func (p *Peer) Handle(peerID string, privateKey ed25519.PrivateKey) {
	defer p.Conn.Close()
	defer connManager.Remove(p.Conn)
	defer peerManager.Remove(p)

	if !p.isClient {
		p.sendHandshake(peerID, privateKey) // we are the client, let the server know who we are
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
			m, err := rpc.Decode(in)
			if err != nil {
				log.Fatal(err)
			}

			if m.Handshake != nil {
				// TODO: validate sig, set pubkey
				p.ID = m.Handshake.PeerID
				peerManager.Add(p)
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
