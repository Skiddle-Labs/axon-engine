package main

import (
	"bufio"
	"bytes"
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
		{
			name:     "Position with all castling",
			fen:      "r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R w KQkq - 0 1",
			expected: "r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R w KQkq -",
		},
		{
			name:     "Black to move and partial castling",
			fen:      "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR b Kq - 0 1",
			expected: "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR b Kq -",
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

func TestSaveToDisk(t *testing.T) {
	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)

	positions := []string{
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq -",
		"rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3",
	}

	// Test Win
	buf.Reset()
	SaveToDisk(writer, positions, ResultWin)
	writer.Flush()
	expected := positions[0] + " [1.0]\n" + positions[1] + " [1.0]\n"
	if buf.String() != expected {
		t.Errorf("SaveToDisk(Win) = %q; want %q", buf.String(), expected)
	}

	// Test Loss
	buf.Reset()
	SaveToDisk(writer, positions, ResultLoss)
	writer.Flush()
	expected = positions[0] + " [0.0]\n" + positions[1] + " [0.0]\n"
	if buf.String() != expected {
		t.Errorf("SaveToDisk(Loss) = %q; want %q", buf.String(), expected)
	}

	// Test Draw
	buf.Reset()
	SaveToDisk(writer, positions, ResultDraw)
	writer.Flush()
	expected = positions[0] + " [0.5]\n" + positions[1] + " [0.5]\n"
	if buf.String() != expected {
		t.Errorf("SaveToDisk(Draw) = %q; want %q", buf.String(), expected)
	}
}
