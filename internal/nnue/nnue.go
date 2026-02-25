package nnue

import (
	"encoding/binary"
	"os"

	"github.com/Skiddle-Labs/axon-engine/internal/types"
)

// Network contains the weights and biases for the neural network.
// This follows a standard HalfKP architecture with a single hidden layer (Accumulator).
type Network struct {
	// Feature Transformer (Input -> Hidden)
	// Weights mapping 768 input features (12 pieces * 64 squares) to 256 hidden neurons.
	FeatureWeights [types.InputFeatures][types.L1Size]int16
	FeatureBiases  [types.L1Size]int16

	// Output Layer (Hidden -> Output)
	// Weights mapping the concatenated white and black accumulators (256 * 2) to a single output.
	OutputWeights [types.L1Size * 2]int16
	OutputBias    int32
}

// GetFeatureIndex returns the index in the feature vector for a given piece and square.
// The index is calculated based on color, piece type, and square.
func GetFeatureIndex(p types.Piece, sq types.Square) int {
	color := int(p.Color())
	pType := int(p.Type()) - 1 // Piece types are 1-indexed (Pawn=1)
	if pType < 0 {
		return 0
	}
	return color*384 + pType*64 + int(sq)
}

// CurrentNetwork is the global instance of the loaded NNUE weights.
// If this is nil, the engine will fall back to Hand-Coded Evaluation.
var CurrentNetwork *Network

// UseNNUE allows enabling/disabling NNUE evaluation at runtime.
var UseNNUE = true

// LoadNetwork loads the NNUE weights and biases from a binary file.
// The file should contain raw little-endian values in the order:
// 1. FeatureWeights [768][256] int16
// 2. FeatureBiases  [256] int16
// 3. OutputWeights  [512] int16
// 4. OutputBias     int32
func LoadNetwork(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	net := &Network{}

	// Load Feature Weights
	for i := 0; i < types.InputFeatures; i++ {
		if err := binary.Read(file, binary.LittleEndian, &net.FeatureWeights[i]); err != nil {
			return err
		}
	}

	// Load Feature Biases
	if err := binary.Read(file, binary.LittleEndian, &net.FeatureBiases); err != nil {
		return err
	}

	// Load Output Weights
	if err := binary.Read(file, binary.LittleEndian, &net.OutputWeights); err != nil {
		return err
	}

	// Load Output Bias
	if err := binary.Read(file, binary.LittleEndian, &net.OutputBias); err != nil {
		return err
	}

	CurrentNetwork = net
	return nil
}

// Evaluate performs the forward pass of the neural network.
// It transforms the white and black accumulators using a SCReLU activation
// function and a final linear layer to produce a centipawn score.
func Evaluate(whiteAcc, blackAcc *types.Accumulator, side types.Color) int {
	if !UseNNUE || CurrentNetwork == nil {
		return 0
	}

	// Select accumulators based on perspective (us vs them)
	var us, them *types.Accumulator
	if side == types.White {
		us, them = whiteAcc, blackAcc
	} else {
		us, them = blackAcc, whiteAcc
	}

	var output int32 = 0

	// 1. Process 'us' perspective
	for i := 0; i < types.L1Size; i++ {
		val := int32(us[i])
		if val > 0 {
			if val > 127 {
				val = 127
			}
			// SCReLU: clamp(x, 0, 127)^2
			output += val * val * int32(CurrentNetwork.OutputWeights[i])
		}
	}

	// 2. Process 'them' perspective
	for i := 0; i < types.L1Size; i++ {
		val := int32(them[i])
		if val > 0 {
			if val > 127 {
				val = 127
			}
			output += val * val * int32(CurrentNetwork.OutputWeights[types.L1Size+i])
		}
	}

	// Quantization scaling:
	// QA is the quantization of the squared activation.
	// QB is the quantization of the final evaluation score.
	const QA = 255
	const QB = 64

	return int((output/QA + CurrentNetwork.OutputBias) / QB)
}
