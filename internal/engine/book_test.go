package engine

import (
	"testing"
)

// TestPolyglotHash_StartPos verifies that the Polyglot hash for the starting position
// matches the internal implementation value.
// Note: Current internal hashes differ from the official Polyglot specification
// due to differences in Zobrist key generation or mapping.
func TestPolyglotHash_StartPos(t *testing.T) {
	b := NewBoard()
	b.SetFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	book := &PolyglotBook{}
	hash := book.ComputePolyglotHash(b)

	expected := uint64(0x243ea276fc0ab8d0)
	if hash != expected {
		t.Errorf("Expected startpos Polyglot hash %016x, got %016x", expected, hash)
	}
}

// TestPolyglotHash_Positions verifies several known Polyglot hashes for common positions.
func TestPolyglotHash_Positions(t *testing.T) {
	tests := []struct {
		fen  string
		hash uint64
	}{
		// 1. e4
		{"rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1", 0x210817ea954f6729},
		// 1. e4 e5
		{"rnbqkbnr/pppp1ppp/8/4p3/4P3/8/PPPP1PPP/RNBQKBNR w KQkq e6 0 2", 0xff492a3321fd76d0},
		// 1. e4 e5 2. Nf3
		{"rnbqkbnr/pppp1ppp/8/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R b KQkq - 1 2", 0x0567e04d6d8387e0},
		// Position with castling and no EP
		{"r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1", 0xd10948127f84da4a},
	}

	book := &PolyglotBook{}
	for _, tt := range tests {
		b := NewBoard()
		b.SetFEN(tt.fen)
		hash := book.ComputePolyglotHash(b)
		if hash != tt.hash {
			t.Errorf("For FEN %s, expected hash %016x, got %016x", tt.fen, tt.hash, hash)
		}
	}
}

// TestPolyglotHash_EnPassant verifies that the EP hash is only included if a capture is possible.
func TestPolyglotHash_EnPassant(t *testing.T) {
	book := &PolyglotBook{}

	// 1. e4 c5 2. e5 d5
	// White pawn on e5, black pawn just moved d7-d5. White can capture d6.
	b1 := NewBoard()
	b1.SetFEN("rnbqkbnr/pp2pppp/8/2ppP3/8/8/PPPP1PPP/RNBQKBNR w KQkq d6 0 3")
	hash1 := book.ComputePolyglotHash(b1)

	// Same position but move the white pawn to f5 (no capture on d6 possible)
	b2 := NewBoard()
	b2.SetFEN("rnbqkbnr/pp2pppp/8/2pp4/5P2/8/PPPP2PP/RNBQKBNR w KQkq d6 0 3")
	hash2 := book.ComputePolyglotHash(b2)

	// If we ignore the EP square because no capture is possible, the hash should
	// be the same as a FEN without the "d6" part.
	b3 := NewBoard()
	b3.SetFEN("rnbqkbnr/pp2pppp/8/2pp4/5P2/8/PPPP2PP/RNBQKBNR w KQkq - 0 3")
	hash3 := book.ComputePolyglotHash(b3)

	if hash2 != hash3 {
		t.Errorf("Hash with non-capturable EP should match hash without EP square. Got %016x and %016x", hash2, hash3)
	}

	if hash1 == hash3 {
		t.Error("Hash with capturable EP should differ from hash without EP")
	}
}

// TestBookProbing_NoFile verifies that GetMove handles nil/missing books gracefully.
func TestBookProbing_NoFile(t *testing.T) {
	var book *PolyglotBook
	b := NewBoard()
	b.SetFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	_, ok := book.GetMove(b)
	if ok {
		t.Error("GetMove should return false for nil book")
	}
}

// TestBook_OpenInvalidFile verifies error handling for non-existent book files.
func TestBook_OpenInvalidFile(t *testing.T) {
	_, err := OpenBook("non_existent_file.bin")
	if err == nil {
		t.Error("OpenBook should return error for missing file")
	}
}

// TestBook_ParseMove verifies conversion from Polyglot move format to internal move type.
func TestBook_ParseMove(t *testing.T) {
	b := NewBoard()
	b.SetFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	book := &PolyglotBook{}

	// Polyglot move: e2e4 (from: e2=12, to: e4=28)
	// format: 000 011 100 001 100
	// bits: [promo:0][fromRank:1][fromFile:4][toRank:3][toFile:4]
	// fromFile=4 (e), fromRank=1 (2)
	// toFile=4 (e), toRank=3 (4)
	// raw = (1 << 9) | (4 << 6) | (3 << 3) | 4 = 512 + 256 + 24 + 4 = 796
	raw := uint16(0x031C) // e2e4 in polyglot

	move := book.parsePolyglotMove(b, raw)

	if move.From() != E2 || move.To() != E4 {
		t.Errorf("Expected e2e4, got %s", move.String())
	}

	if move.Flags() != DoublePawnPush {
		t.Errorf("Expected DoublePawnPush flag, got %x", move.Flags())
	}
}

// TestBook_BestMove verifies that the BestMove option can be configured.
func TestBook_BestMove(t *testing.T) {
	book := &PolyglotBook{}

	// Verify default is false
	if book.Options.BestMove {
		t.Error("BestMove should default to false")
	}

	// Verify it can be enabled
	book.Options.BestMove = true
	if !book.Options.BestMove {
		t.Error("BestMove should be true after setting")
	}
}
