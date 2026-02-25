package eval

import (
	"github.com/Skiddle-Labs/axon-engine/internal/engine"
)

// Material values for Midgame (MG) and Endgame (EG)
var (
	PawnMG, PawnEG     = 82, 94
	KnightMG, KnightEG = 337, 281
	BishopMG, BishopEG = 365, 297
	RookMG, RookEG     = 477, 512
	QueenMG, QueenEG   = 1025, 936
)

// Pawn Structure Weights
var (
	PawnDoubledMG, PawnDoubledEG     = -10, -10
	PawnIsolatedMG, PawnIsolatedEG   = -20, -20
	PawnSupportedMG, PawnSupportedEG = 15, 20
	PawnPhalanxMG, PawnPhalanxEG     = 10, 12
	PawnBackwardMG, PawnBackwardEG   = -15, -20
	PawnPassedMG, PawnPassedEG       = 2, 5 // Multipliers for rank*rank
)

// Mobility Weights
var (
	KnightMobilityMG, KnightMobilityEG = 2, 2
	BishopMobilityMG, BishopMobilityEG = 3, 3
	RookMobilityMG, RookMobilityEG     = 2, 2
	QueenMobilityMG, QueenMobilityEG   = 1, 1
)

// Other Positional Weights
var (
	BishopPairMG, BishopPairEG         = 30, 50
	WeakAttackerMG, WeakAttackerEG     = -25, -40
	HangingDivisorMG, HangingDivisorEG = 4, 2 // Penalty = PieceValue / Divisor
)

// King Safety Tables and Weights
var KingAttackerWeight = [7]int{
	0, // None
	0, // Pawn
	2, // Knight
	2, // Bishop
	3, // Rook
	5, // Queen
	0, // King
}

var (
	KingShieldClose = 10
	KingShieldFar   = 5
)

var SafetyTable = [100]int{
	0, 0, 1, 2, 3, 5, 7, 9, 12, 15,
	18, 22, 26, 30, 35, 39, 44, 50, 56, 62,
	68, 75, 82, 89, 97, 105, 113, 122, 131, 140,
	150, 160, 171, 182, 194, 206, 218, 231, 244, 258,
	272, 287, 302, 318, 334, 351, 368, 386, 404, 423,
	442, 462, 482, 503, 524, 546, 568, 591, 614, 638,
	662, 687, 712, 738, 764, 791, 818, 846, 874, 903,
	932, 962, 992, 1023, 1054, 1086, 1118, 1151, 1184, 1218,
	1252, 1287, 1322, 1358, 1394, 1431, 1468, 1506, 1544, 1583,
	1622, 1662, 1702, 1743, 1784, 1826, 1868, 1911, 1954, 1998,
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
		0, 0, 0, 0, 0, 0, 0, 0,
		50, 50, 50, 50, 50, 50, 50, 50,
		10, 10, 20, 30, 30, 20, 10, 10,
		5, 5, 10, 25, 25, 10, 5, 5,
		0, 0, 0, 20, 20, 0, 0, 0,
		5, -5, -10, 0, 0, -10, -5, 5,
		5, 10, 10, -20, -20, 10, 10, 5,
		0, 0, 0, 0, 0, 0, 0, 0,
	},
	engine.Knight: {
		-50, -40, -30, -30, -30, -30, -40, -50,
		-40, -20, 0, 0, 0, 0, -20, -40,
		-30, 0, 10, 15, 15, 10, 0, -30,
		-30, 5, 15, 20, 20, 15, 5, -30,
		-30, 0, 15, 20, 20, 15, 0, -30,
		-30, 5, 10, 15, 15, 10, 5, -30,
		-40, -20, 0, 5, 5, 0, -20, -40,
		-50, -40, -30, -30, -30, -30, -40, -50,
	},
	engine.Bishop: {
		-20, -10, -10, -10, -10, -10, -10, -20,
		-10, 0, 0, 0, 0, 0, 0, -10,
		-10, 0, 5, 10, 10, 5, 0, -10,
		-10, 5, 5, 10, 10, 5, 5, -10,
		-10, 0, 10, 10, 10, 10, 0, -10,
		-10, 10, 10, 10, 10, 10, 10, -10,
		-10, 5, 0, 0, 0, 0, 5, -10,
		-20, -10, -10, -10, -10, -10, -10, -20,
	},
	engine.Rook: {
		0, 0, 0, 0, 0, 0, 0, 0,
		5, 10, 10, 10, 10, 10, 10, 5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		0, 0, 0, 5, 5, 0, 0, 0,
	},
	engine.Queen: {
		-20, -10, -10, -5, -5, -10, -10, -20,
		-10, 0, 0, 0, 0, 0, 0, -10,
		-10, 0, 5, 5, 5, 5, 0, -10,
		-5, 0, 5, 5, 5, 5, 0, -5,
		0, 0, 5, 5, 5, 5, 0, -5,
		-10, 5, 5, 5, 5, 5, 0, -10,
		-10, 0, 5, 0, 0, 0, 0, -10,
		-20, -10, -10, -5, -5, -10, -10, -20,
	},
	engine.King: {
		-30, -40, -40, -50, -50, -40, -40, -30,
		-30, -40, -40, -50, -50, -40, -40, -30,
		-30, -40, -40, -50, -50, -40, -40, -30,
		-30, -40, -40, -50, -50, -40, -40, -30,
		-20, -30, -30, -40, -40, -30, -30, -20,
		-10, -20, -20, -20, -20, -20, -20, -10,
		20, 20, 0, 0, 0, 0, 20, 20,
		20, 30, 10, 0, 0, 10, 30, 20,
	},
}

