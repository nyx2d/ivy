package network

import (
	"log"

	"github.com/hashicorp/mdns"
)

const serviceName = "ivy"

func Broadcast(peerID string, servicePort int) error {
	service, err := mdns.NewMDNSService(peerID, serviceName, "", "", servicePort, nil, []string{serviceName, peerID})
	if err != nil {
		return err
	}

	// kicks off gorotuine in constructor
	server, err := mdns.NewServer(&mdns.Config{Zone: service})
	if err != nil {
		return err
	}
	_ = server // TODO: figure out server shutdown

	log.Printf("broadcasting as peer %s\n", peerID)
	return nil
}
