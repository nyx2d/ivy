package network

import (
	"net"
	"sync"
)

var connManager = newConnectionManager()

type connectionManager struct {
	sync.Mutex

	connections map[string]struct{} // net.Addr.String() set
}

func newConnectionManager() *connectionManager {
	return &connectionManager{connections: make(map[string]struct{})}
}

func (c *connectionManager) Active(a net.Addr) bool {
	c.Lock()
	defer c.Unlock()
	_, ok := c.connections[a.String()]
	return ok
}

func (c *connectionManager) Add(conn net.Conn) {
	c.Lock()
	defer c.Unlock()
	c.connections[conn.RemoteAddr().String()] = struct{}{}
}

func (c *connectionManager) Remove(conn net.Conn) {
	c.Lock()
	defer c.Unlock()
	delete(c.connections, conn.RemoteAddr().String())
}
