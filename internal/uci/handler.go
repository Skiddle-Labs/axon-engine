package uci

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/personal-github/axon-engine/internal/board"
	"github.com/personal-github/axon-engine/internal/search"
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
		case "go":
			h.handleGo(parts)
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
		for i := moveIndex + 1; i < len(parts); i++ {
			move := h.parseMove(parts[i])
			if move != board.NoMove {
				h.board.MakeMove(move)
			}
		}
	}
}

func (h *Handler) parseMove(moveStr string) board.Move {
	ml := h.board.GenerateMoves()
	for i := 0; i < ml.Count; i++ {
		m := ml.Moves[i]

		// Verify legality by making and unmaking the move
		if !h.board.MakeMove(m) {
			continue
		}
		h.board.UnmakeMove(m)

		s := fmt.Sprintf("%s%s", m.From().String(), m.To().String())

		if len(moveStr) == 5 {
			var p string
			switch m.Flags() & 0xB000 {
			case board.PromoQueen:
				p = "q"
			case board.PromoRook:
				p = "r"
			case board.PromoBishop:
				p = "b"
			case board.PromoKnight:
				p = "n"
			}
			if s+p == moveStr {
				return m
			}
		} else if s == moveStr {
			return m
		}
	}
	return board.NoMove
}

func (h *Handler) handleGo(parts []string) {
	engine := search.NewEngine(h.board)
	depth := 6

	for i := 1; i < len(parts); i++ {
		switch parts[i] {
		case "depth":
			if i+1 < len(parts) {
				if d, err := strconv.Atoi(parts[i+1]); err == nil {
					depth = d
				}
				i++
			}
		case "infinite":
			depth = 64
		}
	}

	bestMove := engine.Search(depth)

	if bestMove == board.NoMove {
		return
	}

	moveStr := fmt.Sprintf("%s%s", bestMove.From().String(), bestMove.To().String())
	if bestMove.Flags()&0x8000 != 0 {
		switch bestMove.Flags() & 0xB000 {
		case board.PromoQueen:
			moveStr += "q"
		case board.PromoRook:
			moveStr += "r"
		case board.PromoBishop:
			moveStr += "b"
		case board.PromoKnight:
			moveStr += "n"
		}
	}

	h.send(fmt.Sprintf("bestmove %s", moveStr))
}

func (h *Handler) handleDisplay() {
	h.send(h.board.String())
}

func (h *Handler) send(msg string) {
	fmt.Fprintln(h.writer, msg)
}
