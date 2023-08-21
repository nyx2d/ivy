package network

import (
	"io"
	"net"

	"github.com/nyx2d/ivy/wire"
	log "github.com/sirupsen/logrus"
)

type Peer struct {
	ID   string
	Conn net.Conn

	EncryptedConn *wire.EncryptedConn

	isClient bool // if this peer is a client of ours
}

func (m *Manager) HandleConn(c net.Conn, asServer bool) error {
	m.addConn(c)
	encryptedConn, err := wire.NewEncryptedConn(c)
	if err != nil {
		m.removeConn(c)
		return err
	}

	if asServer {
		err = encryptedConn.HandshakeAsServer(m.privateKey)
	} else {
		err = encryptedConn.HandshakeAsClient(m.privateKey)
	}
	if err != nil {
		m.removeConn(c)
		return err
	}

	// create peer
	p := &Peer{
		ID:            encryptedConn.PeerID(),
		Conn:          c,
		EncryptedConn: encryptedConn,
		isClient:      asServer,
	}
	m.addPeer(p)
	go m.HandlePeer(p)
	return nil
}

func (m *Manager) HandlePeer(p *Peer) {
	defer m.removeConn(p.Conn)
	defer p.Conn.Close()
	defer m.removePeer(p)

	readC, readErrC := p.EncryptedConn.ReadMessages()
	for {
		select {
		case err := <-readErrC:
			if err == io.EOF {
				log.Errorf("⛔ peer closed connection %s (%s)\n", p.Conn.RemoteAddr().String(), p.TypeIndicator())
				return
			}
			log.Errorf("⛔ read err: %v\n", err)
			return

		case msg := <-readC:
			log.Tracef("🔗 message %v from %s (%s)\n", msg, p.Conn.RemoteAddr().String(), p.TypeIndicator())
		}
	}
}

func (p *Peer) TypeIndicator() string {
	if p.isClient {
		return "client"
	}
	return "server"
}
