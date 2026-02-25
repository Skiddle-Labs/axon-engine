package eval

import (
	"github.com/Skiddle-Labs/axon-engine/internal/engine"
)

// Evaluate returns a score for the current board position using tapered evaluation.
func Evaluate(b *engine.Board) int {
	mgW, egW, _ := calculatePhase(b)

	mgWhite, egWhite := evaluateColor(b, engine.White)
	mgBlack, egBlack := evaluateColor(b, engine.Black)

	mgScore := mgWhite - mgBlack
	egScore := egWhite - egBlack

	score := (mgScore*mgW + egScore*egW) / TotalPhase

	if b.SideToMove == engine.Black {
		return -score
	}
	return score
}

func calculatePhase(b *engine.Board) (int, int, int) {
	phase := TotalPhase
	phase -= (b.Pieces[engine.White][engine.Knight].Count() + b.Pieces[engine.Black][engine.Knight].Count()) * KnightPhase
	phase -= (b.Pieces[engine.White][engine.Bishop].Count() + b.Pieces[engine.Black][engine.Bishop].Count()) * BishopPhase
	phase -= (b.Pieces[engine.White][engine.Rook].Count() + b.Pieces[engine.Black][engine.Rook].Count()) * RookPhase
	phase -= (b.Pieces[engine.White][engine.Queen].Count() + b.Pieces[engine.Black][engine.Queen].Count()) * QueenPhase

	if phase < 0 {
		phase = 0
	}

	egW := phase
	mgW := TotalPhase - phase
	return mgW, egW, phase
}

