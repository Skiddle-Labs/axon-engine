package engine

import (
	"testing"

	"github.com/Skiddle-Labs/axon-engine/internal/types"
)

// TestFENParsing verifies that the board correctly parses FEN strings.
func TestFENParsing(t *testing.T) {
	b := NewBoard()
	fen := "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1"
	err := b.SetFEN(fen)
	if err != nil {
		t.Fatalf("Failed to parse FEN: %v", err)
	}

	// Check piece at E4
	p := b.PieceAt(types.E4)
	if p != types.WhitePawn {
		t.Errorf("Expected WhitePawn at E4, got %v", p)
	}

	// Check side to move
	if b.SideToMove != types.Black {
		t.Errorf("Expected side to move to be Black, got %v", b.SideToMove)
	}

	// Check En Passant square
	if b.EnPassant != types.E3 {
		t.Errorf("Expected En Passant square to be E3, got %v", b.EnPassant)
	}

	// Check Castling rights
	expectedCastling := WhiteKingside | WhiteQueenside | BlackKingside | BlackQueenside
	if b.Castling != expectedCastling {
		t.Errorf("Expected castling rights %v, got %v", expectedCastling, b.Castling)
	}
}

// TestMoveGenerationStartingPosition verifies that there are 20 legal moves in the starting position.
func TestMoveGenerationStartingPosition(t *testing.T) {
	b := NewBoard()
	b.SetFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	ml := b.GenerateMoves()

	// Filter for legal moves (MakeMove returns false if king is in check)
	legalCount := 0
	for i := 0; i < ml.Count; i++ {
		if b.MakeMove(ml.Moves[i]) {
			b.UnmakeMove(ml.Moves[i])
			legalCount++
		}
	}

	if legalCount != 20 {
		t.Errorf("Expected 20 legal moves in starting position, got %d", legalCount)
	}
}

// TestMakeUnmakeConsistency ensures that making and unmaking moves preserves board state and hashes.
func TestMakeUnmakeConsistency(t *testing.T) {
	b := NewBoard()
	initialFEN := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	b.SetFEN(initialFEN)

	initialHash := b.Hash

	// E2E4
	m := NewMove(types.E2, types.E4, DoublePawnPush)
	if !b.MakeMove(m) {
		t.Fatal("E2E4 move was rejected as illegal")
	}

	if b.Hash == initialHash {
		t.Error("Hash should change after making a move")
	}

	b.UnmakeMove(m)

	if b.Hash != initialHash {
		t.Errorf("Hash mismatch after unmake. Expected %v, got %v", initialHash, b.Hash)
	}

	// Check if recomputed hash matches incremental hash
	if b.Hash != b.ComputeHash() {
		t.Error("Incremental hash does not match recomputed hash after unmake")
	}
}

// TestIsSquareAttacked verifies the attack detection logic.
func TestIsSquareAttacked(t *testing.T) {
	b := NewBoard()
	// Scholar's Mate position
	b.SetFEN("r1bqkbnr/pppp1ppp/2n5/4p3/2B1P3/5Q2/PPPP1PPP/RNB1K1NR w KQkq - 0 1")

	// F7 is attacked by White Queen and Bishop
	if !b.IsSquareAttacked(types.F7, types.White) {
		t.Error("Square F7 should be attacked by White")
	}

	// E1 (White King) is NOT attacked by Black
	if b.IsSquareAttacked(types.E1, types.Black) {
		t.Error("Square E1 should not be attacked by Black in this position")
	}

	// D1 is attacked by White Queen (self-attack check - should be handled by caller)
	// But let's check if the logic detects the Queen on F3 attacking D1
	if !b.IsSquareAttacked(types.D1, types.White) {
		t.Error("Square D1 should be 'attacked' by the White Queen on F3")
	}
}

// TestKnightAttacks verifies the precomputed knight move table.
func TestKnightAttacks(t *testing.T) {
	// Knight on D4 should attack 8 squares
	attacks := KnightAttacks[types.D4]
	count := attacks.Count()
	if count != 8 {
		t.Errorf("Knight on D4 should attack 8 squares, got %d", count)
	}

	// Knight on A1 should attack 2 squares (B3, C2)
	attacks = KnightAttacks[types.A1]
	count = attacks.Count()
	if count != 2 {
		t.Errorf("Knight on A1 should attack 2 squares, got %d", count)
	}
}

// TestSlidingAttacks verifies the ray-casting attack generation.
func TestSlidingAttacks(t *testing.T) {
	b := NewBoard()
	// Clear board, place Rook on D4
	b.Clear()
	b.Pieces[types.White][types.Rook].Set(types.D4)
	b.Colors[types.White].Set(types.D4)

	occ := b.Occupancy()
	attacks := GetRookAttacks(types.D4, occ)

	// A rook on an empty board should attack 14 squares (7 on rank, 7 on file)
	if attacks.Count() != 14 {
		t.Errorf("Rook on D4 empty board should attack 14 squares, got %d", attacks.Count())
	}

	// Block the rook with a piece on D6
	b.Pieces[types.Black][types.Pawn].Set(types.D6)
	b.Colors[types.Black].Set(types.D6)
	occ = b.Occupancy()
	attacks = GetRookAttacks(types.D4, occ)

	// Should now attack D5, D6, but not D7 or D8
	if attacks.Test(types.D7) || attacks.Test(types.D8) {
		t.Error("Rook should be blocked by piece on D6")
	}
	if !attacks.Test(types.D6) {
		t.Error("Rook should attack the blocking piece square D6")
	}
}
