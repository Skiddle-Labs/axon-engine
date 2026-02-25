package eval

import (
	"github.com/Skiddle-Labs/axon-engine/internal/engine"
)

// Material values for Midgame (MG) and Endgame (EG)
// These values are exported to allow for automated tuning.
var (
	PawnMG, PawnEG     = 82, 94
	KnightMG, KnightEG = 337, 281
	BishopMG, BishopEG = 365, 297
	RookMG, RookEG     = 477, 512
	QueenMG, QueenEG   = 1025, 936
)

// King Safety Tables
var KingAttackerWeight = [7]int{
	0, // None
	0, // Pawn
	2, // Knight
	2, // Bishop
	3, // Rook
	5, // Queen
	0, // King
}

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
// These determine how much each piece contributes to the "midgame-ness" of a position.
var (
	KnightPhase = 1
	BishopPhase = 1
	RookPhase   = 2
	QueenPhase  = 4
	TotalPhase  = 24 // 4*Knight + 4*Bishop + 4*Rook + 2*Queen
)

// Piece-Square Tables (PST)
// Orientation: The tables are stored from Rank 8 (top) to Rank 1 (bottom).
// This allows for a visual representation that matches a physical chess board.

// MgPST handles the positional bonuses in the Midgame.
var MgPST = [7][64]int{
	engine.Pawn: {
		0, 0, 0, 0, 0, 0, 0, 0, // Rank 8
		50, 50, 50, 50, 50, 50, 50, 50, // Rank 7
		10, 10, 20, 30, 30, 20, 10, 10, // Rank 6
		5, 5, 10, 25, 25, 10, 5, 5, // Rank 5
		0, 0, 0, 20, 20, 0, 0, 0, // Rank 4
		5, -5, -10, 0, 0, -10, -5, 5, // Rank 3
		5, 10, 10, -20, -20, 10, 10, 5, // Rank 2
		0, 0, 0, 0, 0, 0, 0, 0, // Rank 1
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

// EgPST handles the positional bonuses in the Endgame.
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
			mg -= 10
			eg -= 10
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
			mg -= 20
			eg -= 20
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
			mg += 15
			eg += 20
		} else if phalanx {
			mg += 10
			eg += 12
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
			mg -= 15
			eg -= 20
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
			mg += bonus * 2
			eg += bonus * 5
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
		mg += mobility * 2
		eg += mobility * 2
	}

	// Bishops
	bishops := b.Pieces[c][engine.Bishop]
	mg += bishops.Count() * BishopMG
	eg += bishops.Count() * BishopEG
	if bishops.Count() >= 2 {
		mg += 30
		eg += 50
	}
	for bishops != 0 {
		sq := bishops.PopLSB()
		mg += getPST(engine.Bishop, sq, c, true)
		eg += getPST(engine.Bishop, sq, c, false)
		mobility := engine.GetBishopAttacks(sq, occ).Count()
		mg += mobility * 3
		eg += mobility * 3
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
		mg += mobility * 2
		eg += mobility * 2
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
		mg += mobility * 1
		eg += mobility * 1
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
					mg -= engine.PieceValues[pt] / 4
					eg -= engine.PieceValues[pt] / 2
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
						mg -= 25
						eg -= 40
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
						score += 10
					} else if rank < 6 && pawns.Test(engine.NewSquare(f, rank+2)) {
						score += 5
					}
				}
			}
		}
	} else {
		if rank > 0 {
			for f := file - 1; f <= file+1; f++ {
				if f >= 0 && f <= 7 {
					if pawns.Test(engine.NewSquare(f, rank-1)) {
						score += 10
					} else if rank > 1 && pawns.Test(engine.NewSquare(f, rank-2)) {
						score += 5
					}
				}
			}
		}
	}

	// 2. Attacking Zone
	them := c ^ 1
	occ := b.Occupancy()
	zone := engine.KingAttacks[kingSq] | (engine.Bitboard(1) << kingSq)

	attackerCount := 0
	attackerWeight := 0

	// Check Knights
	knights := b.Pieces[them][engine.Knight]
	for knights != 0 {
		sq := knights.PopLSB()
		if !(engine.KnightAttacks[sq] & zone).IsEmpty() {
			attackerCount++
			attackerWeight += KingAttackerWeight[engine.Knight]
		}
	}

	// Check Bishops
	bishops := b.Pieces[them][engine.Bishop]
	for bishops != 0 {
		sq := bishops.PopLSB()
		if !(engine.GetBishopAttacks(sq, occ) & zone).IsEmpty() {
			attackerCount++
			attackerWeight += KingAttackerWeight[engine.Bishop]
		}
	}

	// Check Rooks
	rooks := b.Pieces[them][engine.Rook]
	for rooks != 0 {
		sq := rooks.PopLSB()
		if !(engine.GetRookAttacks(sq, occ) & zone).IsEmpty() {
			attackerCount++
			attackerWeight += KingAttackerWeight[engine.Rook]
		}
	}

	// Check Queens
	queens := b.Pieces[them][engine.Queen]
	for queens != 0 {
		sq := queens.PopLSB()
		if !(engine.GetQueenAttacks(sq, occ) & zone).IsEmpty() {
			attackerCount++
			attackerWeight += KingAttackerWeight[engine.Queen]
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
