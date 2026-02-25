package eval

import (
	"github.com/Skiddle-Labs/axon-engine/internal/engine"
)

// Material values for Midgame (MG) and Endgame (EG)
var (
	PawnMG, PawnEG     = 85, 84
	KnightMG, KnightEG = 349, 293
	BishopMG, BishopEG = 358, 305
	RookMG, RookEG     = 489, 524
	QueenMG, QueenEG   = 1037, 948
)

// Pawn Structure Weights
var (
	PawnDoubledMG, PawnDoubledEG     = 2, -7
	PawnIsolatedMG, PawnIsolatedEG   = -14, -8
	PawnSupportedMG, PawnSupportedEG = 22, 8
	PawnPhalanxMG, PawnPhalanxEG     = 5, 5
	PawnBackwardMG, PawnBackwardEG   = -14, -8
	PawnPassedMG, PawnPassedEG       = 0, 5 // Multipliers for rank*rank
)

// Mobility Weights
var (
	KnightMobilityMG, KnightMobilityEG = 6, 7
	BishopMobilityMG, BishopMobilityEG = 8, 6
	RookMobilityMG, RookMobilityEG     = 6, 9
	QueenMobilityMG, QueenMobilityEG   = 2, 13
)

// Other Positional Weights
var (
	BishopPairMG, BishopPairEG         = 24, 60
	WeakAttackerMG, WeakAttackerEG     = -37, -28
	HangingDivisorMG, HangingDivisorEG = 16, 14 // Penalty = PieceValue / Divisor
)

// King Safety Tables and Weights
var KingAttackerWeight = [7]int{
	0,  // None
	0,  // Pawn
	8,  // Knight
	8,  // Bishop
	11, // Rook
	10, // Queen
	0,  // King
}

var (
	KingShieldClose = 18
	KingShieldFar   = 10
)

var SafetyTable = [100]int{
	1, 0, 1, 3, 4, 5, 6, 8, 7, 16,
	13, 19, 30, 34, 39, 45, 42, 54, 65, 63,
	73, 85, 81, 93, 105, 110, 123, 128, 131, 149,
	155, 164, 179, 187, 204, 210, 222, 233, 249, 260,
	276, 287, 302, 324, 332, 356, 373, 386, 401, 424,
	442, 462, 482, 503, 524, 546, 568, 591, 614, 638,
	662, 687, 712, 738, 764, 791, 818, 846, 874, 903,
	932, 962, 992, 1023, 1054, 1086, 1118, 1151, 1184, 1218,
	1252, 1287, 1322, 1358, 1394, 1431, 1468, 1506, 1544, 1583,
	1622, 1662, 1702, 1743, 1784, 1826, 1867, 1911, 1954, 1998,
}

// Phase values for interpolation
var (
	KnightPhase = 1
	BishopPhase = 1
	RookPhase   = 2
	QueenPhase  = 4
	TotalPhase  = 24
)

