package network

import (
	"log"

	"github.com/hashicorp/mdns"
)

const serviceName = "ivy"

func (m *Manager) Broadcast() error {
	service, err := mdns.NewMDNSService(m.peerID, serviceName, "", "", m.serverAddr.Port, nil, []string{serviceName, m.peerID})
	if err != nil {
		return err
	}

	// kicks off gorotuine in constructor
	server, err := mdns.NewServer(&mdns.Config{Zone: service})
	if err != nil {
		return err
	}
	_ = server // TODO: figure out server shutdown

	log.Printf("broadcasting as peer %s\n", m.peerID)
	return nil
}
