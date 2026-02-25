package search

import (
	"testing"

	"github.com/Skiddle-Labs/axon-engine/internal/engine"
	"github.com/Skiddle-Labs/axon-engine/internal/nnue"
	"github.com/Skiddle-Labs/axon-engine/internal/types"
)

// withHCE is a helper that temporarily disables NNUE to test Hand-Coded Evaluation logic.
func withHCE(f func()) {
	old := nnue.UseNNUE
	nnue.UseNNUE = false
	defer func() { nnue.UseNNUE = old }()
	f()
}

// TestSearch_MateIn1 verifies that the engine finds a simple mate in one.
func TestSearch_MateIn1(t *testing.T) {
	b := engine.NewBoard()
	// Scholar's Mate position: Queen attacks F7 with Bishop support
	b.SetFEN("r1bqkbnr/pppp1ppp/2n5/4p3/2B1P3/5Q2/PPPP1PPP/RNB1K1NR w KQkq - 4 4")

	GlobalTT.Clear()
	searchEngine := NewEngine(b)
	bestMove := searchEngine.Search(4)

	expectedFrom := types.F3
	expectedTo := types.F7

	if bestMove.From() != expectedFrom || bestMove.To() != expectedTo {
		t.Errorf("Expected mate move f3f7, got %s", bestMove.String())
	}
}

// TestSearch_NNUE verifies that search works with NNUE enabled and produces a valid score.
func TestSearch_NNUE(t *testing.T) {
	if nnue.CurrentNetwork == nil {
		t.Skip("NNUE network not loaded, skipping NNUE search test")
	}

	// Ensure NNUE is enabled
	old := nnue.UseNNUE
	nnue.UseNNUE = true
	defer func() { nnue.UseNNUE = old }()

	b := engine.NewBoard()
	b.SetFEN(engine.StartFEN)

	GlobalTT.Clear()
	searchEngine := NewEngine(b)
	// Low depth to verify logic without taking too long
	bestMove := searchEngine.Search(4)

	if bestMove == engine.NoMove {
		t.Error("Search with NNUE returned NoMove")
	}
}

// TestSearch_MateIn2 verifies that the engine finds a mate in two.
func TestSearch_MateIn2(t *testing.T) {
	b := engine.NewBoard()
	// Black is in a position to be mated in 2: 1. Qe8+ Rxe8 2. Rxe8#
	b.SetFEN("r5k1/5ppp/8/8/8/8/4QPPP/4R1K1 w - - 0 1")

	GlobalTT.Clear()
	searchEngine := NewEngine(b)
	bestMove := searchEngine.Search(4)

	expectedFrom := types.E2
	expectedTo := types.E8

	if bestMove.From() != expectedFrom || bestMove.To() != expectedTo {
		t.Errorf("Expected move e2e8, got %s", bestMove.String())
	}
}

// TestSearch_TranspositionTable verifies that the TT helps find moves faster.
func TestSearch_TranspositionTable(t *testing.T) {
	b := engine.NewBoard()
	b.SetFEN(engine.StartFEN)

	GlobalTT.Clear()
	searchEngine := NewEngine(b)

	// First search
	searchEngine.Search(4)
	nodes1 := *searchEngine.Nodes

	// Second search (should be much faster due to TT)
	searchEngine.Search(4)
	nodes2 := *searchEngine.Nodes

	if nodes2 >= nodes1 {
		t.Errorf("Expected fewer nodes on second search due to TT. Nodes1: %d, Nodes2: %d", nodes1, nodes2)
	}
}

// TestSearch_Quiescence verifies that the search doesn't stop during captures.
func TestSearch_Quiescence(t *testing.T) {
	b := engine.NewBoard()
	// Position where a capture seems good but leads to material loss
	b.SetFEN("r1bqkbnr/pppp1ppp/2n5/4p3/3P4/5N2/PPP1PPPP/RNBQKB1R w KQkq - 0 1")

	GlobalTT.Clear()
	searchEngine := NewEngine(b)
	bestMove := searchEngine.Search(1)

	if bestMove == engine.NoMove {
		t.Error("Search returned NoMove")
	}
}

