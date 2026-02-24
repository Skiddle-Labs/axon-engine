package eval

import (
	"github.com/personal-github/axon-engine/internal/engine"
)

// Material values for Midgame (MG) and Endgame (EG)
const (
	PawnMG, PawnEG     = 82, 94
	KnightMG, KnightEG = 337, 281
	BishopMG, BishopEG = 365, 297
	RookMG, RookEG     = 477, 512
	QueenMG, QueenEG   = 1025, 936

	// Phase values for interpolation
	KnightPhase = 1
	BishopPhase = 1
	RookPhase   = 2
	QueenPhase  = 4
	TotalPhase  = 24 // 4*1 + 4*1 + 4*2 + 2*4
)

// Piece-Square Tables (PST) for Midgame
var mgPST = [7][64]int{
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

// Piece-Square Tables (PST) for Endgame
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

// Evaluate returns a score for the current board position using tapered evaluation.
func Evaluate(b *engine.Board) int {
	mgW, egW, _ := calculatePhase(b)

	mgWhite, egWhite := evaluateColor(b, engine.White)
	mgBlack, egBlack := evaluateColor(b, engine.Black)

	mgScore := mgWhite - mgBlack
	egScore := egWhite - egBlack

	// Interpolate scores based on the phase
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

	// Threats: hanging pieces and bad trades
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
					// Piece is hanging
					mg -= engine.PieceValues[pt] / 3
					eg -= engine.PieceValues[pt] / 2
				} else {
					// Piece is defended, check for attacks by lesser pieces
					weakestEnemyAttacker := engine.None
					for ept := engine.Pawn; ept < pt; ept++ {
						if !(enemyAttackers & b.Pieces[them][ept]).IsEmpty() {
							weakestEnemyAttacker = ept
							break
						}
					}

					if weakestEnemyAttacker != engine.None {
						// Attacked by lesser piece
						mg -= 20
						eg -= 30
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

func getPST(pt engine.PieceType, sq engine.Square, c engine.Color, midgame bool) int {
	index := int(sq)
	if c == engine.Black {
		rank := int(sq) / 8
		file := int(sq) % 8
		index = (7-rank)*8 + file
	}

	if midgame {
		return mgPST[pt][index]
	}
	return egPST[pt][index]
}
