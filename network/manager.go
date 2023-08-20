package network

import (
	"crypto/ed25519"
	"encoding/base64"
	"net"
	"sync"
)

type Manager struct {
	peerID     string
	publicKey  ed25519.PublicKey
	privateKey ed25519.PrivateKey

	serverAddr *net.TCPAddr // populated by Serve()

	connMutex sync.Mutex
	conns     map[string]net.Conn // ConnID -> Conn

	peerMutex sync.Mutex
	peers     map[string]*Peer // PeerID -> Peer
}

func NewManager(privateKey ed25519.PrivateKey) *Manager {
	publicKey := privateKey.Public().(ed25519.PublicKey)
	peerID := base64.StdEncoding.EncodeToString(publicKey)

	return &Manager{
		peerID:     peerID,
		publicKey:  publicKey,
		privateKey: privateKey,

		conns: make(map[string]net.Conn),
		peers: make(map[string]*Peer),
	}
}

func (m *Manager) addConn(conn net.Conn) {
	m.connMutex.Lock()
	defer m.connMutex.Unlock()
	m.conns[conn.RemoteAddr().String()] = conn
}

func (m *Manager) connActive(a net.Addr) bool {
	m.connMutex.Lock()
	defer m.connMutex.Unlock()
	_, ok := m.conns[a.String()]
	return ok
}

func (m *Manager) removeConn(conn net.Conn) {
	m.connMutex.Lock()
	defer m.connMutex.Unlock()
	delete(m.conns, conn.RemoteAddr().String())
}

func (m *Manager) addPeer(peer *Peer) {
	m.peerMutex.Lock()
	defer m.peerMutex.Unlock()
	m.peers[peer.ID] = peer
}

func (m *Manager) peerActive(id string) bool {
	m.peerMutex.Lock()
	defer m.peerMutex.Unlock()
	_, ok := m.peers[id]
	return ok
}

func (m *Manager) removePeer(peer *Peer) {
	m.peerMutex.Lock()
	defer m.peerMutex.Unlock()
	delete(m.peers, peer.ID)
}