// Piece-Square Tables (PST)
var MgPST = [7][64]int{
	engine.Pawn: {
		0, 0, 0, -1, 0, 0, 0, 0,
		62, 62, 54, 57, 56, 62, 47, 49,
		22, 22, 32, 22, 41, 32, 22, 21,
		7, 17, 9, 27, 26, 14, 15, -4,
		-6, -3, 12, 22, 22, 1, -4, -12,
		-6, -13, 2, -11, -5, -6, 7, -5,
		-2, -1, -2, -14, -10, 22, 18, -3,
		2, 0, 0, 1, 0, 0, 0, 0,
	},
	engine.Knight: {
		-62, -44, -23, -27, -18, -38, -35, -62,
		-52, -31, 12, 4, 0, 7, -8, -42,
		-37, 11, 10, 18, 27, 22, 12, -18,
		-18, 9, 3, 32, 16, 27, 7, -18,
		-18, 8, 8, 8, 19, 9, 11, -18,
		-34, -7, -2, 8, 20, 8, 17, -26,
		-28, -31, -12, 3, 3, 12, -11, -28,
		-55, -28, -39, -23, -18, -18, -28, -41,
	},
	engine.Bishop: {
		-18, -5, -22, -20, -11, -11, 0, -9,
		-22, -12, -12, -12, -1, 11, -4, -22,
		-10, 6, 8, 2, 6, 17, 12, 2,
		-5, -7, -3, 17, 14, 9, -3, -2,
		-6, -4, -2, 10, 10, -2, -4, 1,
		-8, 7, 6, -2, -1, 14, 8, -1,
		-1, 15, 10, -2, 1, 8, 17, -9,
		-27, -1, -5, -7, -4, -13, -16, -28,
	},
	engine.Rook: {
		12, 11, 7, 12, 12, 11, 9, 10,
		15, 18, 22, 22, 20, 22, 14, 17,
		6, 12, 10, 10, 7, 11, 12, 5,
		-1, -4, 10, 10, 9, 10, -3, 1,
		-17, -11, 2, 0, 6, -11, 4, -9,
		-17, -8, -4, -5, 2, -5, -7, -16,
		-17, 0, -4, 1, 3, 8, -8, -17,
		-6, -1, 12, 17, 17, 12, -12, -2,
	},
	engine.Queen: {
		-16, 2, 2, 7, 7, 2, 2, -8,
		-21, -12, -6, 11, 7, 12, 12, 2,
		-12, -12, -7, 1, 17, 17, 12, 2,
		-17, -12, -7, -7, -2, 12, 11, 5,
		-12, -12, -7, -7, -7, -7, 2, 2,
		-20, -7, -7, -7, -7, -7, 4, 2,
		-22, -12, 5, 1, 9, 3, -12, 0,
		-9, -14, -5, 7, -5, -22, -19, -31,
	},
	engine.King: {
		-38, -29, -30, -45, -52, -30, -28, -19,
		-19, -28, -29, -39, -42, -28, -28, -24,
		-18, -28, -28, -51, -48, -28, -28, -19,
		-20, -28, -34, -53, -53, -37, -28, -21,
		-29, -21, -31, -52, -51, -40, -21, -26,
		-4, -11, -28, -30, -32, -27, -8, -9,
		15, 25, -6, -12, -12, -10, 32, 30,
		8, 29, 19, -12, 12, -2, 42, 22,
	},
}

var EgPST = [7][64]int{
	engine.Pawn: {
		0, 0, -1, 0, -1, 0, 0, 0,
		92, 92, 84, 69, 73, 68, 86, 92,
		62, 62, 56, 38, 38, 40, 62, 62,
		38, 24, 18, 18, 18, 18, 23, 28,
		23, 16, 8, 8, 8, 8, 13, 13,
		11, 12, 5, 17, 17, 12, 3, 4,
		12, 9, 12, 12, 12, 12, 7, 3,
		0, 0, 0, 0, 0, 0, 0, 0,
	},
	engine.Knight: {
		-51, -32, -18, -21, -18, -25, -42, -62,
		-28, -8, -9, 11, 0, -11, -11, -33,
		-19, 2, 9, 10, 10, 12, 6, -20,
		-18, 10, 21, 24, 22, 18, 10, -18,
		-18, 3, 13, 25, 14, 16, 16, -18,
		-18, 5, -2, 10, 6, -2, -10, -18,
		-28, -10, -4, 1, 4, -7, -8, -28,
		-38, -28, -19, -18, -18, -18, -28, -48,
	},
	engine.Bishop: {
		-8, -5, -9, -3, 1, 1, 1, -8,
		0, 4, 6, -10, 3, 5, 1, -19,
		2, 4, -1, -2, -2, 5, 9, 2,
		2, 12, 8, 3, 3, 4, 4, 2,
		2, 7, 7, 11, -2, 0, 0, 2,
		1, 4, 7, 9, 11, -1, 4, 1,
		0, -7, -5, 1, 5, -2, 10, -12,
		-14, 0, -6, 0, 0, -2, -1, -9,
	},
	engine.Rook: {
		12, 12, 12, 12, 12, 12, 12, 10,
		17, 19, 20, 22, 12, 17, 17, 17,
		7, 12, 10, 12, 6, 9, 11, 6,
		7, 6, 12, 4, 4, 12, -1, 7,
		7, 11, 9, 2, -3, -2, -2, -2,
		4, 5, -3, 0, -7, -10, -4, -8,
		-1, 2, 3, 7, -3, -7, -6, -8,
		0, 5, 7, 5, 2, 2, -1, -12,
	},
	engine.Queen: {
		-13, 2, 2, 7, 7, 2, 2, -8,
		-15, -12, 4, 12, 12, 12, 12, 2,
		-11, -10, -7, 13, 17, 17, 12, 2,
		-1, -2, -5, -3, 15, 13, 12, 7,
		-4, -1, -6, 3, 1, 9, 12, 7,
		-2, -7, -4, -7, -7, 5, 6, 2,
		-5, -10, -6, -5, -3, -7, -12, -6,
		-10, -16, -4, 7, 2, -15, -13, -29,
	},
	engine.King: {
		-62, -40, -25, -25, -21, -18, -28, -38,
		-18, -8, 2, 12, 12, 2, -8, -18,
		-18, 2, 21, 18, 18, 32, 2, -18,
		-21, 2, 25, 30, 30, 34, 2, -18,
		-31, -8, 20, 28, 29, 22, 2, -23,
		-29, -8, 10, 19, 20, 14, 2, -21,
		-42, -23, -2, 0, 7, 0, -18, -35,
		-62, -42, -29, -29, -31, -28, -38, -62,
	},
}

