package board

// KingAttacks stores precomputed bitboards of all possible king moves for each square.
var KingAttacks [64]Bitboard

// KnightAttacks stores precomputed bitboards of all possible knight moves for each square.
var KnightAttacks [64]Bitboard

func init() {
	for sq := 0; sq < 64; sq++ {
		KingAttacks[sq] = maskKingAttacks(Square(sq))
		KnightAttacks[sq] = maskKnightAttacks(Square(sq))
	}
}

func maskKingAttacks(sq Square) Bitboard {
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

func maskKnightAttacks(sq Square) Bitboard {
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

// GetBishopAttacks returns the squares attacked by a bishop on the given square,
// considering the current board occupancy.
func GetBishopAttacks(sq Square, occupancy Bitboard) Bitboard {
	var attacks Bitboard

	// Up-Right
	for r, f := sq.Rank()+1, sq.File()+1; r <= 7 && f <= 7; r, f = r+1, f+1 {
		target := NewSquare(f, r)
		attacks.Set(target)
		if occupancy.Test(target) {
			break
		}
	}
	// Up-Left
	for r, f := sq.Rank()+1, sq.File()-1; r <= 7 && f >= 0; r, f = r+1, f-1 {
		target := NewSquare(f, r)
		attacks.Set(target)
		if occupancy.Test(target) {
			break
		}
	}
	// Down-Right
	for r, f := sq.Rank()-1, sq.File()+1; r >= 0 && f <= 7; r, f = r-1, f+1 {
		target := NewSquare(f, r)
		attacks.Set(target)
		if occupancy.Test(target) {
			break
		}
	}
	// Down-Left
	for r, f := sq.Rank()-1, sq.File()-1; r >= 0 && f >= 0; r, f = r-1, f-1 {
		target := NewSquare(f, r)
		attacks.Set(target)
		if occupancy.Test(target) {
			break
		}
	}

	return attacks
}

// GetRookAttacks returns the squares attacked by a rook on the given square,
// considering the current board occupancy.
func GetRookAttacks(sq Square, occupancy Bitboard) Bitboard {
	var attacks Bitboard

	// Up
	for r := sq.Rank() + 1; r <= 7; r++ {
		target := NewSquare(sq.File(), r)
		attacks.Set(target)
		if occupancy.Test(target) {
			break
		}
	}
	// Down
	for r := sq.Rank() - 1; r >= 0; r-- {
		target := NewSquare(sq.File(), r)
		attacks.Set(target)
		if occupancy.Test(target) {
			break
		}
	}
	// Right
	for f := sq.File() + 1; f <= 7; f++ {
		target := NewSquare(f, sq.Rank())
		attacks.Set(target)
		if occupancy.Test(target) {
			break
		}
	}
	// Left
	for f := sq.File() - 1; f >= 0; f-- {
		target := NewSquare(f, sq.Rank())
		attacks.Set(target)
		if occupancy.Test(target) {
			break
		}
	}

	return attacks
}

// GetQueenAttacks returns the squares attacked by a queen on the given square,
// considering the current board occupancy.
func GetQueenAttacks(sq Square, occupancy Bitboard) Bitboard {
	return GetBishopAttacks(sq, occupancy) | GetRookAttacks(sq, occupancy)
}
