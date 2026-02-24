package main

import (
	"os"

	"github.com/personal-github/axon-engine/internal/uci"
)

func main() {
	handler := uci.NewHandler(os.Stdin, os.Stdout)
	handler.Start()
}
