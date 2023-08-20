package network

import (
	"crypto/ed25519"
	"log"
	"net"
)

func Serve(peerID string, privateKey ed25519.PrivateKey) (int, error) {
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}

	log.Printf("server listening at %v", lis.Addr())
	go func() {
		for {
			conn, err := lis.Accept()
			if err != nil {
				log.Fatal(err)
			}
			go func() {
				if connManager.Active(conn.RemoteAddr()) { // already established, reject peer
					log.Println("already has peer", conn.RemoteAddr().String())
					conn.Close()
					return
				}

				peer, err := NewClientPeer(conn)
				if err != nil {
					log.Fatal(err)
				}

				peer.Handle(peerID, privateKey)
			}()
		}
	}()

	return lis.Addr().(*net.TCPAddr).Port, nil
}
