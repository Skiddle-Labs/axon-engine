package engine

import (
	"math/rand"

	"github.com/Skiddle-Labs/axon-engine/internal/types"
)

// Zobrist random keys
var (
	PieceKeys     [13][64]uint64
	SideKey       uint64
	CastlingKeys  [16]uint64
	EnPassantKeys [8]uint64
)

func init() {
	// We use a fixed seed to ensure the hash is consistent across engine restarts.
	r := rand.New(rand.NewSource(1070372))

	for p := 1; p <= 12; p++ {
		for s := 0; s < 64; s++ {
			PieceKeys[p][s] = r.Uint64()
		}
	}

	SideKey = r.Uint64()

	for i := 0; i < 16; i++ {
		CastlingKeys[i] = r.Uint64()
	}

	for i := 0; i < 8; i++ {
		EnPassantKeys[i] = r.Uint64()
	}
}

// ComputeHash calculates the Zobrist hash of the current board state from scratch.
// In a performance-critical search, the hash is updated incrementally during
// MakeMove/UnmakeMove, but this function is useful for initialization and verification.
func (b *Board) ComputeHash() uint64 {
	var h uint64

	// XOR pieces on squares
	for s := 0; s < 64; s++ {
		sq := types.Square(s)
		p := b.PieceAt(sq)
		if p != types.NoPiece {
			h ^= PieceKeys[p][s]
		}
	}

	// XOR side to move (usually only XORed if it's black's turn)
	if b.SideToMove == types.Black {
		h ^= SideKey
	}

	// XOR castling rights
	h ^= CastlingKeys[b.Castling]

	// XOR en passant file
	if b.EnPassant != types.NoSquare {
		h ^= EnPassantKeys[b.EnPassant.File()]
	}

	return h
}

// ComputePawnHash calculates the Zobrist hash of the current pawn structure from scratch.
func (b *Board) ComputePawnHash() uint64 {
	var h uint64

	whitePawns := b.Pieces[types.White][types.Pawn]
	for whitePawns != 0 {
		sq := whitePawns.PopLSB()
		h ^= PieceKeys[types.WhitePawn][sq]
	}

	blackPawns := b.Pieces[types.Black][types.Pawn]
	for blackPawns != 0 {
		sq := blackPawns.PopLSB()
		h ^= PieceKeys[types.BlackPawn][sq]
	}

	return h
}