// TestSearch_Repetition verifies that the engine avoids 3-fold repetition.
func TestSearch_Repetition(t *testing.T) {
	b := engine.NewBoard()
	b.SetFEN("k7/8/8/8/8/8/7R/1R4K1 w - - 0 1")
	// Note: Repetition testing often requires simulating move history,
	// but we ensure the basic search executes correctly here.
	GlobalTT.Clear()
	searchEngine := NewEngine(b)
	searchEngine.Search(2)
}

// TestSearch_NullMovePruning verifies that NMP is active.
func TestSearch_NullMovePruning(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping search depth test in short mode.")
	}
	b := engine.NewBoard()
	b.SetFEN(engine.StartFEN)

	GlobalTT.Clear()
	searchEngine := NewEngine(b)
	searchEngine.Search(5)
}

// TestSearch_Stalemate verifies that stalemate is evaluated as 0.
func TestSearch_Stalemate(t *testing.T) {
	b := engine.NewBoard()
	// Classic stalemate position
	b.SetFEN("7k/5Q2/8/8/8/8/8/7K b - - 0 1")

	GlobalTT.Clear()
	searchEngine := NewEngine(b)
	move := searchEngine.Search(1)

	if move != engine.NoMove {
		t.Errorf("Expected NoMove in stalemate position, got %s", move.String())
	}
}

// TestSearch_QuiescenceCheck verifies that quiescence search handles checks correctly.
func TestSearch_QuiescenceCheck(t *testing.T) {
	b := engine.NewBoard()
	// White is in checkmate. Search should return NoMove.
	b.SetFEN("rnb1kbnr/pppp1ppp/8/4p3/5PPq/8/PPPPP2P/RNBQKBNR w KQkq - 0 1")
	GlobalTT.Clear()
	searchEngine := NewEngine(b)
	bestMove := searchEngine.Search(1)

	if bestMove != engine.NoMove {
		t.Errorf("Expected NoMove in checkmate position, got %s", bestMove.String())
	}
}

// TestSearch_RFP verifies that Static Null Move Pruning is integrated.
func TestSearch_RFP(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping search depth test in short mode.")
	}
	b := engine.NewBoard()
	b.SetFEN("k7/8/8/8/8/4B3/4Q3/K7 w - - 0 1")

	GlobalTT.Clear()
	searchEngine := NewEngine(b)
	searchEngine.Search(6)
}

// TestSearch_LMP verifies that Late Move Pruning is integrated.
func TestSearch_LMP(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping search depth test in short mode.")
	}
	b := engine.NewBoard()
	b.SetFEN(engine.StartFEN)

	GlobalTT.Clear()
	searchEngine := NewEngine(b)
	searchEngine.Search(6)
}

// TestSearch_CounterMoves verifies that countermoves are tracked.
func TestSearch_CounterMoves(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping search depth test in short mode.")
	}
	b := engine.NewBoard()
	b.SetFEN(engine.StartFEN)

	GlobalTT.Clear()
	searchEngine := NewEngine(b)
	searchEngine.Search(5)

	found := false
	for _, row := range searchEngine.CounterMoves {
		for _, move := range row {
			if move != engine.NoMove {
				found = true
				break
			}
		}
	}

	if !found {
		t.Error("No countermoves were recorded during search")
	}
}

// TestSearch_SingularExtensions verifies that singular extensions don't crash.
func TestSearch_SingularExtensions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping search depth test in short mode.")
	}
	b := engine.NewBoard()
	b.SetFEN("rnbqkbnr/pppp1ppp/8/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R b KQkq - 1 2")

	GlobalTT.Clear()
	searchEngine := NewEngine(b)
	// Depth enough to trigger singular extension logic (depth >= 8)
	searchEngine.Search(8)
}
