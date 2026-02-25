//go:build amd64

package nnue

import (
	"github.com/Skiddle-Labs/axon-engine/internal/types"
	"golang.org/x/sys/cpu"
)

// Function prototypes for x64 assembly implementations.
// These are implemented in simd_x64.s to leverage AVX2 instructions.

//go:noescape
func updateAccumulatorAVX2(acc *types.Accumulator, weights *int16)

//go:noescape
func subAccumulatorAVX2(acc *types.Accumulator, weights *int16)

//go:noescape
func evaluateAVX2(us, them *types.Accumulator, weights *int16, bias int32) int32

var (
	hasAVX2 = false
)

func init() {
	// Detect AVX2 support at runtime.
	// This ensures the engine runs safely on older hardware while
	// maximizing performance on modern CPUs.
	hasAVX2 = cpu.X86.HasAVX2
}

// UpdateAccumulator adds feature weights to the L1 accumulator.
// Using AVX2 SIMD allows processing 16 weights (16x16-bit) in a single instruction,
// providing a massive performance boost during the search's incremental updates.
func UpdateAccumulator(acc *types.Accumulator, weights []int16) {
	if hasAVX2 {
		updateAccumulatorAVX2(acc, &weights[0])
	} else {
		// Fallback to pure Go implementation for non-AVX2 hardware.
		for i := 0; i < types.L1Size; i++ {
			acc[i] += weights[i]
		}
	}
}

// RemoveAccumulator subtracts feature weights from the L1 accumulator.
// Invoked during UnmakeMove or when a piece is captured to incrementally
// roll back the neural network's hidden layer state.
func RemoveAccumulator(acc *types.Accumulator, weights []int16) {
	if hasAVX2 {
		subAccumulatorAVX2(acc, &weights[0])
	} else {
		// Fallback to pure Go implementation.
		for i := 0; i < types.L1Size; i++ {
			acc[i] -= weights[i]
		}
	}
}

// EvaluateForward performs the NNUE forward pass (Hidden -> Output) with SIMD acceleration.
// It applies the SCReLU activation function and calculates the dot product
// between the white/black accumulators and the output layer weights.
func EvaluateForward(whiteAcc, blackAcc *types.Accumulator, side types.Color) int {
	if !UseNNUE || CurrentNetwork == nil {
		return 0
	}

	// Perspective selection: the network is evaluated from the perspective
	// of the side to move (Us vs Them).
	var us, them *types.Accumulator
	if side == types.White {
		us, them = whiteAcc, blackAcc
	} else {
		us, them = blackAcc, whiteAcc
	}

	if hasAVX2 {
		// Call assembly kernel for high-performance vectorized evaluation.
		// Returns the final centipawn score after internal quantization.
		res := evaluateAVX2(us, them, &CurrentNetwork.OutputWeights[0], CurrentNetwork.OutputBias)
		return int(res)
	}

	// Fallback to the optimized pure Go implementation.
	return Evaluate(whiteAcc, blackAcc, side)
}
