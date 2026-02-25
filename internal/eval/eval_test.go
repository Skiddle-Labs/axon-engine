package eval

import (
	"testing"

	"github.com/Skiddle-Labs/axon-engine/internal/engine"
)

// TestEvaluate_StartingPosition verifies that the starting position is evaluated as 0 (balanced).
func TestEvaluate_StartingPosition(t *testing.T) {
	b := engine.NewBoard()
	b.SetFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	score := Evaluate(b)

	// In the exact starting position, with symmetrical PSTs and material, the score should be 0.
	if score != 0 {
		t.Errorf("Expected score 0 for starting position, got %d", score)
	}
}

// TestEvaluate_MaterialAdvantage verifies that white having an extra queen results in a positive score.
func TestEvaluate_MaterialAdvantage(t *testing.T) {
	b := engine.NewBoard()
	// Starting position but black is missing a queen
	b.SetFEN("rn2kbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	score := Evaluate(b)

	if score <= 0 {
		t.Errorf("Expected positive score when White has material advantage, got %d", score)
	}
}

// TestEvaluate_PawnStructure verifies that doubled pawns are penalized.
func TestEvaluate_PawnStructure(t *testing.T) {
	b1 := engine.NewBoard()
	b2 := engine.NewBoard()

	// b1: Doubled pawns on A file
	b1.SetFEN("k7/8/8/8/8/P7/P7/K7 w - - 0 1")
	// b2: Connected pawns on 2nd rank
	b2.SetFEN("k7/8/8/8/8/8/PP6/K7 w - - 0 1")

	// Note: SetFEN calculates hash and state.
	// We want to compare the evaluation of the two positions.
	score1 := Evaluate(b1)
	score2 := Evaluate(b2)

	if score2 <= score1 {
		t.Errorf("Expected connected pawns (score:%d) to be better than doubled pawns (score:%d)", score2, score1)
	}
}

// TestEvaluate_BishopPair verifies the bishop pair bonus.
func TestEvaluate_BishopPair(t *testing.T) {
	b := engine.NewBoard()
	// Position with only two bishops for white
	b.SetFEN("k7/8/8/8/8/8/8/K1BB4 w - - 0 1")

	pmg, peg := evaluatePawnStructure(b, engine.White)
	mg, eg := evaluateColor(b, engine.White, pmg, peg)

	// mg and eg should include the bishop values + PST + mobility + bishop pair bonus
	// BishopMG is 365. 365 * 2 = 730. Bishop pair bonus is 30. King PST A1 is -30.
	if mg < 750 {
		t.Errorf("Midgame score %d too low, likely missing Bishop Pair bonus", mg)
	}

	if eg < 610 { // BishopEG is 297. 297 * 2 = 594. Bonus 50. King PST A1 is -50.
		t.Errorf("Endgame score %d too low, likely missing Bishop Pair bonus", eg)
	}
}

// TestEvaluate_Tapering verifies that the evaluation changes based on game phase.
func TestEvaluate_Tapering(t *testing.T) {
	b := engine.NewBoard()

	// Midgame position (lots of pieces)
	b.SetFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
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

	pmg1, peg1 := evaluatePawnStructure(b1, engine.White)
	mg1, _ := evaluateColor(b1, engine.White, pmg1, peg1)
	pmg2, peg2 := evaluatePawnStructure(b2, engine.White)
	mg2, _ := evaluateColor(b2, engine.White, pmg2, peg2)

	if mg1 <= mg2 {
		t.Errorf("Safe king (MG:%d) should evaluate higher than exposed king (MG:%d)", mg1, mg2)
	}
}

// TestEvaluate_Threats verifies that hanging pieces and bad trades are penalized.
func TestEvaluate_Threats(t *testing.T) {
	// 1. Hanging Piece: White Knight attacked by Black Pawn, no defenders
	b1 := engine.NewBoard()
	b1.SetFEN("k7/8/8/3p4/4N3/8/8/K7 w - - 0 1")
	pmg1, peg1 := evaluatePawnStructure(b1, engine.White)
	mg1, eg1 := evaluateColor(b1, engine.White, pmg1, peg1)

	// Same position but Knight is not attacked
	b2 := engine.NewBoard()
	b2.SetFEN("k7/8/8/8/4N3/8/8/K7 w - - 0 1")
	pmg2, peg2 := evaluatePawnStructure(b2, engine.White)
	mg2, eg2 := evaluateColor(b2, engine.White, pmg2, peg2)

	if mg1 >= mg2 {
		t.Errorf("Hanging piece should be penalized in midgame. mg1:%d, mg2:%d", mg1, mg2)
	}
	if eg1 >= eg2 {
		t.Errorf("Hanging piece should be penalized in endgame. eg1:%d, eg2:%d", eg1, eg2)
	}

	// 2. Bad Trade: White Rook defended by Pawn, but attacked by Black Pawn
	b3 := engine.NewBoard()
	b3.SetFEN("k7/8/8/3p4/4R3/4P3/8/K7 w - - 0 1")
	pmg3, peg3 := evaluatePawnStructure(b3, engine.White)
	mg3, eg3 := evaluateColor(b3, engine.White, pmg3, peg3)

	// Same position but Rook is not attacked
	b4 := engine.NewBoard()
	b4.SetFEN("k7/8/8/8/4R3/4P3/8/K7 w - - 0 1")
	pmg4, peg4 := evaluatePawnStructure(b4, engine.White)
	mg4, eg4 := evaluateColor(b4, engine.White, pmg4, peg4)

	if mg3 >= mg4 {
		t.Errorf("Bad trade (Rook vs Pawn) should be penalized in midgame. mg3:%d, mg4:%d", mg3, mg4)
	}
	if eg3 >= eg4 {
		t.Errorf("Bad trade (Rook vs Pawn) should be penalized in endgame. eg3:%d, eg4:%d", eg3, eg4)
	}
}
