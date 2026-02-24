package main

import (
	"os"

	"github.com/personal-github/axon-engine/internal/protocol"
)

func main() {
	// Initialize the UCI protocol handler
	p := protocol.NewProtocol(os.Stdin, os.Stdout)

	// Start the main engine loop
	p.Start()
}
