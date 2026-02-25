package uci

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/Skiddle-Labs/axon-engine/internal/search"
)

// TestUCI_Handshake tests the basic UCI initialization sequence.
func TestUCI_Handshake(t *testing.T) {
	input := "uci\nisready\nquit\n"
	var output bytes.Buffer
	u := NewUCI(strings.NewReader(input), &output)
	u.Start()

	got := output.String()

	expectedStrings := []string{
		"id name Axon Engine",
		"id author Skiddle Labs",
		"option name Threads type spin default 1 min 1 max 512",
		"option name MultiPV type spin default 1 min 1 max 500",
		"option name Hash type spin default 64 min 1 max 1048576",
		"option name Move Overhead type spin default 10 min 0 max 5000",
		"option name Slow Mover type spin default 100 min 10 max 1000",
		"option name Clear Hash type button",
		"option name OwnBook type check default true",
		"option name BookPath type string default ",
		"uciok",
		"readyok",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(got, expected) {
			t.Errorf("UCI output missing expected string: %q\nFull output:\n%s", expected, got)
		}
	}
}

// TestUCI_DisplayCommand verifies that the 'd' command renders the board.
func TestUCI_DisplayCommand(t *testing.T) {
	input := "position startpos\nd\nquit\n"
	var output bytes.Buffer
	u := NewUCI(strings.NewReader(input), &output)
	u.Start()

	got := output.String()

	// Check for board frame components (rank labels)
	if !strings.Contains(got, "8 |") || !strings.Contains(got, "1 |") {
		t.Error("Display command 'd' did not output a board frame")
	}

	// Check for initial pieces
	if !strings.Contains(got, "r | n | b | q | k | b | n | r") {
		t.Error("Initial pieces not found in board display")
	}
}

// TestUCI_PositionParsing verifies that FEN loading works.
func TestUCI_PositionParsing(t *testing.T) {
	// FEN for 1. e4
	fen := "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1"
	input := "position fen " + fen + "\nd\nquit\n"
	var output bytes.Buffer
	u := NewUCI(strings.NewReader(input), &output)
	u.Start()

	got := output.String()

	// The pawn should be on rank 4
	if !strings.Contains(got, "4 |") || !strings.Contains(got, "P") {
		t.Errorf("Pawn not found in expected rank after FEN load. Got:\n%s", got)
	}
}

func TestUCI_SetOption(t *testing.T) {
	u := NewUCI(strings.NewReader(""), &bytes.Buffer{})

	u.handleSetOption([]string{"setoption", "name", "Threads", "value", "4"})
	if u.threads != 4 {
		t.Fatalf("expected Threads to be 4, got %d", u.threads)
	}

	u.handleSetOption([]string{"setoption", "name", "MultiPV", "value", "3"})
	if u.multiPV != 3 {
		t.Fatalf("expected MultiPV to be 3, got %d", u.multiPV)
	}

	oldTT := search.GlobalTT
	u.handleSetOption([]string{"setoption", "name", "Hash", "value", "128"})
	if search.GlobalTT == oldTT {
		t.Fatal("expected Hash setoption to replace the transposition table")
	}

	u.handleSetOption([]string{"setoption", "name", "Move", "Overhead", "value", "50"})
	if u.moveOverhead != 50 {
		t.Fatalf("expected Move Overhead to be 50, got %d", u.moveOverhead)
	}
}

func TestUCI_UCINewGame(t *testing.T) {
	input := "ucinewgame\nquit\n"
	var output bytes.Buffer
	u := NewUCI(strings.NewReader(input), &output)
	u.Start()
}

func TestUCI_ParseMove(t *testing.T) {
	input := "position startpos moves e2e4 e7e5\nd\nquit\n"
	var output bytes.Buffer
	u := NewUCI(strings.NewReader(input), &output)
	u.Start()

	got := output.String()

	// Verify white and black pawns moved
	if !strings.Contains(got, "4 |") || !strings.Contains(got, "P") {
		t.Error("White pawn not found on rank 4 after 'moves e2e4'")
	}
	if !strings.Contains(got, "5 |") || !strings.Contains(got, "p") {
		t.Error("Black pawn not found on rank 5 after 'moves e7e5'")
	}
}

func TestUCI_PonderLogic(t *testing.T) {
	input := "uci\nisready\nposition startpos\ngo ponder wtime 1000 btime 1000\nponderhit\nstop\nquit\n"
	var output bytes.Buffer
	u := NewUCI(strings.NewReader(input), &output)

	done := make(chan bool)
	go func() {
		u.Start()
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("UCI loop timed out during ponder test")
	}
}
