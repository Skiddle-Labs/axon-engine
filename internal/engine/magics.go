package engine

import (
	"math/rand"

	"github.com/Skiddle-Labs/axon-engine/internal/types"
)

// magic holds the data needed for magic bitboard lookups for a single square.
type magic struct {
	mask  Bitboard
	magic uint64
	shift uint8
}

var rookMagics [64]magic
var bishopMagics [64]magic
var rookTable [64][]Bitboard
var bishopTable [64][]Bitboard

// getIndex returns the index for the given occupancy.
func (m *magic) getIndex(occ Bitboard) int {
	return int(((uint64(occ&m.mask) * m.magic) >> m.shift))
}

// GetRookAttacks returns the squares attacked by a rook on the given square,
// considering the current board occupancy, using magic bitboards.
func GetRookAttacks(sq types.Square, occupancy Bitboard) Bitboard {
	return rookTable[sq][rookMagics[sq].getIndex(occupancy)]
}

// GetBishopAttacks returns the squares attacked by a bishop on the given square,
// considering the current board occupancy, using magic bitboards.
func GetBishopAttacks(sq types.Square, occupancy Bitboard) Bitboard {
	return bishopTable[sq][bishopMagics[sq].getIndex(occupancy)]
}

// GetQueenAttacks returns the squares attacked by a queen on the given square,
// considering the current board occupancy, using magic bitboards.
func GetQueenAttacks(sq types.Square, occupancy Bitboard) Bitboard {
	return GetRookAttacks(sq, occupancy) | GetBishopAttacks(sq, occupancy)
}

// maskRookOccupancy returns the relevant occupancy mask for a rook on sq.
func maskRookOccupancy(sq types.Square) Bitboard {
	var occupancy Bitboard
	rank, file := sq.Rank(), sq.File()
	for r := rank + 1; r < 7; r++ {
		occupancy.Set(types.NewSquare(file, r))
	}
	for r := rank - 1; r > 0; r-- {
		occupancy.Set(types.NewSquare(file, r))
	}
	for f := file + 1; f < 7; f++ {
		occupancy.Set(types.NewSquare(f, rank))
	}
	for f := file - 1; f > 0; f-- {
		occupancy.Set(types.NewSquare(f, rank))
	}
	return occupancy
}

// maskBishopOccupancy returns the relevant occupancy mask for a bishop on sq.
func maskBishopOccupancy(sq types.Square) Bitboard {
	var occupancy Bitboard
	rank, file := sq.Rank(), sq.File()
	for r, f := rank+1, file+1; r < 7 && f < 7; r, f = r+1, f+1 {
		occupancy.Set(types.NewSquare(f, r))
	}
	for r, f := rank+1, file-1; r < 7 && f > 0; r, f = r+1, f-1 {
		occupancy.Set(types.NewSquare(f, r))
	}
	for r, f := rank-1, file+1; r > 0 && f < 7; r, f = r-1, f+1 {
		occupancy.Set(types.NewSquare(f, r))
	}
	for r, f := rank-1, file-1; r > 0 && f > 0; r, f = r-1, f-1 {
		occupancy.Set(types.NewSquare(f, r))
	}
	return occupancy
}

// getRookAttacksSlow calculates rook attacks by stepping through squares.
func getRookAttacksSlow(sq types.Square, occupancy Bitboard) Bitboard {
	var attacks Bitboard
	rank, file := sq.Rank(), sq.File()
	for r := rank + 1; r <= 7; r++ {
		target := types.NewSquare(file, r)
		attacks.Set(target)
		if occupancy.Test(target) {
			break
		}
	}
	for r := rank - 1; r >= 0; r-- {
		target := types.NewSquare(file, r)
		attacks.Set(target)
		if occupancy.Test(target) {
			break
		}
	}
	for f := file + 1; f <= 7; f++ {
		target := types.NewSquare(f, rank)
		attacks.Set(target)
		if occupancy.Test(target) {
			break
		}
	}
	for f := file - 1; f >= 0; f-- {
		target := types.NewSquare(f, rank)
		attacks.Set(target)
		if occupancy.Test(target) {
			break
		}
	}
	return attacks
}

