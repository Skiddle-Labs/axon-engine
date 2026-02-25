package engine

import (
	"math/bits"

	"github.com/Skiddle-Labs/axon-engine/internal/types"
)

// Bitboard represents a 64-bit unsigned integer where each bit
// corresponds to a square on the chess board.
// Square A1 is bit 0, B1 is bit 1, ..., H8 is bit 63.
type Bitboard uint64

// Set sets the bit at the given square.
func (b *Bitboard) Set(s types.Square) {
	*b |= (1 << s)
}

// Clear clears the bit at the given square.
func (b *Bitboard) Clear(s types.Square) {
	*b &= ^(1 << s)
}

// Test returns true if the bit at the given square is set.
func (b Bitboard) Test(s types.Square) bool {
	return (b & (1 << s)) != 0
}

// Count returns the number of set bits (population count).
func (b Bitboard) Count() int {
	return bits.OnesCount64(uint64(b))
}

// IsEmpty returns true if no bits are set.
func (b Bitboard) IsEmpty() bool {
	return b == 0
}

// PopLSB finds and clears the least significant bit that is set and returns its square.
// It assumes the bitboard is not empty.
func (b *Bitboard) PopLSB() types.Square {
	s := b.LSB()
	*b &= *b - 1
	return s
}

// LSB returns the square of the least significant bit that is set.
func (b Bitboard) LSB() types.Square {
	return types.Square(bits.TrailingZeros64(uint64(b)))
}

// MSB returns the square of the most significant bit that is set.
func (b Bitboard) MSB() types.Square {
	return types.Square(63 - bits.LeadingZeros64(uint64(b)))
}

// Bitboard constants for common sets of squares.
const (
	FileA Bitboard = 0x0101010101010101 << iota
	FileB
	FileC
	FileD
	FileE
	FileF
	FileG
	FileH
)

const (
	Rank1 Bitboard = 0xFF << (8 * iota)
	Rank2
	Rank3
	Rank4
	Rank5
	Rank6
	Rank7
	Rank8
)
