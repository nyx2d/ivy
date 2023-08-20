package network

import "sync"

var peerManager = newPeerConnManager()

type peerConnManager struct {
	sync.Mutex

	peers map[string]*Peer // peerID -> peer
}

func newPeerConnManager() *peerConnManager {
	return &peerConnManager{peers: make(map[string]*Peer)}
}

func (p *peerConnManager) Get(peerID string) (*Peer, bool) {
	p.Lock()
	defer p.Unlock()
	peer, ok := p.peers[peerID]
	return peer, ok
}

func (p *peerConnManager) Active(peerID string) bool {
	p.Lock()
	defer p.Unlock()
	_, ok := p.peers[peerID]
	return ok
}

func (p *peerConnManager) Add(peer *Peer) {
	p.Lock()
	defer p.Unlock()
	p.peers[peer.ID] = peer
}

func (p *peerConnManager) Remove(peer *Peer) {
	p.Lock()
	defer p.Unlock()
	delete(p.peers, peer.ID)
}
