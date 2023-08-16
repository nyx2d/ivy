package network

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"syscall"
	"time"

	"github.com/nyx2d/ivy/rpc"
)

type Peer struct {
	Conn net.Conn

	isClient bool // if this peer is a client of ours
}

// NewPeer takes in an established peer connection with us as the server
func NewClientPeer(c net.Conn) (*Peer, error) {
	connManager.Add(c)
	log.Printf("got connection from client at %s\n", c.RemoteAddr().String())

	return &Peer{
		Conn: c,

		isClient: true,
	}, nil
}

// / NewServerPeer establishes a connection to a new peer with us as the client
func NewServerPeer(a net.Addr) (*Peer, error) {
	conn, err := net.Dial(a.Network(), a.String())
	if err != nil {
		return nil, err
	}

	connManager.Add(conn)
	log.Printf("established connection to server at %s\n", conn.RemoteAddr().String())

	return &Peer{Conn: conn}, nil
}

func (p *Peer) Handle() {
	defer p.Conn.Close()
	defer connManager.Remove(p.Conn)

	readC, readErrC := p.read()
	heartbeatErrC := p.heartbeat()
	for {
		select {
		case err := <-readErrC:
			if err == io.EOF {
				log.Printf("peer closed connection %s\n", p.Conn.RemoteAddr().String())
				return
			}
			log.Fatal(err)

		case err := <-heartbeatErrC:
			if errors.Is(err, syscall.EPIPE) {
				log.Printf("peer closed connection %s\n", p.Conn.RemoteAddr().String())
				return
			}
			log.Fatal(err)

		case in := <-readC:
			m, err := rpc.Decode(in)
			if err != nil {
				log.Fatal(err)
			}

			if m.RPCRequest != nil && m.RPCRequest.HeartbeatRPCRequest != nil {
				log.Println(m.RPCRequest.HeartbeatRPCRequest.Message)
			}
		}
	}
}

func (p *Peer) heartbeat() <-chan error {
	errC := make(chan error)
	go func() {
		for {
			r := rpc.RPCMessage{RPCRequest: &rpc.RPCRequest{HeartbeatRPCRequest: &rpc.HeartbeatRPCRequest{
				Message: fmt.Sprintf("heartbeat from %s", p.Conn.LocalAddr().String()),
			}}}
			b, err := r.Encode()
			if err != nil {
				errC <- err
				return
			}

			err = p.sendMessage(b)
			if err != nil {
				errC <- err
				return
			}

			time.Sleep(15 * time.Second)
		}
	}()
	return errC
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
