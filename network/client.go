package network

import (
	"log"
	"net"
	"time"

	"github.com/hashicorp/mdns"
)

const scanPeriod = 2 * time.Second

func FindPeers(peerID string) {
	ticker := time.NewTicker(scanPeriod)
	for ; true; <-ticker.C {
		entries := make(chan *mdns.ServiceEntry, 256)
		err := mdns.Query(&mdns.QueryParam{
			Service:     serviceName,
			Domain:      "local",
			Timeout:     scanPeriod,
			Entries:     entries,
			DisableIPv6: true,
		})
		if err != nil {
			log.Fatal(err)
		}

	loop:
		for {
			select {
			case entry := <-entries:
				if len(entry.InfoFields) < 2 || entry.InfoFields[0] != serviceName {
					continue // not a node, extra guard against bad mDNS DNS-SD behavior (looking at you roku)
				}
				if entry.InfoFields[1] == peerID {
					continue // skip self
				}

				addr := &net.TCPAddr{IP: entry.AddrV4, Port: entry.Port}
				if entry.Info != peerID && !connManager.Active(addr) {
					go func() {
						peer, err := NewServerPeer(addr)
						if err != nil {
							log.Printf("error connecting to peer: %s\n", err)
							return
						}
						peer.Handle()
					}()
				}

			case <-time.After(scanPeriod):
				break loop
			}
		}
	}
}
