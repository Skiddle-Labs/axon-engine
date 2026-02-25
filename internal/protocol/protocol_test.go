package protocol

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/personal-github/axon-engine/internal/search"
)

// TestProtocol_Handshake tests the basic UCI initialization sequence.
func TestProtocol_Handshake(t *testing.T) {
	input := "uci\nisready\nquit\n"
	var output bytes.Buffer
	p := NewProtocol(strings.NewReader(input), &output)
	p.Start()

	got := output.String()

	expectedIDs := []string{
		"id name Axon Engine",
		"id author Axon Team",
		"option name Hash type spin default 64 min 1 max 65536",
		"option name Threads type spin default 1 min 1 max 128",
		"option name MultiPV type spin default 1 min 1 max 128",
		"option name Ponder type check default false",
		"option name Move Overhead type spin default 10 min 0 max 5000",
		"option name Slow Mover type spin default 100 min 10 max 1000",
		"option name Clear Hash type button",
		"option name UCI_AnalyseMode type check default false",
		"option name UCI_Opponent type string",
		"option name Book File type string default <none>",
		"option name Book Best Move type check default false",
		"option name Book Depth type spin default 255 min 0 max 255",
		"uciok",
		"readyok",
	}

	for _, expected := range expectedIDs {
		if !strings.Contains(got, expected) {
			t.Errorf("Protocol output missing expected string: %q\nFull output:\n%s", expected, got)
		}
	}
}

// TestProtocol_DisplayCommand verifies that the 'd' command renders the engine.
func TestProtocol_DisplayCommand(t *testing.T) {
	input := "position startpos\nd\nquit\n"
	var output bytes.Buffer
	p := NewProtocol(strings.NewReader(input), &output)
	p.Start()

	got := output.String()

	// Check for board frame
	if !strings.Contains(got, "+---+---+---+---+---+---+---+---+") {
		t.Error("Display command 'd' did not output a board frame")
	}

	// Check for initial pieces in rank 1 and 8
	if !strings.Contains(got, "R | N | B | Q | K | B | N | R") {
		t.Error("Initial White pieces not found in board display")
	}
	if !strings.Contains(got, "r | n | b | q | k | b | n | r") {
		t.Error("Initial Black pieces not found in board display")
	}
}

// TestProtocol_PositionParsing verifies that FEN loading and move application works.
func TestProtocol_PositionParsing(t *testing.T) {
	// FEN for 1. e4
	fen := "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1"
	input := "position fen " + fen + "\nd\nquit\n"
	var output bytes.Buffer
	p := NewProtocol(strings.NewReader(input), &output)
	p.Start()

	got := output.String()

	// The pawn should be on rank 4 (labeled "4 |") at file e (index 4 in the 8x8 grid)
	// In the engine.String() implementation: "4 | . | . | . | . | P | . | . | . |"
	if !strings.Contains(got, "4 | . | . | . | . | P | . | . | . |") {
		t.Errorf("Pawn not found at e4 in FEN position. Got:\n%s", got)
	}
}

// TestProtocol_SetOption tests the UCI option configuration.
func TestProtocol_SetOption(t *testing.T) {
	// We test the 'setoption' command for Hash.
	// This ensures the parser correctly identifies the command.
	input := "setoption name Hash value 128\nquit\n"
	var output bytes.Buffer
	p := NewProtocol(strings.NewReader(input), &output)

	// We run it mainly to ensure no panics or errors occur during command parsing.
	p.Start()
}

func TestProtocol_SetOptionValues(t *testing.T) {
	p := NewProtocol(strings.NewReader(""), &bytes.Buffer{})

	p.handleSetOption([]string{"setoption", "name", "Threads", "value", "4"})
	if p.threads != 4 {
		t.Fatalf("expected Threads to be 4, got %d", p.threads)
	}

	p.handleSetOption([]string{"setoption", "name", "MultiPV", "value", "3"})
	if p.multiPV != 3 {
		t.Fatalf("expected MultiPV to be 3, got %d", p.multiPV)
	}

	oldTT := search.GlobalTT
	p.handleSetOption([]string{"setoption", "name", "Hash", "value", "128"})
	if search.GlobalTT == oldTT {
		t.Fatal("expected Hash setoption to replace the transposition table")
	}

	p.handleSetOption([]string{"setoption", "name", "Move", "Overhead", "value", "50"})
	if p.moveOverhead != 50 {
		t.Fatalf("expected Move Overhead to be 50, got %d", p.moveOverhead)
	}

	p.handleSetOption([]string{"setoption", "name", "Book", "Best", "Move", "value", "true"})
	if !p.bookBestMove {
		t.Fatal("expected Book Best Move to be true")
	}

	p.handleSetOption([]string{"setoption", "name", "Book", "Depth", "value", "10"})
	if p.bookDepth != 10 {
		t.Fatalf("expected Book Depth to be 10, got %d", p.bookDepth)
	}
}

func TestProtocol_SetOptionInvalidTokens(t *testing.T) {
	p := NewProtocol(strings.NewReader(""), &bytes.Buffer{})
	p.handleSetOption([]string{"setoption", "name", "Threads"})
	if p.threads != 1 {
		t.Fatalf("expected Threads to remain default 1, got %d", p.threads)
	}
}

// TestProtocol_UCINewGame tests the ucinewgame command.
func TestProtocol_UCINewGame(t *testing.T) {
	input := "ucinewgame\nquit\n"
	var output bytes.Buffer
	p := NewProtocol(strings.NewReader(input), &output)
	p.Start()
}

// TestProtocol_ParseMove verifies that algebraic move strings are correctly mapped to internal moves.
func TestProtocol_ParseMove(t *testing.T) {
	input := "position startpos moves e2e4 e7e5\nd\nquit\n"
	var output bytes.Buffer
	p := NewProtocol(strings.NewReader(input), &output)
	p.Start()

	got := output.String()

	// Verify e4 pawn
	if !strings.Contains(got, "4 | . | . | . | . | P | . | . | . |") {
		t.Error("White pawn not found on e4 after 'moves e2e4'")
	}
	// Verify e5 pawn
	if !strings.Contains(got, "5 | . | . | . | . | p | . | . | . |") {
		t.Error("Black pawn not found on e5 after 'moves e7e5'")
	}
}

func TestProtocol_PonderLogic(t *testing.T) {
	input := "uci\nisready\nposition startpos\ngo ponder wtime 100000 btime 100000\nponderhit\nstop\nquit\n"
	var output bytes.Buffer
	p := NewProtocol(strings.NewReader(input), &output)

	done := make(chan bool)
	go func() {
		p.Start()
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Protocol loop timed out during ponder test")
	}
}
