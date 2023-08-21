package network

import (
	"crypto/ecdh"
	"encoding/base64"
	"errors"

	"github.com/nyx2d/ivy/rpc"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/chacha20poly1305"
)

func (m *Manager) handleMessage(p *Peer, msg rpc.RPCMessage) error {
	if msg.Handshake != nil {
		if p.isClient {
			return m.handleClientHandshake(p, msg)
		}
		return m.handleServerHandshake(p, msg)
	}
	return nil
}

func (m *Manager) sendHandshake(p *Peer) error {
	b, err := rpc.NewHandshake(m.peerID, p.transportPublicKey.Bytes(), m.privateKey)
	if err != nil {
		return err
	}
	return p.sendMessage(b)
}

func (m *Manager) handleServerHandshake(p *Peer, msg rpc.RPCMessage) error {
	log.Tracef("📨 got handshake from %s, %s (%s)\n", p.ID, p.Conn.RemoteAddr().String(), p.TypeIndicator())

	if p.ID != msg.Handshake.PeerID {
		log.Errorf("⛔ peer id mismatch from server %s != %s\n", p.ID, msg.Handshake.PeerID)
		return errors.New("peer id mismatch")
	}

	ok := rpc.VerifyHandshake(msg)
	if !ok {
		log.Errorf("⛔ handshake verification failed for %s\n", p.Conn.RemoteAddr().String())
		return errors.New("handshake verification failed")
	}

	peerTransportPublicKey, err := ecdh.X25519().NewPublicKey(msg.Handshake.PublicKey)
	if err != nil {
		log.Errorf("⛔ invalid transport public key for %s\n", p.Conn.RemoteAddr().String())
		return err
	}
	p.peerTransportPublicKey = peerTransportPublicKey
	log.Tracef("🔐 received transport public key from %s\n", p.Conn.RemoteAddr().String())

	return m.deriveSharedKey(p)
}

func (m *Manager) handleClientHandshake(p *Peer, msg rpc.RPCMessage) error {
	p.ID = msg.Handshake.PeerID
	log.Tracef("📨 got handshake from %s, %s (%s)\n", p.ID, p.Conn.RemoteAddr().String(), p.TypeIndicator())

	if m.peerActive(p.ID) {
		// already have peer, drop them
		log.Errorf("⛔ already have peer %s, dropping %s\n", p.ID, p.Conn.RemoteAddr().String())
		return errors.New("already have peer")
	}

	ok := rpc.VerifyHandshake(msg)
	if !ok {
		log.Errorf("⛔ handshake verification failed for %s\n", p.Conn.RemoteAddr().String())
		return errors.New("handshake verification failed")
	}

	err := m.addPeer(p)
	if err != nil {
		log.Errorf("⛔ add peer err: %v\n", err)
		return err
	}

	peerTransportPublicKey, err := ecdh.X25519().NewPublicKey(msg.Handshake.PublicKey)
	if err != nil {
		log.Errorf("⛔ invalid transport public key for %s\n", p.Conn.RemoteAddr().String())
		return err
	}
	p.peerTransportPublicKey = peerTransportPublicKey
	log.Tracef("🔐 received transport public key from %s\n", p.Conn.RemoteAddr().String())

	err = m.deriveSharedKey(p)
	if err != nil {
		return err
	}

	// now that they're added, send them a handshake back
	return m.sendHandshake(p)
}

func (m *Manager) deriveSharedKey(p *Peer) error {
	shared, err := p.transportPrivateKey.ECDH(p.peerTransportPublicKey)
	if err != nil {
		return err
	}
	cypher, err := chacha20poly1305.NewX(shared)
	if err != nil {
		return err
	}
	p.transportSharedKey = shared
	p.transportCipher = cypher

	log.Tracef("🔑 derived shared key %s with %s\n", base64.StdEncoding.EncodeToString(shared), p.Conn.RemoteAddr().String())

	return nil
}
