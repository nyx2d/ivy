package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"log"

	"github.com/nyx2d/ivy/network"
)

func main() {
	pubKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatal(err)
	}
	peerID := base64.StdEncoding.EncodeToString(pubKey)

	port, err := network.Serve(peerID, privateKey)
	if err != nil {
		log.Fatal(err)
	}

	err = network.Broadcast(peerID, port)
	if err != nil {
		log.Fatal(err)
	}

	network.FindPeers(peerID, privateKey)
}
