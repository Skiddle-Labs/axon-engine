package eval

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

// TestEvaluate_StartingPosition verifies that the starting position HCE score includes Tempo.
func TestEvaluate_StartingPosition(t *testing.T) {
	withHCE(func() {
		b := engine.NewBoard()
		b.SetFEN(engine.StartFEN)

		score := Evaluate(b)

		// In the starting position, symmetry results in 0, but White gets TempoMG.
		if score != TempoMG {
			t.Errorf("Expected HCE score %d (Tempo) for starting position, got %d", TempoMG, score)
		}
	})
}

// TestEvaluate_NNUE_Loaded verifies that NNUE evaluation returns a non-zero score for the startpos (usually).
func TestEvaluate_NNUE_Loaded(t *testing.T) {
	if nnue.CurrentNetwork == nil {
		t.Skip("NNUE network not loaded, skipping NNUE test")
	}

	old := nnue.UseNNUE
	nnue.UseNNUE = true
	defer func() { nnue.UseNNUE = old }()

	b := engine.NewBoard()
	b.SetFEN(engine.StartFEN)

	score := Evaluate(b)
	// It's extremely unlikely a trained network evaluates the starting position as exactly 0.
	if score == 0 {
		t.Log("Note: NNUE evaluated startpos as 0, which is unusual but possible.")
	}
}

// TestEvaluate_MaterialAdvantage verifies that white having an extra queen results in a positive score.
func TestEvaluate_MaterialAdvantage(t *testing.T) {
	withHCE(func() {
		b := engine.NewBoard()
		// Starting position but black is missing a queen
		b.SetFEN("rn2kbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

		score := Evaluate(b)

		if score <= 0 {
			t.Errorf("Expected positive score when White has material advantage, got %d", score)
		}
	})
}

// TestEvaluate_PawnStructure verifies that doubled pawns are penalized in HCE.
func TestEvaluate_PawnStructure(t *testing.T) {
	withHCE(func() {
		b1 := engine.NewBoard()
		b2 := engine.NewBoard()

		// b1: Doubled pawns on A file
		b1.SetFEN("k7/8/8/8/8/P7/P7/K7 w - - 0 1")
		// b2: Connected pawns on 2nd rank
		b2.SetFEN("k7/8/8/8/8/8/PP6/K7 w - - 0 1")

		score1 := Evaluate(b1)
		score2 := Evaluate(b2)

		if score2 <= score1 {
			t.Errorf("Expected connected pawns (score:%d) to be better than doubled pawns (score:%d) in HCE", score2, score1)
		}
	})
}

// TestEvaluate_BishopPair verifies the bishop pair bonus.
func TestEvaluate_BishopPair(t *testing.T) {
	b := engine.NewBoard()
	// Position with only two bishops for white
	b.SetFEN("k7/8/8/8/8/8/8/K1BB4 w - - 0 1")

	pmg, peg := evaluatePawnStructure(b, types.White)
	mg, eg := evaluateColor(b, types.White, pmg, peg)

	// mg and eg should include the bishop values + PST + mobility + bishop pair bonus
	if mg < 700 {
		t.Errorf("Midgame score %d too low, likely missing Bishop Pair bonus", mg)
	}

	if eg < 610 {
		t.Errorf("Endgame score %d too low, likely missing Bishop Pair bonus", eg)
	}
}

// TestEvaluate_Tapering verifies that the evaluation changes based on game phase.
func TestEvaluate_Tapering(t *testing.T) {
	b := engine.NewBoard()

	// Midgame position (lots of pieces)
	b.SetFEN(engine.StartFEN)
	mgW1, egW1, _ := calculatePhase(b)

	// Endgame position (few pieces)
	b.SetFEN("k7/8/8/8/8/8/4P3/K7 w - - 0 1")
	mgW2, egW2, _ := calculatePhase(b)

	if mgW1 <= mgW2 {
		t.Errorf("Expected higher Midgame weight for starting position. Got MG1:%d, MG2:%d", mgW1, mgW2)
	}

	if egW2 <= egW1 {
		t.Errorf("Expected higher Endgame weight for empty engine. Got EG1:%d, EG2:%d", egW1, egW2)
	}
}

// TestEvaluate_KingSafety verifies that white king behind pawns is better than exposed.
func TestEvaluate_KingSafety(t *testing.T) {
	b1 := engine.NewBoard()
	b1.SetFEN("8/8/8/8/8/PPP5/1K6/8 w - - 0 1") // Safe behind pawns

	b2 := engine.NewBoard()
	b2.SetFEN("8/8/8/8/1K6/8/8/8 w - - 0 1") // Exposed

	pmg1, peg1 := evaluatePawnStructure(b1, types.White)
	mg1, _ := evaluateColor(b1, types.White, pmg1, peg1)
	pmg2, peg2 := evaluatePawnStructure(b2, types.White)
	mg2, _ := evaluateColor(b2, types.White, pmg2, peg2)

	if mg1 <= mg2 {
		t.Errorf("Safe king (MG:%d) should evaluate higher than exposed king (MG:%d)", mg1, mg2)
	}
}

// TestEvaluate_Threats verifies that hanging pieces and bad trades are penalized.
func TestEvaluate_Threats(t *testing.T) {
	// 1. Hanging Piece: White Knight attacked by Black Pawn, no defenders
	b1 := engine.NewBoard()
	b1.SetFEN("k7/8/8/3p4/4N3/8/8/K7 w - - 0 1")
	pmg1, peg1 := evaluatePawnStructure(b1, types.White)
	mg1, eg1 := evaluateColor(b1, types.White, pmg1, peg1)

	// Same position but Knight is not attacked
	b2 := engine.NewBoard()
	b2.SetFEN("k7/8/8/8/4N3/8/8/K7 w - - 0 1")
	pmg2, peg2 := evaluatePawnStructure(b2, types.White)
	mg2, eg2 := evaluateColor(b2, types.White, pmg2, peg2)

	if mg1 >= mg2 {
		t.Errorf("Hanging piece should be penalized in midgame. mg1:%d, mg2:%d", mg1, mg2)
	}
	if eg1 >= eg2 {
		t.Errorf("Hanging piece should be penalized in endgame. eg1:%d, eg2:%d", eg1, eg2)
	}
}
