package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"log"

	"github.com/nyx2d/ivy/network"
)

func main() {
	// TODO: check for file first, command line args, etc
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)

	network := network.NewManager(privateKey)
	err = network.Serve()
	if err != nil {
		log.Fatal(err)
	}

	err = network.Broadcast()
	if err != nil {
		log.Fatal(err)
	}

	network.FindPeers()
}
