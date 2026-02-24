package board

var seeValues = [7]int{0, 100, 300, 300, 500, 900, 20000}

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
		victimType = Pawn
	}

	// gain[d] stores the material gain at depth d
	gain := make([]int, 64)
	gain[0] = seeValues[victimType]

	occ := b.Occupancy()
	occ.Clear(from)

	attackers := b.allAttackers(to, occ)
	us := b.SideToMove ^ 1
	d := 1

	for d < 64 {
		atType, atSq := b.getLeastValuableAttacker(attackers, us)
		if atSq == NoSquare {
			break
		}

		// Update gain and board state for the next capture
		gain[d] = seeValues[atType] - gain[d-1]

		// Remove the attacker and find new attackers (including discovered X-rays)
		occ.Clear(atSq)
		attackers = b.allAttackers(to, occ)

		us ^= 1
		d++
	}

	// Backtrack to find the optimal score (minimaxing the gains)
	for d--; d > 0; d-- {
		gain[d-1] = -maxSEE(-gain[d-1], gain[d])
	}

	return gain[0]
}

// allAttackers returns a bitboard of all pieces attacking the given square.
func (b *Board) allAttackers(sq Square, occ Bitboard) Bitboard {
	var attackers Bitboard

	// Pawns: Use the "reverse" attack patterns
	attackers |= (b.blackPawnAttacks(sq) & b.Pieces[White][Pawn])
	attackers |= (b.whitePawnAttacks(sq) & b.Pieces[Black][Pawn])

	// Knights
	attackers |= (KnightAttacks[sq] & (b.Colors[White] | b.Colors[Black]) & (b.Pieces[White][Knight] | b.Pieces[Black][Knight]))

	// Kings
	attackers |= (KingAttacks[sq] & (b.Colors[White] | b.Colors[Black]) & (b.Pieces[White][King] | b.Pieces[Black][King]))

	// Sliders: Bishops and Queens
	bishopQueens := (b.Pieces[White][Bishop] | b.Pieces[Black][Bishop] | b.Pieces[White][Queen] | b.Pieces[Black][Queen])
	attackers |= (GetBishopAttacks(sq, occ) & bishopQueens)

	// Sliders: Rooks and Queens
	rookQueens := (b.Pieces[White][Rook] | b.Pieces[Black][Rook] | b.Pieces[White][Queen] | b.Pieces[Black][Queen])
	attackers |= (GetRookAttacks(sq, occ) & rookQueens)

	return attackers
}

// getLeastValuableAttacker finds the weakest piece of a given color attacking a square.
func (b *Board) getLeastValuableAttacker(attackers Bitboard, color Color) (PieceType, Square) {
	for pt := Pawn; pt <= King; pt++ {
		subset := attackers & b.Pieces[color][pt]
		if !subset.IsEmpty() {
			return pt, subset.LSB()
		}
	}
	return None, NoSquare
}

func maxSEE(a, b int) int {
	if a > b {
		return a
	}
	return b
}
