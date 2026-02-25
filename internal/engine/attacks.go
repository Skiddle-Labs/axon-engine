package engine

import (
	"github.com/Skiddle-Labs/axon-engine/internal/types"
)

// KingAttacks stores precomputed bitboards of all possible king moves for each square.
var KingAttacks [64]Bitboard

// KnightAttacks stores precomputed bitboards of all possible knight moves for each square.
var KnightAttacks [64]Bitboard

func init() {
	for sq := 0; sq < 64; sq++ {
		KingAttacks[sq] = maskKingAttacks(types.Square(sq))
		KnightAttacks[sq] = maskKnightAttacks(types.Square(sq))
	}
}

func maskKingAttacks(sq types.Square) Bitboard {
	var attacks Bitboard
	b := Bitboard(1 << sq)

	// Up / Down
	if (b & Rank8) == 0 {
		attacks |= (b << 8)
	}
	if (b & Rank1) == 0 {
		attacks |= (b >> 8)
	}
	// Left / Right
	if (b & FileA) == 0 {
		attacks |= (b >> 1)
	}
	if (b & FileH) == 0 {
		attacks |= (b << 1)
	}

	// Diagonals
	if (b & (Rank8 | FileA)) == 0 {
		attacks |= (b << 7)
	}
	if (b & (Rank8 | FileH)) == 0 {
		attacks |= (b << 9)
	}
	if (b & (Rank1 | FileA)) == 0 {
		attacks |= (b >> 9)
	}
	if (b & (Rank1 | FileH)) == 0 {
		attacks |= (b >> 7)
	}

	return attacks
}

func maskKnightAttacks(sq types.Square) Bitboard {
	var attacks Bitboard
	b := Bitboard(1 << sq)

	// Up 2, Left 1 / Right 1
	if (b & (Rank7 | Rank8 | FileA)) == 0 {
		attacks |= (b << 15)
	}
	if (b & (Rank7 | Rank8 | FileH)) == 0 {
		attacks |= (b << 17)
	}

	// Up 1, Left 2 / Right 2
	if (b & (Rank8 | FileA | FileB)) == 0 {
		attacks |= (b << 6)
	}
	if (b & (Rank8 | FileG | FileH)) == 0 {
		attacks |= (b << 10)
	}

	// Down 2, Left 1 / Right 1
	if (b & (Rank1 | Rank2 | FileA)) == 0 {
		attacks |= (b >> 17)
	}
	if (b & (Rank1 | Rank2 | FileH)) == 0 {
		attacks |= (b >> 15)
	}

	// Down 1, Left 2 / Right 2
	if (b & (Rank1 | FileA | FileB)) == 0 {
		attacks |= (b >> 10)
	}
	if (b & (Rank1 | FileG | FileH)) == 0 {
		attacks |= (b >> 6)
	}

	return attacks
}
