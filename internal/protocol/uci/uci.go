package uci

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Skiddle-Labs/axon-engine/internal/engine"
	"github.com/Skiddle-Labs/axon-engine/internal/search"
)

// UCI represents the state of a UCI protocol session.
type UCI struct {
	reader *bufio.Scanner
	writer io.Writer

	board  *engine.Board
	engine *search.Engine

	// Configurable options
	threads      int
	multiPV      int
	moveOverhead int
	slowMover    int

	// Book settings
	book         *engine.PolyglotBook
	bookDepth    int
	bookBestMove bool

	// Search state
	isPondering      bool
	pendingTimeLimit time.Duration

	// Persistent heuristic tables
	historyTable [2][7][64]int
	counterMoves [64][64]engine.Move
}

// NewUCI creates a new UCI protocol handler.
func NewUCI(r io.Reader, w io.Writer) *UCI {
	return &UCI{
		reader:       bufio.NewScanner(r),
		writer:       w,
		board:        engine.NewBoard(),
		threads:      1,
		multiPV:      1,
		moveOverhead: 10,
		slowMover:    100,
		bookDepth:    255,
	}
}

// Start runs the main UCI loop.
func (u *UCI) Start() {
	for u.reader.Scan() {
		line := strings.TrimSpace(u.reader.Text())
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		command := fields[0]

		switch command {
		case "uci":
			u.handleUCI()
		case "isready":
			u.handleIsReady()
		case "setoption":
			u.handleSetOption(fields)
		case "ucinewgame":
			u.handleUCINewGame()
		case "position":
			u.handlePosition(fields)
		case "go":
			u.handleGo(fields)
		case "stop":
			u.handleStop()
		case "ponderhit":
			u.handlePonderHit()
		case "quit":
			return
		case "display", "d":
			fmt.Fprintln(u.writer, u.board.String())
		case "eval":
			u.handleEval()
		case "bench":
			u.handleBench(fields)
		case "count":
			u.handleCount(fields)
		}
	}
}

func (u *UCI) send(s string) {
	fmt.Fprintln(u.writer, s)
}
