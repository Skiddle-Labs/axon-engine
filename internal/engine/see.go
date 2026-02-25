package engine

import (
	"github.com/Skiddle-Labs/axon-engine/internal/types"
)

var PieceValues = [7]int{0, 100, 300, 300, 500, 900, 20000}

// SEE (Static Exchange Evaluation) determines the material balance of a series
// of captures on a single square. It returns a score in centipawns.
// Positive values mean the move is a winning exchange.
func (b *Board) SEE(m Move) int {
	from := m.From()
	to := m.To()
	flags := m.Flags()

	// Initial attacker and victim
	victimPiece := b.PieceAt(to)
	victimType := victimPiece.Type()

	if flags == EnPassantFlag {
		victimType = types.Pawn
	}

	// gain[d] stores the material gain at depth d
	gain := make([]int, 64)
	gain[0] = PieceValues[victimType]

	occ := b.Occupancy()
	occ.Clear(from)

	attackers := b.AllAttackers(to, occ)
	us := b.SideToMove ^ 1
	d := 1

	for d < 64 {
		atType, atSq := b.getLeastValuableAttacker(attackers, us)
		if atSq == types.NoSquare {
			break
		}

		// Update gain and board state for the next capture
		gain[d] = PieceValues[atType] - gain[d-1]

		// Remove the attacker and find new attackers (including discovered X-rays)
		occ.Clear(atSq)
		attackers = b.AllAttackers(to, occ)

		us ^= 1
		d++
	}

	// Backtrack to find the optimal score (minimaxing the gains)
	for d--; d > 0; d-- {
		gain[d-1] = -maxSEE(-gain[d-1], gain[d])
	}

	return gain[0]
}

// AllAttackers returns a bitboard of all pieces attacking the given square.
func (b *Board) AllAttackers(sq types.Square, occ Bitboard) Bitboard {
	var attackers Bitboard

	// Pawns: Use the "reverse" attack patterns
	attackers |= (b.blackPawnAttacks(sq) & b.Pieces[types.White][types.Pawn])
	attackers |= (b.whitePawnAttacks(sq) & b.Pieces[types.Black][types.Pawn])

	// Knights
	attackers |= (KnightAttacks[sq] & (b.Colors[types.White] | b.Colors[types.Black]) & (b.Pieces[types.White][types.Knight] | b.Pieces[types.Black][types.Knight]))

	// Kings
	attackers |= (KingAttacks[sq] & (b.Colors[types.White] | b.Colors[types.Black]) & (b.Pieces[types.White][types.King] | b.Pieces[types.Black][types.King]))

	// Sliders: Bishops and Queens
	bishopQueens := (b.Pieces[types.White][types.Bishop] | b.Pieces[types.Black][types.Bishop] | b.Pieces[types.White][types.Queen] | b.Pieces[types.Black][types.Queen])
	attackers |= (GetBishopAttacks(sq, occ) & bishopQueens)

	// Sliders: Rooks and Queens
	rookQueens := (b.Pieces[types.White][types.Rook] | b.Pieces[types.Black][types.Rook] | b.Pieces[types.White][types.Queen] | b.Pieces[types.Black][types.Queen])
	attackers |= (GetRookAttacks(sq, occ) & rookQueens)

	return attackers
}

// getLeastValuableAttacker finds the weakest piece of a given color attacking a square.
func (b *Board) getLeastValuableAttacker(attackers Bitboard, color types.Color) (types.PieceType, types.Square) {
	for pt := types.Pawn; pt <= types.King; pt++ {
		subset := attackers & b.Pieces[color][pt]
		if !subset.IsEmpty() {
			return pt, subset.LSB()
		}
	}
	return types.None, types.NoSquare
}

func maxSEE(a, b int) int {
	if a > b {
		return a
	}
	return b
}
