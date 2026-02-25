package engine

import (
	"bytes"
	"testing"

	"github.com/Skiddle-Labs/axon-engine/internal/types"
)

// TestPackingConsistency verifies that packing and unpacking a board state
// preserves all relevant information including pieces, side, castling, and EP.
func TestPackingConsistency(t *testing.T) {
	tests := []struct {
		name   string
		fen    string
		score  int
		result int
	}{
		{"Startpos", StartFEN, 10, 0},
		{"Midgame", "r1bqkbnr/pppp1ppp/2n5/4p3/2B1P3/5Q2/PPPP1PPP/RNB1K1NR w KQkq - 0 1", -50, 1},
		{"En Passant", "rnbqkbnr/ppp1pppp/8/3pP3/8/8/PPPP1PPP/RNBQKBNR w KQkq d6 0 1", 100, -1},
		{"Endgame", "8/8/4k3/8/8/4K3/8/8 w - - 0 1", 0, 0},
		{"Castling Rights", "r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R w KQkq - 0 1", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBoard()
			if err := b.SetFEN(tt.fen); err != nil {
				t.Fatalf("Failed to set FEN: %v", err)
			}

			packed := b.Pack(tt.score, tt.result)
			unpackedBoard, unpackedScore, unpackedResult := packed.Unpack()

			// Check Score and Result
			if unpackedScore != tt.score {
				t.Errorf("Score mismatch: expected %d, got %d", tt.score, unpackedScore)
			}
			if unpackedResult != tt.result {
				t.Errorf("Result mismatch: expected %d, got %d", tt.result, unpackedResult)
			}

			// Check basic board state
			if unpackedBoard.SideToMove != b.SideToMove {
				t.Errorf("SideToMove mismatch: expected %v, got %v", b.SideToMove, unpackedBoard.SideToMove)
			}
			if unpackedBoard.Castling != b.Castling {
				t.Errorf("Castling mismatch: expected %v, got %v", b.Castling, unpackedBoard.Castling)
			}
			if unpackedBoard.EnPassant != b.EnPassant {
				t.Errorf("EnPassant mismatch: expected %v, got %v", b.EnPassant, unpackedBoard.EnPassant)
			}

			// Check pieces
			for s := 0; s < 64; s++ {
				sq := types.Square(s)
				if unpackedBoard.PieceAt(sq) != b.PieceAt(sq) {
					t.Errorf("Piece mismatch at %v: expected %v, got %v", sq, b.PieceAt(sq), unpackedBoard.PieceAt(sq))
				}
			}

			// Check Hash (this proves RefreshAccumulators and ComputeHash were called correctly)
			if unpackedBoard.Hash != b.Hash {
				t.Errorf("Hash mismatch: expected %v, got %v", b.Hash, unpackedBoard.Hash)
			}
		})
	}
}

// TestSerialization verifies that the PackedPos struct can be correctly serialized
// to and deserialized from a byte stream.
func TestSerialization(t *testing.T) {
	b := NewBoard()
	b.SetFEN(StartFEN)
	packed := b.Pack(123, 1)

	var buf bytes.Buffer
	if err := packed.Serialize(&buf); err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	if buf.Len() != PackedPosSize {
		t.Errorf("Expected buffer length %d, got %d", PackedPosSize, buf.Len())
	}

	deserialized, err := Deserialize(&buf)
	if err != nil {
		t.Fatalf("Failed to deserialize: %v", err)
	}

	if deserialized.Score != packed.Score {
		t.Errorf("Score mismatch after serialization: expected %d, got %d", packed.Score, deserialized.Score)
	}
	if deserialized.Occupancy != packed.Occupancy {
		t.Errorf("Occupancy mismatch after serialization")
	}
	if deserialized.Castling != packed.Castling {
		t.Errorf("Castling mismatch after serialization")
	}
	if deserialized.Result != packed.Result {
		t.Errorf("Result mismatch after serialization")
	}
}
