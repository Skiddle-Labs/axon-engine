package engine

import (
	"testing"
)

// TestHashConsistency verifies that incremental hash updates in MakeMove/UnmakeMove
// always match the hash computed from scratch using ComputeHash.
func TestHashConsistency(t *testing.T) {
	tests := []struct {
		name string
		fen  string
	}{
		{"Starting Position", StartFEN},
		{"Midgame", "r1bqkbnr/pppp1ppp/2n5/4p3/2B1P3/5Q2/PPPP1PPP/RNB1K1NR w KQkq - 0 1"},
		{"En Passant Possible", "rnbqkbnr/ppp1pppp/8/3pP3/8/8/PPPP1PPP/RNBQKBNR w KQkq d6 0 2"},
		{"En Passant Impossible but marked", "rnbqkbnr/ppp1pppp/8/3p4/8/8/PPPPPPPP/RNBQKBNR w KQkq d6 0 1"},
		{"Castling available", "r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R w KQkq - 0 1"},
		{"Endgame", "8/8/4k3/8/8/4K3/8/8 w - - 0 1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBoard()
			if err := b.SetFEN(tt.fen); err != nil {
				t.Fatalf("Failed to set FEN: %v", err)
			}

			initialHash := b.Hash
			if initialHash != b.ComputeHash() {
				t.Errorf("Initial hash mismatch. Board.Hash: %v, ComputeHash(): %v", initialHash, b.ComputeHash())
			}

			moves := b.GenerateMoves()
			for i := 0; i < moves.Count; i++ {
				move := moves.Moves[i]
				if b.MakeMove(move) {
					if b.Hash != b.ComputeHash() {
						t.Errorf("Hash mismatch after MakeMove(%s). Board.Hash: %v, ComputeHash(): %v", move.String(), b.Hash, b.ComputeHash())
					}

					if b.PawnHash != b.ComputePawnHash() {
						t.Errorf("Pawn hash mismatch after MakeMove(%s). Board.PawnHash: %v, ComputePawnHash(): %v", move.String(), b.PawnHash, b.ComputePawnHash())
					}

					b.UnmakeMove(move)
					if b.Hash != initialHash {
						t.Errorf("Hash mismatch after UnmakeMove(%s). Expected %v, got %v", move.String(), initialHash, b.Hash)
					}
				}
			}
		})
	}
}

// TestEnPassantAwareHashing verifies the specialized logic where the En Passant file
// is only included in the Zobrist hash if a capture is actually possible.
func TestEnPassantAwareHashing(t *testing.T) {
	// 1. Case where EP capture is IMPOSSIBLE:
	// Position with EP square 'd6' but no White pawn can capture it.
	fenWithEP := "rnbqkbnr/ppp1pppp/8/3p4/8/8/PPPPPPPP/RNBQKBNR w KQkq d6 0 1"
	// Same position but with no EP square marked.
	fenNoEP := "rnbqkbnr/ppp1pppp/8/3p4/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

	b1 := NewBoard()
	b1.SetFEN(fenWithEP)

	b2 := NewBoard()
	b2.SetFEN(fenNoEP)

	if b1.Hash != b2.Hash {
		t.Errorf("Hashes should be identical when EP capture is impossible. Got %v and %v", b1.Hash, b2.Hash)
	}

	// 2. Case where EP capture IS possible:
	// White pawn on e5 can capture the d5 pawn via EP on d6.
	fenCapturePossible := "rnbqkbnr/ppp1pppp/8/3pP3/8/8/PPPP1PPP/RNBQKBNR w KQkq d6 0 1"
	// Same position but with EP square cleared.
	fenCaptureNoEP := "rnbqkbnr/ppp1pppp/8/3pP3/8/8/PPPP1PPP/RNBQKBNR w KQkq - 0 1"

	b3 := NewBoard()
	b3.SetFEN(fenCapturePossible)

	b4 := NewBoard()
	b4.SetFEN(fenCaptureNoEP)

	if b3.Hash == b4.Hash {
		t.Errorf("Hashes should be DIFFERENT when EP capture is possible. Both got %v", b3.Hash)
	}
}

// TestNullMoveHashing verifies hash consistency after null moves.
func TestNullMoveHashing(t *testing.T) {
	b := NewBoard()
	b.SetFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	initialHash := b.Hash
	b.MakeNullMove()

	if b.Hash == initialHash {
		t.Error("Hash should change after null move")
	}

	if b.Hash != b.ComputeHash() {
		t.Errorf("Null move incremental hash mismatch. Board.Hash: %v, ComputeHash(): %v", b.Hash, b.ComputeHash())
	}

	b.UnmakeNullMove()

	if b.Hash != initialHash {
		t.Errorf("Hash mismatch after unmaking null move. Expected %v, got %v", initialHash, b.Hash)
	}
}
