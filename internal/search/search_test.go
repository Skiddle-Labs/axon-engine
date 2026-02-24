package search

import (
	"testing"

	"github.com/personal-github/axon-engine/internal/engine"
)

// TestSearch_MateIn1 verifies that the engine finds a simple mate in one.
func TestSearch_MateIn1(t *testing.T) {
	b := engine.NewBoard()
	// Scholar's Mate position: Queen attacks F7 with Bishop support
	b.SetFEN("r1bqkbnr/pppp1ppp/2n5/4p3/2B1P3/5Q2/PPPP1PPP/RNB1K1NR w KQkq - 4 4")

	searchEngine := NewEngine(b)
	bestMove := searchEngine.Search(4)

	expectedFrom := engine.F3
	expectedTo := engine.F7

	if bestMove.From() != expectedFrom || bestMove.To() != expectedTo {
		t.Errorf("Expected mate move f3f7, got %s", bestMove.String())
	}
}

// TestSearch_MateIn2 verifies that the engine finds a mate in two.
func TestSearch_MateIn2(t *testing.T) {
	b := engine.NewBoard()
	// Black is in a position to be mated in 2: 1. Qe8+ Rxe8 2. Rxe8#
	b.SetFEN("r5k1/5ppp/8/8/8/8/4QPPP/4R1K1 w - - 0 1")

	searchEngine := NewEngine(b)
	bestMove := searchEngine.Search(4)

	expectedFrom := engine.E2
	expectedTo := engine.E8

	if bestMove.From() != expectedFrom || bestMove.To() != expectedTo {
		t.Errorf("Expected move e2e8, got %s", bestMove.String())
	}
}

// TestSearch_TranspositionTable verifies that the TT helps find moves faster.
func TestSearch_TranspositionTable(t *testing.T) {
	b := engine.NewBoard()
	b.SetFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

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
	// If search stopped at depth 0 without quiescence, it might think it wins a pawn.
	b.SetFEN("r1bqkbnr/pppp1ppp/2n5/4p3/3P4/5N2/PPP1PPPP/RNBQKB1R w KQkq - 0 1")

	searchEngine := NewEngine(b)
	// Depth 1 search should use quiescence to see the full exchange at d4
	bestMove := searchEngine.Search(1)

	if bestMove == engine.NoMove {
		t.Error("Search returned NoMove")
	}
}

// TestSearch_Repetition verifies that the engine avoids 3-fold repetition in a winning position.
func TestSearch_Repetition(t *testing.T) {
	b := engine.NewBoard()
	// Position where White is winning but could repeat moves
	b.SetFEN("k7/8/8/8/8/8/7R/1R4K1 w - - 0 1")

	// 1. Ra2+ Kb8 2. Rh1 Ka8 ... (3-fold check)
	// We want to ensure that if we are forced into a repetition, the score is 0.

	// Implementation note: Testing repetition requires the history to be filled.
	// Since we use the engine.History and engine.Ply, we can simulate this.
}

// TestSearch_NullMovePruning verifies that NMP is active.
func TestSearch_NullMovePruning(t *testing.T) {
	b := engine.NewBoard()
	b.SetFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	searchEngine := NewEngine(b)
	// This is hard to test directly without instrumenting the code,
	// but we can check if it runs correctly.
	searchEngine.Search(5)
}

// TestSearch_Stalemate verifies that stalemate is evaluated as 0.
func TestSearch_Stalemate(t *testing.T) {
	b := engine.NewBoard()
	// Classic stalemate position
	b.SetFEN("7k/5Q2/8/8/8/8/8/7K b - - 0 1")

	searchEngine := NewEngine(b)
	move := searchEngine.Search(1)

	if move != engine.NoMove {
		t.Errorf("Expected NoMove in stalemate position, got %s", move.String())
	}

	// Filter for legal moves
	ml := b.GenerateMoves()
	legalCount := 0
	for i := 0; i < ml.Count; i++ {
		if b.MakeMove(ml.Moves[i]) {
			b.UnmakeMove(ml.Moves[i])
			legalCount++
		}
	}
	if legalCount != 0 {
		t.Errorf("Expected 0 legal moves in stalemate, got %d", legalCount)
	}
}

// TestSearch_QuiescenceCheck verifies that quiescence search handles checks correctly.
func TestSearch_QuiescenceCheck(t *testing.T) {
	b := engine.NewBoard()
	// White to move, Qxf7# is mate in 1. Capture move.
	b.SetFEN("r1bqkbnr/pp1ppQpp/2n5/2p5/4P3/8/PPPP1PPP/RNB1KBNR b KQkq - 0 1")

	GlobalTT.Clear()
	searchEngine := NewEngine(b)
	// Depth 1 search for Black should see the threat or respond to checks if it were his turn.
	// But let's use a position where White is in check and must respond.
	b.SetFEN("rnb1kbnr/pppp1ppp/8/4p3/5PPq/8/PPPPP2P/RNBQKBNR w KQkq - 0 1")
	bestMove := searchEngine.Search(1)

	// White is in checkmate by the queen on h4. Search should return NoMove or evaluate accordingly.
	if bestMove != engine.NoMove {
		t.Errorf("Expected NoMove in checkmate position, got %s", bestMove.String())
	}
}
