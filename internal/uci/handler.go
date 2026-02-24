package uci

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/personal-github/axon-engine/internal/board"
)

const startFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

// Handler manages UCI protocol communication.
type Handler struct {
	reader *bufio.Scanner
	writer io.Writer
	board  *board.Board
}

// NewHandler creates a new UCI handler.
func NewHandler(input io.Reader, output io.Writer) *Handler {
	return &Handler{
		reader: bufio.NewScanner(input),
		writer: output,
		board:  board.NewBoard(),
	}
}

// Start begins the main loop for processing UCI commands.
func (h *Handler) Start() {
	for h.reader.Scan() {
		line := strings.TrimSpace(h.reader.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		command := parts[0]

		switch command {
		case "uci":
			h.handleUCI()
		case "isready":
			h.handleIsReady()
		case "position":
			h.handlePosition(parts)
		case "d":
			h.handleDisplay()
		case "quit":
			return
		default:
			// Ignore unknown commands for now
		}
	}
}

func (h *Handler) handleUCI() {
	h.send("id name Axon Engine")
	h.send("id author Axon Team")
	// Options would be sent here
	h.send("uciok")
}

func (h *Handler) handleIsReady() {
	h.send("readyok")
}

func (h *Handler) handlePosition(parts []string) {
	if len(parts) < 2 {
		return
	}

	var fen string
	moveIndex := -1

	for i, part := range parts {
		if part == "moves" {
			moveIndex = i
			break
		}
	}

	if parts[1] == "startpos" {
		fen = startFEN
	} else if parts[1] == "fen" {
		endIndex := len(parts)
		if moveIndex != -1 {
			endIndex = moveIndex
		}
		if endIndex <= 2 {
			return
		}
		fen = strings.Join(parts[2:endIndex], " ")
	} else {
		return
	}

	h.board.SetFEN(fen)

	if moveIndex != -1 {
		// TODO: Implement move application
	}
}

func (h *Handler) handleDisplay() {
	h.send(h.board.String())
}

func (h *Handler) send(msg string) {
	fmt.Fprintln(h.writer, msg)
}
