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

	cbor "github.com/fxamacker/cbor/v2"
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
			log.Println(in)
		}
	}
}

func (p *Peer) heartbeat() <-chan error {
	errC := make(chan error)

	go func() {
		for {
			b, err := cbor.Marshal(fmt.Sprintf("heartbeat from %s", p.Conn.LocalAddr().String()))
			if err != nil {
				errC <- err
				return
			}

			err = binary.Write(p.Conn, binary.LittleEndian, uint64(len(b)))
			if err != nil {
				errC <- err
				return
			}

			_, err = p.Conn.Write(b)
			if err != nil {
				errC <- err
				return
			}

			time.Sleep(15 * time.Second)
		}
	}()

	return errC
}

func (p *Peer) read() (<-chan string, <-chan error) {
	c := make(chan string)
	errC := make(chan error)

	go func() {
		for {
			sizeBuf := make([]byte, 8)
			_, err := io.ReadFull(p.Conn, sizeBuf)
			if err != nil {
				errC <- err
				return
			}
			size := binary.LittleEndian.Uint64(sizeBuf)

			readBuf := make([]byte, size)
			_, err = io.ReadFull(p.Conn, readBuf)
			if err != nil {
				errC <- err
				return
			}

			var val string
			err = cbor.Unmarshal(readBuf, &val)
			if err != nil {
				errC <- err
				return
			}

			c <- val
		}
	}()

	return c, errC
}