func evaluateColor(b *engine.Board, c engine.Color) (int, int) {
	mg, eg := 0, 0
	them := c ^ 1
	occ := b.Occupancy()

	// Material and PST
	for pt := engine.Pawn; pt <= engine.King; pt++ {
		pieces := b.Pieces[c][pt]
		for pieces != 0 {
			sq := pieces.PopLSB()
			idx := getPST(sq, c)
			mg += MgPST[pt][idx]
			eg += EgPST[pt][idx]

			switch pt {
			case engine.Pawn:
				mg += PawnMG
				eg += PawnEG
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
		}
	}

	// Pawn Structure
	pawns := b.Pieces[c][engine.Pawn]
	enemyPawns := b.Pieces[them][engine.Pawn]
	pawnCopy := pawns
	for pawnCopy != 0 {
		sq := pawnCopy.PopLSB()
		file := sq.File()
		rank := sq.Rank()

		// Doubled pawns
		if (pawns & (engine.FileA << file)).Count() > 1 {
			mg += PawnDoubledMG
			eg += PawnDoubledEG
		}

		// Isolated pawns
		isIsolated := true
		if file > 0 && (pawns&(engine.FileA<<(file-1))) != 0 {
			isIsolated = false
		}
		if file < 7 && (pawns&(engine.FileA<<(file+1))) != 0 {
			isIsolated = false
		}
		if isIsolated {
			mg += PawnIsolatedMG
			eg += PawnIsolatedEG
		}

		// Connected pawns (protected by another pawn)
		supported := false
		if c == engine.White {
			if rank > 0 {
				if file > 0 && pawns.Test(engine.NewSquare(file-1, rank-1)) {
					supported = true
				}
				if file < 7 && pawns.Test(engine.NewSquare(file+1, rank-1)) {
					supported = true
				}
			}
		} else {
			if rank < 7 {
				if file > 0 && pawns.Test(engine.NewSquare(file-1, rank+1)) {
					supported = true
				}
				if file < 7 && pawns.Test(engine.NewSquare(file+1, rank+1)) {
					supported = true
				}
			}
		}

		// Phalanx pawns (side-by-side)
		phalanx := false
		if file > 0 && pawns.Test(engine.NewSquare(file-1, rank)) {
			phalanx = true
		}
		if file < 7 && pawns.Test(engine.NewSquare(file+1, rank)) {
			phalanx = true
		}

		if supported {
			mg += PawnSupportedMG
			eg += PawnSupportedEG
		} else if phalanx {
			mg += PawnPhalanxMG
			eg += PawnPhalanxEG
		}

		// Backward pawn detection
		if !supported && !phalanx {
			hasAdjacentBehind := false
			if c == engine.White {
				for r := 0; r <= rank; r++ {
					if (file > 0 && pawns.Test(engine.NewSquare(file-1, r))) ||
						(file < 7 && pawns.Test(engine.NewSquare(file+1, r))) {
						hasAdjacentBehind = true
						break
					}
				}
				if !hasAdjacentBehind && rank < 7 {
					if (file > 0 && enemyPawns.Test(engine.NewSquare(file-1, rank+1))) ||
						(file < 7 && enemyPawns.Test(engine.NewSquare(file+1, rank+1))) {
						mg += PawnBackwardMG
						eg += PawnBackwardEG
					}
				}
			} else {
				for r := 7; r >= rank; r-- {
					if (file > 0 && pawns.Test(engine.NewSquare(file-1, r))) ||
						(file < 7 && pawns.Test(engine.NewSquare(file+1, r))) {
						hasAdjacentBehind = true
						break
					}
				}
				if !hasAdjacentBehind && rank > 0 {
					if (file > 0 && enemyPawns.Test(engine.NewSquare(file-1, rank-1))) ||
						(file < 7 && enemyPawns.Test(engine.NewSquare(file+1, rank-1))) {
						mg += PawnBackwardMG
						eg += PawnBackwardEG
					}
				}
			}
		}

		// Passed pawns
		frontMask := engine.Bitboard(0)
		if c == engine.White {
			for r := rank + 1; r <= 7; r++ {
				frontMask.Set(engine.NewSquare(file, r))
				if file > 0 {
					frontMask.Set(engine.NewSquare(file-1, r))
				}
				if file < 7 {
					frontMask.Set(engine.NewSquare(file+1, r))
				}
			}
		} else {
			for r := rank - 1; r >= 0; r-- {
				frontMask.Set(engine.NewSquare(file, r))
				if file > 0 {
					frontMask.Set(engine.NewSquare(file-1, r))
				}
				if file < 7 {
					frontMask.Set(engine.NewSquare(file+1, r))
				}
			}
		}
		if (frontMask & enemyPawns).IsEmpty() {
			bonus := 0
			if c == engine.White {
				bonus = rank * rank
			} else {
				bonus = (7 - rank) * (7 - rank)
			}
			mg += bonus * PawnPassedMG
			eg += bonus * PawnPassedEG
		}
	}

	// Piece specific evaluation
	for pt := engine.Knight; pt <= engine.Queen; pt++ {
		pieces := b.Pieces[c][pt]
		for pieces != 0 {
			sq := pieces.PopLSB()
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

				// File bonuses
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

	if b.Pieces[c][engine.Bishop].Count() >= 2 {
		mg += BishopPairMG
		eg += BishopPairEG
	}

	// King Safety
	mg += evaluateKingSafety(b, c)

	// Threats
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

func evaluateKingSafety(b *engine.Board, c engine.Color) int {
	score := 0
	kingBB := b.Pieces[c][engine.King]
	if kingBB.IsEmpty() {
		return 0
	}
	kingSq := kingBB.LSB()
	pawns := b.Pieces[c][engine.Pawn]

	// 1. Pawn Shield
	rank := kingSq.Rank()
	file := kingSq.File()
	if c == engine.White {
		if rank < 7 {
			for f_idx := file - 1; f_idx <= file+1; f_idx++ {
				if f_idx >= 0 && f_idx <= 7 {
					if pawns.Test(engine.NewSquare(f_idx, rank+1)) {
						score += KingShieldClose
					} else if rank < 6 && pawns.Test(engine.NewSquare(f_idx, rank+2)) {
						score += KingShieldFar
					}
				}
			}
		}
	} else {
		if rank > 0 {
			for f_idx := file - 1; f_idx <= file+1; f_idx++ {
				if f_idx >= 0 && f_idx <= 7 {
					if pawns.Test(engine.NewSquare(f_idx, rank-1)) {
						score += KingShieldClose
					} else if rank > 1 && pawns.Test(engine.NewSquare(f_idx, rank-2)) {
						score += KingShieldFar
					}
				}
			}
		}
	}

	// 2. Attacking Zone
	them := c ^ 1
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
		score -= SafetyTable[penaltyIndex]
	}

	return score
}

// getPST maps a square to its value in the Piece-Square Table.
func getPST(sq engine.Square, c engine.Color) int {
	rank := int(sq) / 8
	file := int(sq) % 8
	if c == engine.White {
		return (7-rank)*8 + file
	}
	return rank*8 + file
}

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
