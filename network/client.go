package network

import (
	"log"
	"net"
	"time"

	"github.com/hashicorp/mdns"
	"github.com/samber/lo"
	_ "github.com/samber/lo"
)

const scanPeriod = 10
const scanTimeout = 2 * time.Second
const scanBufferSize = 256

func FindPeers(peerID string) {
	ticker := time.NewTicker(scanPeriod)
	for ; true; <-ticker.C {
		entriesChan := make(chan *mdns.ServiceEntry, scanBufferSize)
		err := mdns.Query(&mdns.QueryParam{
			Service:     serviceName,
			Domain:      "local",
			Timeout:     scanTimeout,
			Entries:     entriesChan,
			DisableIPv6: true,
		})
		if err != nil {
			log.Fatal(err)
		}

		entries, _, _, ok := lo.BufferWithTimeout(entriesChan, scanBufferSize, scanTimeout)
		if !ok {
			log.Println("issue with buffer timeout")
			continue
		}

		for _, entry := range entries {
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
		}
	}
}
