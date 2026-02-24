package engine

import "fmt"

// Move represents a chess move.
// It is encoded as a 16-bit integer:
// Bits 0-5:   Source square (0-63)
// Bits 6-11:  Destination square (0-63)
// Bits 12-15: Move flags (Promotions, Castling, En Passant, etc.)
type Move uint16

const (
	NoMove Move = 0

	// Move Flags
	QuietFlag      uint16 = 0x0000
	DoublePawnPush uint16 = 0x1000
	KingsideCast   uint16 = 0x2000
	QueensideCast  uint16 = 0x3000
	CaptureFlag    uint16 = 0x4000
	EnPassantFlag  uint16 = 0x5000
	PromoKnight    uint16 = 0x8000
	PromoBishop    uint16 = 0x9000
	PromoRook      uint16 = 0xA000
	PromoQueen     uint16 = 0xB000
)

func NewMove(from, to Square, flags uint16) Move {
	return Move(uint16(from) | (uint16(to) << 6) | flags)
}

func (m Move) From() Square  { return Square(m & 0x3F) }
func (m Move) To() Square    { return Square((m >> 6) & 0x3F) }
func (m Move) Flags() uint16 { return uint16(m & 0xF000) }

func (m Move) String() string {
	if m == NoMove {
		return "none"
	}
	s := fmt.Sprintf("%s%s", m.From().String(), m.To().String())
	switch m.Flags() & 0xB000 {
	case PromoKnight:
		s += "n"
	case PromoBishop:
		s += "b"
	case PromoRook:
		s += "r"
	case PromoQueen:
		s += "q"
	}
	return s
}

// MoveList stores a collection of moves.
type MoveList struct {
	Moves [256]Move
	Count int
}

// AddMove adds a move to the list.
func (ml *MoveList) AddMove(m Move) {
	if ml.Count < 256 {
		ml.Moves[ml.Count] = m
		ml.Count++
	}
}
