package eval

import (
	"github.com/Skiddle-Labs/axon-engine/internal/engine"
)

// evaluateKingSafety calculates the safety of the king, including pawn shields,
// enemy attacking zones, and pawn storms.
func evaluateKingSafety(b *engine.Board, c engine.Color) (int, int) {
	mg, eg := 0, 0
	kingBB := b.Pieces[c][engine.King]
	if kingBB.IsEmpty() {
		return 0, 0
	}
	kingSq := kingBB.LSB()
	pawns := b.Pieces[c][engine.Pawn]
	them := c ^ 1
	enemyPawns := b.Pieces[them][engine.Pawn]

	// 1. Pawn Shield
	// Rewards pawns directly in front of the king.
	rank := kingSq.Rank()
	file := kingSq.File()
	if c == engine.White {
		if rank < 7 {
			for fIdx := file - 1; fIdx <= file+1; fIdx++ {
				if fIdx >= 0 && fIdx <= 7 {
					if pawns.Test(engine.NewSquare(fIdx, rank+1)) {
						mg += KingShieldClose
					} else if rank < 6 && pawns.Test(engine.NewSquare(fIdx, rank+2)) {
						mg += KingShieldFar
					}
				}
			}
		}
	} else {
		if rank > 0 {
			for fIdx := file - 1; fIdx <= file+1; fIdx++ {
				if fIdx >= 0 && fIdx <= 7 {
					if pawns.Test(engine.NewSquare(fIdx, rank-1)) {
						mg += KingShieldClose
					} else if rank > 1 && pawns.Test(engine.NewSquare(fIdx, rank-2)) {
						mg += KingShieldFar
					}
				}
			}
		}
	}

	// 2. Attacking Zone
	// Penalizes the presence of enemy pieces attacking squares around the king.
	occ := b.Occupancy()
	zone := engine.KingAttacks[kingSq] | (engine.Bitboard(1) << kingSq)

	attackerCount, attackerWeight := 0, 0

	for pt := engine.Knight; pt <= engine.Queen; pt++ {
		pieces := b.Pieces[them][pt]
		for pieces != 0 {
			sq := pieces.PopLSB()
			var attacks engine.Bitboard
			switch pt {
			case engine.Knight:
				attacks = engine.KnightAttacks[sq]
			case engine.Bishop:
				attacks = engine.GetBishopAttacks(sq, occ)
			case engine.Rook:
				attacks = engine.GetRookAttacks(sq, occ)
			case engine.Queen:
				attacks = engine.GetQueenAttacks(sq, occ)
			}
			if !(attacks & zone).IsEmpty() {
				attackerCount++
				attackerWeight += KingAttackerWeight[pt]
			}
		}
	}

	if attackerCount > 0 {
		penaltyIndex := attackerWeight
		if penaltyIndex >= 100 {
			penaltyIndex = 99
		}
		mg -= SafetyTable[penaltyIndex]
	}

	// 3. Pawn Storm
	// Penalize enemy pawns advancing toward our king's position.
	for f := file - 1; f <= file+1; f++ {
		if f < 0 || f > 7 {
			continue
		}
		pawnsOnFile := enemyPawns & (engine.FileA << f)
		if pawnsOnFile != 0 {
			var pSq engine.Square
			var dist int
			if c == engine.White {
				pSq = pawnsOnFile.MSB() // Highest rank pawn
				dist = 7 - pSq.Rank()
			} else {
				pSq = pawnsOnFile.LSB() // Lowest rank pawn
				dist = pSq.Rank()
			}

			if dist < 4 {
				mg += (4 - dist) * PawnStormMG
				eg += (4 - dist) * PawnStormEG
			}
		}
	}

	return mg, eg
}
