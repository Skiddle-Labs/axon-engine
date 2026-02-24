package engine

import (
	"testing"
)

// TestMoveGen_StartPos verifies the number of legal moves in the starting position.
func TestMoveGen_StartPos(t *testing.T) {
	b := NewBoard()
	b.SetFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	ml := b.GenerateMoves()
	legalCount := 0
	for i := 0; i < ml.Count; i++ {
		if b.MakeMove(ml.Moves[i]) {
			b.UnmakeMove(ml.Moves[i])
			legalCount++
		}
	}

	if legalCount != 20 {
		t.Errorf("Expected 20 legal moves in startpos, got %d", legalCount)
	}
}

// TestMoveGen_Kiwipete verifies the number of legal moves in the famous "Kiwipete" position (Perft 2).
func TestMoveGen_Kiwipete(t *testing.T) {
	b := NewBoard()
	b.SetFEN("r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1")

	ml := b.GenerateMoves()
	legalCount := 0
	for i := 0; i < ml.Count; i++ {
		if b.MakeMove(ml.Moves[i]) {
			b.UnmakeMove(ml.Moves[i])
			legalCount++
		}
	}

	// Perft(1) for Kiwipete is 48
	if legalCount != 48 {
		t.Errorf("Expected 48 legal moves in Kiwipete, got %d", legalCount)
	}
}

// TestMoveGen_EnPassant verifies legal en passant captures.
func TestMoveGen_EnPassant(t *testing.T) {
	b := NewBoard()
	// White pawn on e5, Black pawn moves d7-d5
	b.SetFEN("rnbqkbnr/pp2pppp/8/2ppP3/8/8/PPPP1PPP/RNBQKBNR w KQkq d6 0 3")

	ml := b.GenerateMoves()
	foundEP := false
	for i := 0; i < ml.Count; i++ {
		if ml.Moves[i].Flags() == EnPassantFlag {
			if ml.Moves[i].From() == E5 && ml.Moves[i].To() == D6 {
				foundEP = true
				break
			}
		}
	}

	if !foundEP {
		t.Error("En passant move e5d6 not found in move list")
	}
}

// TestMoveGen_Castling verifies legal castling moves.
func TestMoveGen_Castling(t *testing.T) {
	b := NewBoard()
	// Position with all castling available
	b.SetFEN("r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1")

	ml := b.GenerateMoves()
	ks, qs := false, false
	for i := 0; i < ml.Count; i++ {
		if ml.Moves[i].Flags() == KingsideCast {
			ks = true
		}
		if ml.Moves[i].Flags() == QueensideCast {
			qs = true
		}
	}

	if !ks || !qs {
		t.Errorf("Castling moves not found: Kingside=%v, Queenside=%v", ks, qs)
	}
}

// TestSEE_Simple verifies Static Exchange Evaluation for simple captures.
func TestSEE_Simple(t *testing.T) {
	b := NewBoard()

	// Case 1: Pawn takes Pawn (protected by another pawn)
	// White: Pe4, Black: Pd5, Pe6.
	// White pawn takes on d5. White wins 100, but then loses 100. SEE = 0.
	b.SetFEN("rnbqkbnr/pp2pppp/4p3/3p4/4P3/8/PPPP1PPP/RNBQKBNR w KQkq - 0 1")
	move := NewMove(E4, D5, CaptureFlag)
	score := b.SEE(move)
	if score != 0 {
		t.Errorf("SEE Case 1: Expected 0, got %d", score)
	}

	// Case 2: Hanging Piece
	// White Knight on f3 can take hanging Black Pawn on e5
	b.SetFEN("rnbqkbnr/pppp1ppp/8/4p3/8/5N2/PPPPPPPP/RNBQKB1R w KQkq - 0 1")
	move = NewMove(F3, E5, CaptureFlag)
	score = b.SEE(move)
	if score < 0 {
		t.Errorf("SEE Case 2: Expected positive score for hanging piece, got %d", score)
	}
}

// TestSEE_Complex verifies SEE for a complex exchange on a single square.
func TestSEE_Complex(t *testing.T) {
	b := NewBoard()
	// Complex exchange on d4
	// White: Nd4, Nf3, Qd1. Black: d5, Nc6, Nf6.
	// 1. Nxd4?
	b.SetFEN("r1bqkb1r/ppp1pppp/2n2n2/3p4/3N4/5N2/PPPPPPPP/R1BQKB1R w KQkq - 0 1")

	// White Knight takes d5.
	// White wins 100 (pawn).
	// Black takes with Knight (White loses 300).
	// White takes with Knight (White wins 300).
	// Black takes with Knight (White loses 300).
	// ... and so on.
	move := NewMove(D4, D5, CaptureFlag)
	score := b.SEE(move)

	// SEE should correctly identify if the exchange is profitable.
	// In this simple check, we just ensure it doesn't return something nonsensical.
	if score < -500 || score > 500 {
		t.Errorf("SEE Complex: Unlikely score returned: %d", score)
	}
}

// TestMoveGen_Promotion verifies that all 4 promotion types are generated.
func TestMoveGen_Promotion(t *testing.T) {
	b := NewBoard()
	// White pawn on a7, black king on h8
	b.SetFEN("7k/P7/8/8/8/8/8/7K w - - 0 1")

	ml := b.GenerateMoves()
	promos := 0
	for i := 0; i < ml.Count; i++ {
		if ml.Moves[i].Flags()&0x8000 != 0 {
			promos++
		}
	}

	// Should be 4 promos (N, B, R, Q)
	if promos != 4 {
		t.Errorf("Expected 4 promotion moves, got %d", promos)
	}
}
