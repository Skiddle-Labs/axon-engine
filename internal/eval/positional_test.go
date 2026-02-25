package eval

import (
	"testing"

	"github.com/Skiddle-Labs/axon-engine/internal/engine"
	"github.com/Skiddle-Labs/axon-engine/internal/types"
)

// TestEvaluate_Space verifies that controlling central squares without enemy pawn attacks gives a bonus.
func TestEvaluate_Space(t *testing.T) {
	b1 := engine.NewBoard()
	// Empty board - white should have full space in the central 3x4 grid (c-f files, 2-4 ranks)
	b1.SetFEN("8/8/8/8/8/8/8/8 w - - 0 1")
	s1 := evaluateSpace(b1, types.White)

	b2 := engine.NewBoard()
	// Black pawns on d5 and e5 attack d4 and e4, reducing white's "safe" space
	b2.SetFEN("8/8/8/3pp3/8/8/8/8 w - - 0 1")
	s2 := evaluateSpace(b2, types.White)

	if s1 <= s2 {
		t.Errorf("Expected more space bonus for empty board (%d) than when black pawns attack center (%d)", s1, s2)
	}

	if s1 == 0 {
		t.Error("Space bonus should be non-zero for empty board")
	}
}

// TestEvaluate_OppositeBishops verifies that the evaluation is scaled down in OCB endgames.
func TestEvaluate_OppositeBishops(t *testing.T) {
	// Position with 1 pawn each and opposite bishops
	b1 := engine.NewBoard()
	b1.SetFEN("4b3/8/4k3/8/3P4/4K3/8/4B3 w - - 0 1") // White up a pawn, but OCB

	if !isOppositeBishops(b1) {
		t.Fatal("Position should be recognized as Opposite Colored Bishops")
	}

	score := 100
	scaled := scaleEndgame(b1, score)

	if scaled >= score {
		t.Errorf("Expected score to be scaled down in OCB endgame. Original: %d, Scaled: %d", score, scaled)
	}
}

// TestEvaluate_InsufficientMaterial verifies basic draw detection.
func TestEvaluate_InsufficientMaterial(t *testing.T) {
	tests := []struct {
		fen      string
		expected bool
	}{
		{"8/8/4k3/8/8/4K3/8/8 w - - 0 1", true},    // K vs K
		{"8/8/4k3/8/8/4K3/4N3/8 w - - 0 1", true},  // KN vs K
		{"8/8/4k3/8/8/4K3/4B3/8 w - - 0 1", true},  // KB vs K
		{"8/8/4k3/8/4p3/4K3/8/8 w - - 0 1", false}, // K vs KP
		{"8/8/4k3/8/8/4K3/4R3/8 w - - 0 1", false}, // KR vs K
	}

	for _, tt := range tests {
		b := engine.NewBoard()
		b.SetFEN(tt.fen)
		if got := isInsufficientMaterial(b); got != tt.expected {
			t.Errorf("isInsufficientMaterial(%s) = %v; want %v", tt.fen, got, tt.expected)
		}
	}
}

// TestEvaluate_BishopKnightScaling verifies that piece values are adjusted based on pawn count.
func TestEvaluate_BishopKnightScaling(t *testing.T) {
	// This test ensures the scaling logic is called and affects the score.
	b1 := engine.NewBoard()
	b1.SetFEN("k7/8/8/8/8/8/8/K1B5 w - - 0 1") // 0 pawns

	b2 := engine.NewBoard()
	b2.SetFEN("k7/pppppppp/8/8/8/8/PPPPPPPP/K1B5 w - - 0 1") // 16 pawns (capped at 8 in code)

	// evaluateColor returns (mg, eg)
	mg1, _ := evaluateColor(b1, types.White, 0, 0)
	mg2, _ := evaluateColor(b2, types.White, 0, 0)

	if mg1 == mg2 {
		t.Errorf("Expected different bishop evaluation for different pawn counts. Got %d for both.", mg1)
	}

	// Also check knights
	b3 := engine.NewBoard()
	b3.SetFEN("k7/8/8/8/8/8/8/K1N5 w - - 0 1")
	b4 := engine.NewBoard()
	b4.SetFEN("k7/pppppppp/8/8/8/8/PPPPPPPP/K1N5 w - - 0 1")

	mg3, _ := evaluateColor(b3, types.White, 0, 0)
	mg4, _ := evaluateColor(b4, types.White, 0, 0)

	if mg3 == mg4 {
		t.Errorf("Expected different knight evaluation for different pawn counts. Got %d for both.", mg3)
	}
}
