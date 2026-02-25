package engine

import (
	"testing"

	"github.com/Skiddle-Labs/axon-engine/internal/types"
)

// TestPiecesToChars verifies that the SIMD (or fallback) implementation
// correctly maps piece types to their FEN character representations.
func TestPiecesToChars(t *testing.T) {
	var pieces [64]types.Piece
	var got [64]byte

	// 1. Test Empty Board
	for i := range pieces {
		pieces[i] = types.NoPiece
	}
	PiecesToChars(pieces[:], got[:])
	for i, b := range got {
		if b != '.' {
			t.Errorf("Empty board square %d: expected '.', got %c", i, b)
		}
	}

	// 2. Test Full Piece Range
	// Pieces are indexed 1-6 (White) and 7-12 (Black)
	expectedMapping := ".PNBRQKpnbrqk"
	for i := 0; i < 64; i++ {
		pieces[i] = types.Piece(i % 13)
	}

	PiecesToChars(pieces[:], got[:])

	for i, b := range got {
		expected := expectedMapping[i%13]
		if b != expected {
			t.Errorf("Square %d (piece %d): expected %c, got %c", i, i%13, expected, b)
		}
	}
}

// BenchmarkPiecesToChars measures the performance of the piece-to-char conversion.
func BenchmarkPiecesToChars(b *testing.B) {
	board := NewBoard()
	board.SetFEN(StartFEN)
	var got [64]byte

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		PiecesToChars(board.PieceArray[:], got[:])
	}
}
