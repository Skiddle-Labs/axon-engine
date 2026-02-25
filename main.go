package main

import (
	"os"
	"strings"

	"github.com/Skiddle-Labs/axon-engine/internal/protocol/uci"
)

func main() {
	// If command line arguments are provided, process them as a single command
	// and then exit. This allows for usage like './axon bench' or './axon eval'.
	if len(os.Args) > 1 {
		command := strings.Join(os.Args[1:], " ")
		input := strings.NewReader(command + "\nquit\n")
		p := uci.NewUCI(input, os.Stdout)
		p.Start()
		return
	}

	// Default mode: start the UCI protocol loop reading from standard input.
	p := uci.NewUCI(os.Stdin, os.Stdout)
	p.Start()
}
