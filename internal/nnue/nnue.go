package nnue

import (
	"github.com/Skiddle-Labs/axon-engine/internal/types"
)

// NNUE Architecture Constants
const (
	// OutputSize is 1 (the evaluation score)
	OutputSize = 1
)

// Network contains the weights and biases for the entire neural network.
type Network struct {
	// Feature Transformer (Input -> Hidden)
	FeatureWeights [types.InputFeatures][types.L1Size]int16
	FeatureBiases  [types.L1Size]int16

	// Output Layer (Hidden -> Output)
	// This can be expanded to multiple layers if using a larger architecture.
	OutputWeights [types.L1Size * 2]int16 // *2 because we concatenate White and Black perspectives
	OutputBias    int32
}

// Feature index mapping constants
const (
	PawnIdx   = 0
	KnightIdx = 1
	BishopIdx = 2
	RookIdx   = 3
	QueenIdx  = 4
	KingIdx   = 5
)

// GetFeatureIndex returns the index in the feature vector for a given piece and square.
// The index is calculated as: color * 384 + pieceType * 64 + square
func GetFeatureIndex(p types.Piece, sq types.Square) int {
	color := int(p.Color())
	pType := int(p.Type()) - 1 // Pawn is 1, so 1-1=0
	return color*384 + pType*64 + int(sq)
}

// CurrentNetwork is the global instance of the loaded NNUE weights.
var CurrentNetwork *Network

// LoadNetwork would be used to load weights from a binary file.
func LoadNetwork(path string) error {
	// TODO: Implement binary weight loading
	return nil
}

// Evaluate performs the forward pass of the neural network.
func Evaluate(whiteAcc, blackAcc *types.Accumulator) int {
	// This will eventually implement the SCReLU activation and the final linear layer.
	// For now, it's a placeholder.
	return 0
}
