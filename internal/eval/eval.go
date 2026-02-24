package eval

import (
	"github.com/personal-github/axon-engine/internal/engine"
)

// Material values for Midgame (MG) and Endgame (EG)
// These values are tuned for tapered evaluation.
const (
	PawnMG, PawnEG     = 82, 94
	KnightMG, KnightEG = 337, 281
	BishopMG, BishopEG = 365, 297
	RookMG, RookEG     = 477, 512
	QueenMG, QueenEG   = 1025, 936

	// Phase values for interpolation
	// These determine how much each piece contributes to the "midgame-ness" of a position.
	KnightPhase = 1
	BishopPhase = 1
	RookPhase   = 2
	QueenPhase  = 4
	TotalPhase  = 24 // 4*Knight + 4*Bishop + 4*Rook + 2*Queen
)

// Piece-Square Tables (PST)
// Orientation: The tables are stored from Rank 8 (top) to Rank 1 (bottom).
// This allows for a visual representation that matches a physical chess board.

// mgPST handles the positional bonuses in the Midgame.
var mgPST = [7][64]int{
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

// egPST handles the positional bonuses in the Endgame.
var egPST = [7][64]int{
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

// Game Phases in chess are traditionally divided into:
// 1. Opening: Pieces are developed, king is brought to safety.
// 2. Middlegame: Most pieces are active, tactical complexities arise.
// 3. Endgame: Few pieces remain, king becomes active, and pawn promotion is the primary goal.
//
// Axon uses Tapered Evaluation to transition between these phases. Instead of abrupt
// changes, it calculates a Midgame (MG) and Endgame (EG) score simultaneously and
// interpolates between them based on the material remaining on the board.

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

		// Connected pawns: pawns covering each other
		connected := false
		if c == engine.White {
			if file > 0 && pawns.Test(engine.NewSquare(file-1, rank-1)) {
				connected = true
			}
			if file < 7 && pawns.Test(engine.NewSquare(file+1, rank-1)) {
				connected = true
			}
		} else {
			if file > 0 && pawns.Test(engine.NewSquare(file-1, rank+1)) {
				connected = true
			}
			if file < 7 && pawns.Test(engine.NewSquare(file+1, rank+1)) {
				connected = true
			}
		}
		if connected {
			mg += 10
			eg += 10
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

	// Threats Evaluation:
	// A "Threat" exists when a piece is vulnerable to capture in a way that loses material.
	// 1. Hanging Pieces: Attacked by the enemy and not defended by any friendly piece.
	// 2. Bad Trades: Defended, but attacked by an enemy piece of lesser value (e.g. Rook vs Pawn).
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
	return score
}

// getPST maps a square to its value in the Piece-Square Table.
// It uses a relative mapping so that both sides use the same table from their own perspective.
func getPST(pt engine.PieceType, sq engine.Square, c engine.Color, midgame bool) int {
	rank := int(sq) / 8
	file := int(sq) % 8

	index := 0
	if c == engine.White {
		// Table is stored Rank 8 to Rank 1. White A1 is index 56.
		index = (7-rank)*8 + file
	} else {
		// For Black, Rank 8 is at the top of their perspective (index 0..7).
		index = rank*8 + file
	}

	if midgame {
		return mgPST[pt][index]
	}
	return egPST[pt][index]
}
