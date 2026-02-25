//go:build amd64

package engine

import (
	"github.com/Skiddle-Labs/axon-engine/internal/types"
	"golang.org/x/sys/cpu"
)

var (
	hasAVX2 = false
)

//go:noescape
func piecesToCharsAVX2(pieces *types.Piece, chars *byte)

func init() {
	// Detect AVX2 support at runtime.
	// This allows the engine to select the most efficient piece-to-string
	// and bitboard kernels available on the host hardware.
	hasAVX2 = cpu.X86.HasAVX2
}

// PiecesToChars converts a slice of 64 pieces to their FEN character representations.
// It uses AVX2 SIMD if available, otherwise it falls back to a lookup-based Go loop.
func PiecesToChars(pieces []types.Piece, chars []byte) {
	if hasAVX2 && len(pieces) >= 64 && len(chars) >= 64 {
		piecesToCharsAVX2(&pieces[0], &chars[0])
	} else {
		const table = ".PNBRQKpnbrqk"
		for i := 0; i < 64; i++ {
			chars[i] = table[pieces[i]]
		}
	}
}