// Evaluate returns a score for the current board position using tapered evaluation.
func Evaluate(b *engine.Board) int {
	// 1. Determine the game phase based on non-pawn material.
	mgW, egW, _ := calculatePhase(b)

	// 2. Calculate separate scores for midgame and endgame for both sides.
	mgWhite, egWhite := evaluateColor(b, engine.White)
	mgBlack, egBlack := evaluateColor(b, engine.Black)

	mgScore := mgWhite - mgBlack
	egScore := egWhite - egBlack

	// 3. Interpolate between the two based on the phase weight.
	// As pieces are captured, egW increases and mgW decreases.
	score := (mgScore*mgW + egScore*egW) / TotalPhase

	if b.SideToMove == engine.Black {
		return -score
	}
	return score
}

// calculatePhase determines how "close" we are to the endgame.
// It uses material weights to assign a phase value from 0 (Opening/Midgame) to TotalPhase (Endgame).
func calculatePhase(b *engine.Board) (int, int, int) {
	// Start with full phase (Endgame) and subtract based on pieces present.
	phase := TotalPhase

	phase -= (b.Pieces[engine.White][engine.Knight].Count() + b.Pieces[engine.Black][engine.Knight].Count()) * KnightPhase
	phase -= (b.Pieces[engine.White][engine.Bishop].Count() + b.Pieces[engine.Black][engine.Bishop].Count()) * BishopPhase
	phase -= (b.Pieces[engine.White][engine.Rook].Count() + b.Pieces[engine.Black][engine.Rook].Count()) * RookPhase
	phase -= (b.Pieces[engine.White][engine.Queen].Count() + b.Pieces[engine.Black][engine.Queen].Count()) * QueenPhase

	if phase < 0 {
		phase = 0
	}

	// egW (Endgame Weight) is higher when fewer pieces are on the board.
	// mgW (Midgame Weight) is higher when more pieces are on the board.
	egW := phase
	mgW := TotalPhase - phase

	return mgW, egW, phase
}

