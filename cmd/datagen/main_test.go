package main

import (
	"testing"

	"github.com/Skiddle-Labs/axon-engine/internal/engine"
	"github.com/Skiddle-Labs/axon-engine/internal/types"
)

func TestGetPieceChar(t *testing.T) {
	tests := []struct {
		piece    types.Piece
		expected string
	}{
		{types.WhitePawn, "P"},
		{types.WhiteKnight, "N"},
		{types.WhiteBishop, "B"},
		{types.WhiteRook, "R"},
		{types.WhiteQueen, "Q"},
		{types.WhiteKing, "K"},
		{types.BlackPawn, "p"},
		{types.BlackKnight, "n"},
		{types.BlackBishop, "b"},
		{types.BlackRook, "r"},
		{types.BlackQueen, "q"},
		{types.BlackKing, "k"},
	}

	for _, tt := range tests {
		got := GetPieceChar(tt.piece)
		if got != tt.expected {
			t.Errorf("GetPieceChar(%v) = %q; want %q", tt.piece, got, tt.expected)
		}
	}
}

func TestGetEPDFEN(t *testing.T) {
	tests := []struct {
		name     string
		fen      string
		expected string
	}{
		{
			name:     "Starting Position",
			fen:      "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			expected: "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq -",
		},
		{
			name:     "After e4",
			fen:      "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
			expected: "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3",
		},
		{
			name:     "Middle Game No Castling",
			fen:      "r1bq1rk1/pp2ppbp/2np1np1/8/3NP3/2N1BP2/PPPQ2PP/R3KB1R w KQ - 3 9",
			expected: "r1bq1rk1/pp2ppbp/2np1np1/8/3NP3/2N1BP2/PPPQ2PP/R3KB1R w KQ -",
		},
		{
			name:     "Endgame",
			fen:      "8/8/4k3/8/8/4K3/8/8 w - - 0 1",
			expected: "8/8/4k3/8/8/4K3/8/8 w - -",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := engine.NewBoard()
			if err := b.SetFEN(tt.fen); err != nil {
				t.Fatalf("Failed to set FEN: %v", err)
			}
			got := GetEPDFEN(b)
			if got != tt.expected {
				t.Errorf("GetEPDFEN() = %q; want %q", got, tt.expected)
			}
		})
	}
}
