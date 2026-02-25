package nnue

import (
	"testing"

	"github.com/Skiddle-Labs/axon-engine/internal/types"
)

func TestGetFeatureIndex(t *testing.T) {
	// White Pawn on A1 (0)
	// Expected: color(0) * 384 + type(0) * 64 + square(0) = 0
	idx := GetFeatureIndex(types.WhitePawn, types.A1)
	if idx != 0 {
		t.Errorf("Expected White Pawn on A1 to be index 0, got %d", idx)
	}

	// White Knight on B1 (1)
	// Expected: color(0) * 384 + type(1) * 64 + square(1) = 65
	idx = GetFeatureIndex(types.WhiteKnight, types.B1)
	if idx != 65 {
		t.Errorf("Expected White Knight on B1 to be index 65, got %d", idx)
	}

	// Black King on H8 (63)
	// Expected: color(1) * 384 + type(5) * 64 + square(63) = 384 + 320 + 63 = 767
	idx = GetFeatureIndex(types.BlackKing, types.H8)
	if idx != 767 {
		t.Errorf("Expected Black King on H8 to be index 767, got %d", idx)
	}
}

func TestEvaluate_Basic(t *testing.T) {
	// Setup a simple network with known weights
	net := &Network{}

	// Set first output weight to 255 (QA)
	// Set bias to 64 * 10 (QB * 10)
	net.OutputWeights[0] = 255
	net.OutputBias = 64 * 10

	CurrentNetwork = net
	UseNNUE = true

	var accW, accB types.Accumulator

	// Test 1: Zero accumulators
	// Output = (0/255 + 640) / 64 = 10
	score := Evaluate(&accW, &accB, types.White)
	if score != 10 {
		t.Errorf("Expected score 10 for zero accumulators, got %d", score)
	}

	// Test 2: Active neuron
	// Set first neuron of 'us' (White) to 10
	// SCReLU(10) = 10^2 = 100
	// Output = (100 * 255 / 255 + 640) / 64 = (100 + 640) / 64 = 740 / 64 = 11
	accW[0] = 10
	score = Evaluate(&accW, &accB, types.White)
	if score != 11 {
		t.Errorf("Expected score 11 for active neuron, got %d", score)
	}

	// Test 3: Side to move flip
	// If side is Black, 'us' is accB and 'them' is accW
	// accB is zero, so Output = (0 + 640) / 64 = 10
	score = Evaluate(&accW, &accB, types.Black)
	if score != 10 {
		t.Errorf("Expected score 10 for Black perspective with empty accB, got %d", score)
	}

	// Test 4: Maximum activation
	// val = 200 (should be clamped to 127)
	// 127^2 = 16129
	// Output = (16129 * 255 / 255 + 640) / 64 = (16129 + 640) / 64 = 16769 / 64 = 262
	accW[0] = 200
	score = Evaluate(&accW, &accB, types.White)
	if score != 262 {
		t.Errorf("Expected score 262 for clamped maximum activation, got %d", score)
	}
}

func TestEvaluate_Disabled(t *testing.T) {
	CurrentNetwork = &Network{}
	UseNNUE = false
	var accW, accB types.Accumulator

	score := Evaluate(&accW, &accB, types.White)
	if score != 0 {
		t.Errorf("Expected score 0 when NNUE is disabled, got %d", score)
	}
}

func TestLoadNetwork_Missing(t *testing.T) {
	err := LoadNetwork("non_existent_file.nnue")
	if err == nil {
		t.Error("Expected error when loading missing network file, got nil")
	}
}

func TestEvaluate_SIMD_Consistency(t *testing.T) {
	if !hasAVX2 {
		t.Skip("AVX2 not supported on this hardware")
	}

	// Setup a network with random-ish but deterministic weights
	net := &Network{}
	for i := 0; i < types.L1Size*2; i++ {
		net.OutputWeights[i] = int16(i%127 - 64)
	}
	net.OutputBias = 1000
	CurrentNetwork = net
	UseNNUE = true

	var accW, accB types.Accumulator
	for i := 0; i < types.L1Size; i++ {
		accW[i] = int16(i%200 - 50)
		accB[i] = int16((i*3)%200 - 50)
	}

	// Calculate scores with both methods
	pureGoScore := Evaluate(&accW, &accB, types.White)
	simdScore := EvaluateForward(&accW, &accB, types.White)

	if pureGoScore != simdScore {
		t.Errorf("SIMD and pure Go evaluation mismatch: Go=%d, SIMD=%d", pureGoScore, simdScore)
	}

	// Test black perspective
	pureGoScoreB := Evaluate(&accW, &accB, types.Black)
	simdScoreB := EvaluateForward(&accW, &accB, types.Black)

	if pureGoScoreB != simdScoreB {
		t.Errorf("SIMD and pure Go evaluation mismatch (Black): Go=%d, SIMD=%d", pureGoScoreB, simdScoreB)
	}
}

func TestAccumulator_SIMD_Consistency(t *testing.T) {
	if !hasAVX2 {
		t.Skip("AVX2 not supported on this hardware")
	}

	var accGo, accSIMD types.Accumulator
	weights := make([]int16, types.L1Size)
	for i := 0; i < types.L1Size; i++ {
		weights[i] = int16(i%100 - 50)
		accGo[i] = int16(i % 10)
		accSIMD[i] = int16(i % 10)
	}

	// Test Update
	for i := 0; i < types.L1Size; i++ {
		accGo[i] += weights[i]
	}
	UpdateAccumulator(&accSIMD, weights)

	for i := 0; i < types.L1Size; i++ {
		if accGo[i] != accSIMD[i] {
			t.Fatalf("UpdateAccumulator mismatch at index %d: Go=%d, SIMD=%d", i, accGo[i], accSIMD[i])
		}
	}

	// Test Remove
	for i := 0; i < types.L1Size; i++ {
		accGo[i] -= weights[i]
	}
	RemoveAccumulator(&accSIMD, weights)

	for i := 0; i < types.L1Size; i++ {
		if accGo[i] != accSIMD[i] {
			t.Fatalf("RemoveAccumulator mismatch at index %d: Go=%d, SIMD=%d", i, accGo[i], accSIMD[i])
		}
	}
}
