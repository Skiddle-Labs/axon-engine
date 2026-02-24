package eval

import (
	"github.com/personal-github/axon-engine/internal/board"
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
	board.Pawn: {
		0, 0, 0, 0, 0, 0, 0, 0,
		50, 50, 50, 50, 50, 50, 50, 50,
		10, 10, 20, 30, 30, 20, 10, 10,
		5, 5, 10, 25, 25, 10, 5, 5,
		0, 0, 0, 20, 20, 0, 0, 0,
		5, -5, -10, 0, 0, -10, -5, 5,
		5, 10, 10, -20, -20, 10, 10, 5,
		0, 0, 0, 0, 0, 0, 0, 0,
	},
	board.Knight: {
		-50, -40, -30, -30, -30, -30, -40, -50,
		-40, -20, 0, 0, 0, 0, -20, -40,
		-30, 0, 10, 15, 15, 10, 0, -30,
		-30, 5, 15, 20, 20, 15, 5, -30,
		-30, 0, 15, 20, 20, 15, 0, -30,
		-30, 5, 10, 15, 15, 10, 5, -30,
		-40, -20, 0, 5, 5, 0, -20, -40,
		-50, -40, -30, -30, -30, -30, -40, -50,
	},
	board.Bishop: {
		-20, -10, -10, -10, -10, -10, -10, -20,
		-10, 0, 0, 0, 0, 0, 0, -10,
		-10, 0, 5, 10, 10, 5, 0, -10,
		-10, 5, 5, 10, 10, 5, 5, -10,
		-10, 0, 10, 10, 10, 10, 0, -10,
		-10, 10, 10, 10, 10, 10, 10, -10,
		-10, 5, 0, 0, 0, 0, 5, -10,
		-20, -10, -10, -10, -10, -10, -10, -20,
	},
	board.Rook: {
		0, 0, 0, 0, 0, 0, 0, 0,
		5, 10, 10, 10, 10, 10, 10, 5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		0, 0, 0, 5, 5, 0, 0, 0,
	},
	board.Queen: {
		-20, -10, -10, -5, -5, -10, -10, -20,
		-10, 0, 0, 0, 0, 0, 0, -10,
		-10, 0, 5, 5, 5, 5, 0, -10,
		-5, 0, 5, 5, 5, 5, 0, -5,
		0, 0, 5, 5, 5, 5, 0, -5,
		-10, 5, 5, 5, 5, 5, 0, -10,
		-10, 0, 5, 0, 0, 0, 0, -10,
		-20, -10, -10, -5, -5, -10, -10, -20,
	},
	board.King: {
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
	board.Pawn: {
		0, 0, 0, 0, 0, 0, 0, 0,
		80, 80, 80, 80, 80, 80, 80, 80,
		50, 50, 50, 50, 50, 50, 50, 50,
		30, 30, 30, 30, 30, 30, 30, 30,
		20, 20, 20, 20, 20, 20, 20, 20,
		10, 10, 10, 10, 10, 10, 10, 10,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
	},
	board.Knight: {
		-50, -40, -30, -30, -30, -30, -40, -50,
		-40, -20, 0, 5, 5, 0, -20, -40,
		-30, 5, 10, 15, 15, 10, 5, -30,
		-30, 0, 15, 20, 20, 15, 0, -30,
		-30, 5, 15, 20, 20, 15, 5, -30,
		-30, 0, 10, 15, 15, 10, 0, -30,
		-40, -20, 0, 0, 0, 0, -20, -40,
		-50, -40, -30, -30, -30, -30, -40, -50,
	},
	board.Bishop: {
		-20, -10, -10, -10, -10, -10, -10, -20,
		-10, 0, 0, 0, 0, 0, 0, -10,
		-10, 0, 5, 10, 10, 5, 0, -10,
		-10, 5, 5, 10, 10, 5, 5, -10,
		-10, 0, 10, 10, 10, 10, 0, -10,
		-10, 10, 10, 10, 10, 10, 10, -10,
		-10, 5, 0, 0, 0, 0, 5, -10,
		-20, -10, -10, -10, -10, -10, -10, -20,
	},
	board.Rook: {
		0, 0, 0, 0, 0, 0, 0, 0,
		5, 10, 10, 10, 10, 10, 10, 5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		0, 0, 0, 5, 5, 0, 0, 0,
	},
	board.Queen: {
		-20, -10, -10, -5, -5, -10, -10, -20,
		-10, 0, 0, 0, 0, 0, 0, -10,
		-10, 0, 5, 5, 5, 5, 0, -10,
		-5, 0, 5, 5, 5, 5, 0, -5,
		0, 0, 5, 5, 5, 5, 0, -5,
		-10, 5, 5, 5, 5, 5, 0, -10,
		-10, 0, 5, 0, 0, 0, 0, -10,
		-20, -10, -10, -5, -5, -10, -10, -20,
	},
	board.King: {
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
func Evaluate(b *board.Board) int {
	mgW, egW, _ := calculatePhase(b)

	mgWhite, egWhite := evaluateColor(b, board.White)
	mgBlack, egBlack := evaluateColor(b, board.Black)

	mgScore := mgWhite - mgBlack
	egScore := egWhite - egBlack

	// Interpolate scores based on the phase
	score := (mgScore*mgW + egScore*egW) / TotalPhase

	if b.SideToMove == board.Black {
		return -score
	}
	return score
}

func calculatePhase(b *board.Board) (int, int, int) {
	phase := TotalPhase

	phase -= (b.Pieces[board.White][board.Knight].Count() + b.Pieces[board.Black][board.Knight].Count()) * KnightPhase
	phase -= (b.Pieces[board.White][board.Bishop].Count() + b.Pieces[board.Black][board.Bishop].Count()) * BishopPhase
	phase -= (b.Pieces[board.White][board.Rook].Count() + b.Pieces[board.Black][board.Rook].Count()) * RookPhase
	phase -= (b.Pieces[board.White][board.Queen].Count() + b.Pieces[board.Black][board.Queen].Count()) * QueenPhase

	if phase < 0 {
		phase = 0
	}

	egW := phase
	mgW := TotalPhase - phase

	return mgW, egW, phase
}

func evaluateColor(b *board.Board, c board.Color) (int, int) {
	mg, eg := 0, 0
	occ := b.Occupancy()

	// Pawns
	pawns := b.Pieces[c][board.Pawn]
	mg += pawns.Count() * PawnMG
	eg += pawns.Count() * PawnEG
	pawnCopy := pawns
	for pawnCopy != 0 {
		sq := pawnCopy.PopLSB()
		mg += getPST(board.Pawn, sq, c, true)
		eg += getPST(board.Pawn, sq, c, false)

		// Structure
		if (pawns & (board.FileA << sq.File())).Count() > 1 {
			mg -= 10
			eg -= 10
		}
		isIsolated := true
		if sq.File() > 0 && (pawns&(board.FileA<<(sq.File()-1))) != 0 {
			isIsolated = false
		}
		if sq.File() < 7 && (pawns&(board.FileA<<(sq.File()+1))) != 0 {
			isIsolated = false
		}
		if isIsolated {
			mg -= 20
			eg -= 20
		}
	}

	// Knights
	knights := b.Pieces[c][board.Knight]
	mg += knights.Count() * KnightMG
	eg += knights.Count() * KnightEG
	for knights != 0 {
		sq := knights.PopLSB()
		mg += getPST(board.Knight, sq, c, true)
		eg += getPST(board.Knight, sq, c, false)
		mobility := board.KnightAttacks[sq].Count()
		mg += mobility * 2
		eg += mobility * 2
	}

	// Bishops
	bishops := b.Pieces[c][board.Bishop]
	mg += bishops.Count() * BishopMG
	eg += bishops.Count() * BishopEG
	for bishops != 0 {
		sq := bishops.PopLSB()
		mg += getPST(board.Bishop, sq, c, true)
		eg += getPST(board.Bishop, sq, c, false)
		mobility := board.GetBishopAttacks(sq, occ).Count()
		mg += mobility * 3
		eg += mobility * 3
	}

	// Rooks
	rooks := b.Pieces[c][board.Rook]
	mg += rooks.Count() * RookMG
	eg += rooks.Count() * RookEG
	for rooks != 0 {
		sq := rooks.PopLSB()
		mg += getPST(board.Rook, sq, c, true)
		eg += getPST(board.Rook, sq, c, false)
		mobility := board.GetRookAttacks(sq, occ).Count()
		mg += mobility * 2
		eg += mobility * 2
	}

	// Queens
	queens := b.Pieces[c][board.Queen]
	mg += queens.Count() * QueenMG
	eg += queens.Count() * QueenEG
	for queens != 0 {
		sq := queens.PopLSB()
		mg += getPST(board.Queen, sq, c, true)
		eg += getPST(board.Queen, sq, c, false)
		mobility := board.GetQueenAttacks(sq, occ).Count()
		mg += mobility * 1
		eg += mobility * 1
	}

	// King
	kingBB := b.Pieces[c][board.King]
	if !kingBB.IsEmpty() {
		sq := kingBB.LSB()
		mg += getPST(board.King, sq, c, true)
		eg += getPST(board.King, sq, c, false)

		// King Safety (Pawn Shield) - only in Midgame
		mg += evaluateKingSafety(b, c, sq)
	}

	return mg, eg
}

func evaluateKingSafety(b *board.Board, c board.Color, kingSq board.Square) int {
	score := 0
	pawns := b.Pieces[c][board.Pawn]
	rank := kingSq.Rank()
	file := kingSq.File()

	if c == board.White {
		if rank < 7 {
			for f := file - 1; f <= file+1; f++ {
				if f >= 0 && f <= 7 {
					if pawns.Test(board.NewSquare(f, rank+1)) {
						score += 10
					} else if rank < 6 && pawns.Test(board.NewSquare(f, rank+2)) {
						score += 5
					}
				}
			}
		}
	} else {
		if rank > 0 {
			for f := file - 1; f <= file+1; f++ {
				if f >= 0 && f <= 7 {
					if pawns.Test(board.NewSquare(f, rank-1)) {
						score += 10
					} else if rank > 1 && pawns.Test(board.NewSquare(f, rank-2)) {
						score += 5
					}
				}
			}
		}
	}
	return score
}

func getPST(pt board.PieceType, sq board.Square, c board.Color, midgame bool) int {
	index := int(sq)
	if c == board.Black {
		rank := int(sq) / 8
		file := int(sq) % 8
		index = (7-rank)*8 + file
	}

	if midgame {
		return mgPST[pt][index]
	}
	return egPST[pt][index]
}