var EgPST = [7][64]int{
	engine.Pawn: {
		0, 0, 0, 0, 0, 0, 0, 0,
		80, 80, 80, 80, 80, 80, 80, 80,
		50, 50, 50, 50, 50, 50, 50, 50,
		30, 30, 30, 30, 30, 30, 30, 30,
		20, 20, 20, 20, 20, 20, 20, 20,
		10, 10, 10, 10, 10, 10, 10, 10,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
	},
	engine.Knight: {
		-50, -40, -30, -30, -30, -30, -40, -50,
		-40, -20, 0, 5, 5, 0, -20, -40,
		-30, 5, 10, 15, 15, 10, 5, -30,
		-30, 0, 15, 20, 20, 15, 0, -30,
		-30, 5, 15, 20, 20, 15, 5, -30,
		-30, 0, 10, 15, 15, 10, 0, -30,
		-40, -20, 0, 0, 0, 0, -20, -40,
		-50, -40, -30, -30, -30, -30, -40, -50,
	},
	engine.Bishop: {
		-20, -10, -10, -10, -10, -10, -10, -20,
		-10, 0, 0, 0, 0, 0, 0, -10,
		-10, 0, 5, 10, 10, 5, 0, -10,
		-10, 5, 5, 10, 10, 5, 5, -10,
		-10, 0, 10, 10, 10, 10, 0, -10,
		-10, 10, 10, 10, 10, 10, 10, -10,
		-10, 5, 0, 0, 0, 0, 5, -10,
		-20, -10, -10, -10, -10, -10, -10, -20,
	},
	engine.Rook: {
		0, 0, 0, 0, 0, 0, 0, 0,
		5, 10, 10, 10, 10, 10, 10, 5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		0, 0, 0, 5, 5, 0, 0, 0,
	},
	engine.Queen: {
		-20, -10, -10, -5, -5, -10, -10, -20,
		-10, 0, 0, 0, 0, 0, 0, -10,
		-10, 0, 5, 5, 5, 5, 0, -10,
		-5, 0, 5, 5, 5, 5, 0, -5,
		0, 0, 5, 5, 5, 5, 0, -5,
		-10, 5, 5, 5, 5, 5, 0, -10,
		-10, 0, 5, 0, 0, 0, 0, -10,
		-20, -10, -10, -5, -5, -10, -10, -20,
	},
	engine.King: {
		-50, -40, -30, -20, -20, -30, -40, -50,
		-30, -20, -10, 0, 0, -10, -20, -30,
		-30, -10, 20, 30, 30, 20, -10, -30,
		-30, -10, 30, 40, 40, 30, -10, -30,
		-30, -10, 30, 40, 40, 30, -10, -30,
		-30, -10, 20, 30, 30, 20, -10, -30,
		-30, -30, 0, 0, 0, 0, -30, -30,
		-50, -30, -30, -30, -30, -30, -30, -50,
	},
}

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

		// Connected pawns
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

		// Phalanx pawns
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
		mg += evaluateKingSafety(b, c, sq)
	}

	// Threats
	them := c ^ 1
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
					mg -= engine.PieceValues[pt] / HangingDivisorMG
					eg -= engine.PieceValues[pt] / HangingDivisorEG
				} else {
					weakestEnemyAttacker := engine.None
					for ept := engine.Pawn; ept < pt; ept++ {
						if !(enemyAttackers & b.Pieces[them][ept]).IsEmpty() {
							weakestEnemyAttacker = ept
							break
						}
					}
					if weakestEnemyAttacker != engine.None {
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
