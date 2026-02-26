package nnue

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"io"
	"os"

	"github.com/Skiddle-Labs/axon-engine/internal/types"
)

//go:embed embedded.nnue
var embeddedNetwork []byte

// Network contains the weights and biases for the neural network.
// This follows a standard HalfKP architecture with a single hidden layer (Accumulator).
type Network struct {
	// Feature Transformer (Input -> Hidden)
	// Weights mapping 768 input features (12 pieces * 64 squares) to 256 hidden neurons.
	FeatureWeights [types.InputFeatures][types.L1Size]int16
	FeatureBiases  [types.L1Size]int16

	// Output Layer (Hidden -> Output)
	// Weights mapping the concatenated white and black accumulators (256 * 2) to a single output.
	// Quantized by QB = 64.
	OutputWeights [types.L1Size * 2]int16
	// OutputBias is quantized by QA * QB = 16320.
	// We use int16 to match the trainer's quantization format (i16).
	OutputBias int16
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

// NetworkName is the name of the currently loaded network file.
var NetworkName = "none"

// UseNNUE allows enabling/disabling NNUE evaluation at runtime.
var UseNNUE = true

func init() {
	// If an embedded network is provided, load it by default.
	if len(embeddedNetwork) > 0 {
		if err := LoadNetworkFromBytes(embeddedNetwork); err == nil {
			NetworkName = fmt.Sprintf("axon-hashed-%08x", GetHash(embeddedNetwork))
		}
	}
}

// GetHash returns the FNV-1a 32-bit hash of the network data.
// This is used to uniquely identify the network version.
func GetHash(data []byte) uint32 {
	h := fnv.New32a()
	h.Write(data)
	return h.Sum32()
}

// LoadNetwork loads the NNUE weights and biases from a binary file.
func LoadNetwork(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	// Verify file size. Axon's 768 -> 256 architecture expects ~385 KB.
	// Stockfish networks are much larger (20-80 MB) and use a different architecture.
	if info.Size() > 1024*1024 {
		return fmt.Errorf("network file too large (%d bytes); Stockfish networks are not compatible with Axon's HalfKP 768->256 architecture", info.Size())
	}

	net, err := readNetwork(file)
	if err != nil {
		return fmt.Errorf("failed to load %s: %w", path, err)
	}

	CurrentNetwork = net
	NetworkName = path
	return nil
}

// LoadNetworkFromBytes loads the NNUE weights and biases from a byte slice.
func LoadNetworkFromBytes(data []byte) error {
	if len(data) > 1024*1024 {
		return fmt.Errorf("network data too large (%d bytes); likely an incompatible architecture", len(data))
	}

	net, err := readNetwork(bytes.NewReader(data))
	if err != nil {
		return err
	}

	CurrentNetwork = net
	return nil
}

// readNetwork reads the network structure from an io.Reader.
func readNetwork(r io.Reader) (*Network, error) {
	net := &Network{}

	// Load Feature Weights
	for i := 0; i < types.InputFeatures; i++ {
		if err := binary.Read(r, binary.LittleEndian, &net.FeatureWeights[i]); err != nil {
			return nil, fmt.Errorf("error reading feature weights at index %d: %w", i, err)
		}
	}

	// Load Feature Biases
	if err := binary.Read(r, binary.LittleEndian, &net.FeatureBiases); err != nil {
		return nil, fmt.Errorf("error reading feature biases: %w", err)
	}

	// Load Output Weights
	if err := binary.Read(r, binary.LittleEndian, &net.OutputWeights); err != nil {
		return nil, fmt.Errorf("error reading output weights: %w", err)
	}

	// Load Output Bias
	if err := binary.Read(r, binary.LittleEndian, &net.OutputBias); err != nil {
		return nil, fmt.Errorf("error reading output bias: %w", err)
	}

	return net, nil
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

	// Use int64 for the intermediate sum to prevent overflow.
	// With 512 inputs, each being up to 255^2 * 32767, the total sum can
	// reach ~1.1e12, which exceeds the range of a 32-bit integer (~2.1e9).
	var output int64 = 0

	// 1. Process 'us' perspective
	for i := 0; i < types.L1Size; i++ {
		val := int32(us[i])
		if val > 0 {
			if val > 255 {
				val = 255
			}
			// SCReLU: clamp(x, 0, 255)^2
			output += int64(val) * int64(val) * int64(CurrentNetwork.OutputWeights[i])
		}
	}

	// 2. Process 'them' perspective
	for i := 0; i < types.L1Size; i++ {
		val := int32(them[i])
		if val > 0 {
			if val > 255 {
				val = 255
			}
			output += int64(val) * int64(val) * int64(CurrentNetwork.OutputWeights[types.L1Size+i])
		}
	}

	// Quantization constants:
	// QA is the quantization of the squared activation (255).
	// QB is the quantization of the output layer weights (64).
	const QA = 255
	const QB = 64
	const QAB = QA * QB

	// eval_scale used in training was 400.
	// The internal score is in a scale where QA * QB represents 1.0 (internal units).
	// We convert this to centipawns by scaling by the EvalScale.
	const EvalScale = 400

	internalScore := (output / QA) + int64(CurrentNetwork.OutputBias)

	return int(internalScore * EvalScale / QAB)
}
