package main

import (
	"math"
	"testing"

	"github.com/Skiddle-Labs/axon-engine/internal/engine"
)

// TestSigmoid verifies that the sigmoid function correctly maps scores to results.
func TestSigmoid(t *testing.T) {
	k := 1.0

	tests := []struct {
		score     float64
		expected  float64
		tolerance float64
	}{
		{0, 0.5, 1e-9},
		{10000, 1.0, 1e-6},
		{-10000, 0.0, 1e-6},
		{400, 0.909090909, 1e-6}, // 1 / (1 + 10^-1) = 1 / 1.1 = 0.909...
	}

	for _, tt := range tests {
		got := Sigmoid(tt.score, k)
		if math.Abs(got-tt.expected) > tt.tolerance {
			t.Errorf("Sigmoid(score=%v, k=%v) = %v; want %v", tt.score, k, got, tt.expected)
		}
	}
}

// TestCalculateMSE ensures that the MSE calculation runs without error and returns plausible values.
func TestCalculateMSE(t *testing.T) {
	k := 1.0

	// Create a few test positions
	b1 := engine.NewBoard()
	b1.SetFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1") // Startpos

	b2 := engine.NewBoard()
	b2.SetFEN("rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1") // After 1. e4

	entries := []Entry{
		{board: b1, result: 0.5},
		{board: b2, result: 0.5},
	}
	precomputed := PrecomputeEntries(entries)

	mse := CalculateMSEParallel(precomputed, k)

	if math.IsNaN(mse) {
		t.Error("CalculateMSE returned NaN")
	}

	if mse < 0 || mse > 1.0 {
		t.Errorf("CalculateMSE returned out of bounds value: %v", mse)
	}
}

// TestFindBestK verifies the search for the optimal scaling constant.
func TestFindBestK(t *testing.T) {
	b1 := engine.NewBoard()
	b1.SetFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	entries := []Entry{
		{board: b1, result: 0.5},
	}
	precomputed := PrecomputeEntries(entries)

	bestK := FindBestK(precomputed)

	if bestK < 0.1 || bestK > 2.0 {
		t.Errorf("FindBestK returned value out of expected range [0.1, 2.0]: %v", bestK)
	}
}