// getBishopAttacksSlow calculates bishop attacks by stepping through squares.
func getBishopAttacksSlow(sq types.Square, occupancy Bitboard) Bitboard {
	var attacks Bitboard
	rank, file := sq.Rank(), sq.File()
	for r, f := rank+1, file+1; r <= 7 && f <= 7; r, f = r+1, f+1 {
		target := types.NewSquare(f, r)
		attacks.Set(target)
		if occupancy.Test(target) {
			break
		}
	}
	for r, f := rank+1, file-1; r <= 7 && f >= 0; r, f = r+1, f-1 {
		target := types.NewSquare(f, r)
		attacks.Set(target)
		if occupancy.Test(target) {
			break
		}
	}
	for r, f := rank-1, file+1; r >= 0 && f <= 7; r, f = r-1, f+1 {
		target := types.NewSquare(f, r)
		attacks.Set(target)
		if occupancy.Test(target) {
			break
		}
	}
	for r, f := rank-1, file-1; r >= 0 && f >= 0; r, f = r-1, f-1 {
		target := types.NewSquare(f, r)
		attacks.Set(target)
		if occupancy.Test(target) {
			break
		}
	}
	return attacks
}

// generateOccupancy generates a bitboard from an index and a mask.
func generateOccupancy(index int, bits int, mask Bitboard) Bitboard {
	var occupancy Bitboard
	for i := 0; i < bits; i++ {
		sq := mask.PopLSB()
		if (index & (1 << i)) != 0 {
			occupancy.Set(sq)
		}
	}
	return occupancy
}

// findMagic finds a magic number for a square by trial and error.
func findMagic(sq types.Square, mask Bitboard, bits uint8, isRook bool, rng *rand.Rand) (magic, []Bitboard) {
	numOcc := 1 << bits
	occupancies := make([]Bitboard, numOcc)
	attacks := make([]Bitboard, numOcc)
	for i := 0; i < numOcc; i++ {
		m := mask // Copy because PopLSB modifies it
		occupancies[i] = generateOccupancy(i, int(bits), m)
		if isRook {
			attacks[i] = getRookAttacksSlow(sq, occupancies[i])
		} else {
			attacks[i] = getBishopAttacksSlow(sq, occupancies[i])
		}
	}

	table := make([]Bitboard, numOcc)
	used := make([]uint32, numOcc)
	var epoch uint32

	for {
		epoch++
		magicNum := rng.Uint64() & rng.Uint64() & rng.Uint64()
		m := magic{mask: mask, magic: magicNum, shift: 64 - bits}

		fail := false
		for i := 0; i < numOcc; i++ {
			idx := m.getIndex(occupancies[i])
			if used[idx] != epoch {
				used[idx] = epoch
				table[idx] = attacks[i]
			} else if table[idx] != attacks[i] {
				fail = true
				break
			}
		}
		if !fail {
			resTable := make([]Bitboard, numOcc)
			copy(resTable, table)
			return m, resTable
		}
	}
}

var rookRelevantBits = [64]uint8{
	12, 11, 11, 11, 11, 11, 11, 12,
	11, 10, 10, 10, 10, 10, 10, 11,
	11, 10, 10, 10, 10, 10, 10, 11,
	11, 10, 10, 10, 10, 10, 10, 11,
	11, 10, 10, 10, 10, 10, 10, 11,
	11, 10, 10, 10, 10, 10, 10, 11,
	11, 10, 10, 10, 10, 10, 10, 11,
	12, 11, 11, 11, 11, 11, 11, 12,
}

var bishopRelevantBits = [64]uint8{
	6, 5, 5, 5, 5, 5, 5, 6,
	5, 5, 5, 5, 5, 5, 5, 5,
	5, 5, 7, 7, 7, 7, 5, 5,
	5, 5, 7, 9, 9, 7, 5, 5,
	5, 5, 7, 9, 9, 7, 5, 5,
	5, 5, 7, 7, 7, 7, 5, 5,
	5, 5, 5, 5, 5, 5, 5, 5,
	6, 5, 5, 5, 5, 5, 5, 6,
}

func init() {
	rng := rand.New(rand.NewSource(42))
	for sq := 0; sq < 64; sq++ {
		s := types.Square(sq)
		rookMagics[sq], rookTable[sq] = findMagic(s, maskRookOccupancy(s), rookRelevantBits[sq], true, rng)
		bishopMagics[sq], bishopTable[sq] = findMagic(s, maskBishopOccupancy(s), bishopRelevantBits[sq], false, rng)
	}
}