func evaluateColor(b *engine.Board, c engine.Color) (int, int) {
	mg, eg := 0, 0
	occ := b.Occupancy()

	// Pawns
	pawns := b.Pieces[c][engine.Pawn]
	enemyPawns := b.Pieces[c^1][engine.Pawn]
	mg += pawns.Count() * PawnMG
	eg += pawns.Count() * PawnEG
	pawnCopy := pawns
	for pawnCopy != 0 {
		sq := pawnCopy.PopLSB()
		mg += getPST(engine.Pawn, sq, c, true)
		eg += getPST(engine.Pawn, sq, c, false)

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
		isBackward := false
		if !supported && !phalanx {
			// Check if any friendly pawns are on adjacent files at this rank or behind
			hasAdjacentBehind := false
			if c == engine.White {
				for r := 0; r <= rank; r++ {
					if (file > 0 && pawns.Test(engine.NewSquare(file-1, r))) ||
						(file < 7 && pawns.Test(engine.NewSquare(file+1, r))) {
						hasAdjacentBehind = true
						break
					}
				}
				// If no neighbors behind, check if the square in front is attacked by enemy pawn
				if !hasAdjacentBehind && rank < 7 {
					if (file > 0 && enemyPawns.Test(engine.NewSquare(file-1, rank+1))) ||
						(file < 7 && enemyPawns.Test(engine.NewSquare(file+1, rank+1))) {
						isBackward = true
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
						isBackward = true
					}
				}
			}
		}

		if isBackward {
			mg += PawnBackwardMG
			eg += PawnBackwardEG
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

	// Knights
	knights := b.Pieces[c][engine.Knight]
	mg += knights.Count() * KnightMG
	eg += knights.Count() * KnightEG
	for knights != 0 {
		sq := knights.PopLSB()
		mg += getPST(engine.Knight, sq, c, true)
		eg += getPST(engine.Knight, sq, c, false)
		mobility := engine.KnightAttacks[sq].Count()
		mg += mobility * KnightMobilityMG
		eg += mobility * KnightMobilityEG
	}

	// Bishops
	bishops := b.Pieces[c][engine.Bishop]
	mg += bishops.Count() * BishopMG
	eg += bishops.Count() * BishopEG
	if bishops.Count() >= 2 {
		mg += BishopPairMG
		eg += BishopPairEG
	}
	for bishops != 0 {
		sq := bishops.PopLSB()
		mg += getPST(engine.Bishop, sq, c, true)
		eg += getPST(engine.Bishop, sq, c, false)
		mobility := engine.GetBishopAttacks(sq, occ).Count()
		mg += mobility * BishopMobilityMG
		eg += mobility * BishopMobilityEG
	}

	// Rooks
	rooks := b.Pieces[c][engine.Rook]
	mg += rooks.Count() * RookMG
	eg += rooks.Count() * RookEG
	for rooks != 0 {
		sq := rooks.PopLSB()
		mg += getPST(engine.Rook, sq, c, true)
		eg += getPST(engine.Rook, sq, c, false)
		mobility := engine.GetRookAttacks(sq, occ).Count()
		mg += mobility * RookMobilityMG
		eg += mobility * RookMobilityEG
	}

	// Queens
	queens := b.Pieces[c][engine.Queen]
	mg += queens.Count() * QueenMG
	eg += queens.Count() * QueenEG
	for queens != 0 {
		sq := queens.PopLSB()
		mg += getPST(engine.Queen, sq, c, true)
		eg += getPST(engine.Queen, sq, c, false)
		mobility := engine.GetQueenAttacks(sq, occ).Count()
		mg += mobility * QueenMobilityMG
		eg += mobility * QueenMobilityEG
	}

	// King
	kingBB := b.Pieces[c][engine.King]
	if !kingBB.IsEmpty() {
		sq := kingBB.LSB()
		mg += getPST(engine.King, sq, c, true)
		eg += getPST(engine.King, sq, c, false)

		// King Safety (Pawn Shield) - only in Midgame
		mg += evaluateKingSafety(b, c, sq)
	}

	// Threats Evaluation
	them := c ^ 1
	enemyOcc := b.Colors[them]
	usOcc := b.Colors[c]

	for pt := engine.Pawn; pt <= engine.Queen; pt++ {
		subset := b.Pieces[c][pt]
		for subset != 0 {
			sq := subset.PopLSB()

			// Get all pieces attacking this square
			attackers := b.AllAttackers(sq, occ)
			enemyAttackers := attackers & enemyOcc

			if !enemyAttackers.IsEmpty() {
				defenders := attackers & usOcc

				if defenders.IsEmpty() {
					// Hanging piece penalty: Scale by piece value
					mg -= engine.PieceValues[pt] / HangingDivisorMG
					eg -= engine.PieceValues[pt] / HangingDivisorEG
				} else {
					// Defended piece, but check for attacks by lesser pieces
					weakestEnemyAttacker := engine.None
					for ept := engine.Pawn; ept < pt; ept++ {
						if !(enemyAttackers & b.Pieces[them][ept]).IsEmpty() {
							weakestEnemyAttacker = ept
							break
						}
					}

					if weakestEnemyAttacker != engine.None {
						// Attacked by lesser piece (e.g. Knight attacked by Pawn)
						mg += WeakAttackerMG
						eg += WeakAttackerEG
					}
				}
			}
		}
	}

	return mg, eg
}

func evaluateKingSafety(b *engine.Board, c engine.Color, kingSq engine.Square) int {
	score := 0
	pawns := b.Pieces[c][engine.Pawn]
	rank := kingSq.Rank()
	file := kingSq.File()

	// 1. Pawn Shield
	if c == engine.White {
		if rank < 7 {
			for f := file - 1; f <= file+1; f++ {
				if f >= 0 && f <= 7 {
					if pawns.Test(engine.NewSquare(f, rank+1)) {
						score += KingShieldClose
					} else if rank < 6 && pawns.Test(engine.NewSquare(f, rank+2)) {
						score += KingShieldFar
					}
				}
			}
		}
	} else {
		if rank > 0 {
			for f := file - 1; f <= file+1; f++ {
				if f >= 0 && f <= 7 {
					if pawns.Test(engine.NewSquare(f, rank-1)) {
						score += KingShieldClose
					} else if rank > 1 && pawns.Test(engine.NewSquare(f, rank-2)) {
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
func getPST(pt engine.PieceType, sq engine.Square, c engine.Color, midgame bool) int {
	rank := int(sq) / 8
	file := int(sq) % 8

	index := 0
	if c == engine.White {
		index = (7-rank)*8 + file
	} else {
		index = rank*8 + file
	}

	if midgame {
		return MgPST[pt][index]
	}
	return EgPST[pt][index]
}
