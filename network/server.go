package network

import (
	"log"
	"net"
)

func (m *Manager) Serve() error {
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		return err
	}
	m.serverAddr = lis.Addr().(*net.TCPAddr)

	log.Printf("👂 server listening at %v", lis.Addr().String())
	go func() {
		for {
			conn, err := lis.Accept()
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("🔁 new connection from %v", conn.RemoteAddr().String())
			go func() {
				if m.connActive(conn.RemoteAddr()) { // already established, reject peer
					log.Println("already has peer", conn.RemoteAddr().String())
					conn.Close()
					return
				}

				err := m.HandleClient(conn)
				if err != nil {
					log.Fatal(err)
				}
			}()
		}
	}()

	return nil
}
