package eval

import (
	"github.com/Skiddle-Labs/axon-engine/internal/engine"
)

// evaluatePieces calculates the material, mobility, and positional bonuses for all non-pawn pieces.
func evaluatePieces(b *engine.Board, c engine.Color) (int, int) {
	mg, eg := 0, 0
	them := c ^ 1
	occ := b.Occupancy()
	pawns := b.Pieces[c][engine.Pawn]
	enemyPawns := b.Pieces[them][engine.Pawn]

	for pt := engine.Knight; pt <= engine.Queen; pt++ {
		pieces := b.Pieces[c][pt]
		for pieces != 0 {
			sq := pieces.PopLSB()

			// 1. Material Values
			switch pt {
			case engine.Knight:
				mg += KnightMG
				eg += KnightEG
			case engine.Bishop:
				mg += BishopMG
				eg += BishopEG
			case engine.Rook:
				mg += RookMG
				eg += RookEG
			case engine.Queen:
				mg += QueenMG
				eg += QueenEG
			}

			// 2. Mobility and Positional Features
			var attacks engine.Bitboard
			switch pt {
			case engine.Knight:
				attacks = engine.KnightAttacks[sq]
				mg += attacks.Count() * KnightMobilityMG
				eg += attacks.Count() * KnightMobilityEG
				if isOutpost(b, c, sq) {
					mg += KnightOutpostMG
					eg += KnightOutpostEG
				}
			case engine.Bishop:
				attacks = engine.GetBishopAttacks(sq, occ)
				mg += attacks.Count() * BishopMobilityMG
				eg += attacks.Count() * BishopMobilityEG
				if isOutpost(b, c, sq) {
					mg += BishopOutpostMG
					eg += BishopOutpostEG
				}
			case engine.Rook:
				attacks = engine.GetRookAttacks(sq, occ)
				mg += attacks.Count() * RookMobilityMG
				eg += attacks.Count() * RookMobilityEG

				// File bonuses (Open/Half-Open)
				file := sq.File()
				fileBB := engine.FileA << file
				usPawnsOnFile := (pawns & fileBB) != 0
				themPawnsOnFile := (enemyPawns & fileBB) != 0

				if !usPawnsOnFile {
					if !themPawnsOnFile {
						mg += RookOpenFileMG
						eg += RookOpenFileEG
					} else {
						mg += RookHalfOpenFileMG
						eg += RookHalfOpenFileEG
					}
				}
			case engine.Queen:
				attacks = engine.GetQueenAttacks(sq, occ)
				mg += attacks.Count() * QueenMobilityMG
				eg += attacks.Count() * QueenMobilityEG
			}
		}
	}

	// 3. Special Bonuses
	if b.Pieces[c][engine.Bishop].Count() >= 2 {
		mg += BishopPairMG
		eg += BishopPairEG
	}

	return mg, eg
}

// evaluateThreats detects hanging pieces or pieces attacked by weaker ones.
func evaluateThreats(b *engine.Board, c engine.Color) (int, int) {
	mg, eg := 0, 0
	them := c ^ 1
	occ := b.Occupancy()
	enemyOcc := b.Colors[them]
	usOcc := b.Colors[c]

	for pt := engine.Pawn; pt <= engine.Queen; pt++ {
		subset := b.Pieces[c][pt]
		for subset != 0 {
			sq := subset.PopLSB()
			attackers := b.AllAttackers(sq, occ)
			enemyAttackers := attackers & enemyOcc

			if !enemyAttackers.IsEmpty() {
				defenders := attackers & usOcc
				if defenders.IsEmpty() {
					// Hanging piece penalty
					mg -= engine.PieceValues[pt] / HangingDivisorMG
					eg -= engine.PieceValues[pt] / HangingDivisorEG
				} else {
					// Attacked by weaker piece
					for ept := engine.Pawn; ept < pt; ept++ {
						if !(enemyAttackers & b.Pieces[them][ept]).IsEmpty() {
							mg += WeakAttackerMG
							eg += WeakAttackerEG
							break
						}
					}
				}
			}
		}
	}

	return mg, eg
}

// isOutpost determines if a piece is on a square supported by a pawn and cannot be chased by enemy pawns.
func isOutpost(b *engine.Board, c engine.Color, sq engine.Square) bool {
	rank := sq.Rank()
	file := sq.File()

	// Only ranks 3-6 (indices 2-5) for outposts
	if rank < 2 || rank > 5 {
		return false
	}

	pawns := b.Pieces[c][engine.Pawn]
	enemyPawns := b.Pieces[c^1][engine.Pawn]

	// 1. Supported by a pawn
	supported := false
	if c == engine.White {
		if file > 0 && pawns.Test(engine.NewSquare(file-1, rank-1)) {
			supported = true
		}
		if file < 7 && pawns.Test(engine.NewSquare(file+1, rank-1)) {
			supported = true
		}
	} else {
		if file > 0 && pawns.Test(engine.NewSquare(file-1, rank+1)) {
			supported = true
		}
		if file < 7 && pawns.Test(engine.NewSquare(file+1, rank+1)) {
			supported = true
		}
	}

	if !supported {
		return false
	}

	// 2. Cannot be attacked by an enemy pawn
	if c == engine.White {
		for r := rank + 1; r <= 7; r++ {
			if file > 0 && enemyPawns.Test(engine.NewSquare(file-1, r)) {
				return false
			}
			if file < 7 && enemyPawns.Test(engine.NewSquare(file+1, r)) {
				return false
			}
		}
	} else {
		for r := rank - 1; r >= 0; r-- {
			if file > 0 && enemyPawns.Test(engine.NewSquare(file-1, r)) {
				return false
			}
			if file < 7 && enemyPawns.Test(engine.NewSquare(file+1, r)) {
				return false
			}
		}
	}

	return true
}
