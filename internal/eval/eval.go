package eval

import (
	"github.com/personal-github/axon-engine/internal/board"
)

const (
	PawnValue   = 100
	KnightValue = 320
	BishopValue = 330
	RookValue   = 500
	QueenValue  = 900
)

// Piece-Square Tables (PST)
// Values are from white's perspective. For black, we flip the board.
// Table index 0 is A1, 63 is H8.

var pawnPST = [64]int{
	0, 0, 0, 0, 0, 0, 0, 0,
	50, 50, 50, 50, 50, 50, 50, 50,
	10, 10, 20, 30, 30, 20, 10, 10,
	5, 5, 10, 25, 25, 10, 5, 5,
	0, 0, 0, 20, 20, 0, 0, 0,
	5, -5, -10, 0, 0, -10, -5, 5,
	5, 10, 10, -20, -20, 10, 10, 5,
	0, 0, 0, 0, 0, 0, 0, 0,
}

var knightPST = [64]int{
	-50, -40, -30, -30, -30, -30, -40, -50,
	-40, -20, 0, 0, 0, 0, -20, -40,
	-30, 0, 10, 15, 15, 10, 0, -30,
	-30, 5, 15, 20, 20, 15, 5, -30,
	-30, 0, 15, 20, 20, 15, 0, -30,
	-30, 5, 10, 15, 15, 10, 5, -30,
	-40, -20, 0, 5, 5, 0, -20, -40,
	-50, -40, -30, -30, -30, -30, -40, -50,
}

var bishopPST = [64]int{
	-20, -10, -10, -10, -10, -10, -10, -20,
	-10, 0, 0, 0, 0, 0, 0, -10,
	-10, 0, 5, 10, 10, 5, 0, -10,
	-10, 5, 5, 10, 10, 5, 5, -10,
	-10, 0, 10, 10, 10, 10, 0, -10,
	-10, 10, 10, 10, 10, 10, 10, -10,
	-10, 5, 0, 0, 0, 0, 5, -10,
	-20, -10, -10, -10, -10, -10, -10, -20,
}

var rookPST = [64]int{
	0, 0, 0, 0, 0, 0, 0, 0,
	5, 10, 10, 10, 10, 10, 10, 5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	0, 0, 0, 5, 5, 0, 0, 0,
}

var queenPST = [64]int{
	-20, -10, -10, -5, -5, -10, -10, -20,
	-10, 0, 0, 0, 0, 0, 0, -10,
	-10, 0, 5, 5, 5, 5, 0, -10,
	-5, 0, 5, 5, 5, 5, 0, -5,
	0, 0, 5, 5, 5, 5, 0, -5,
	-10, 5, 5, 5, 5, 5, 0, -10,
	-10, 0, 5, 0, 0, 0, 0, -10,
	-20, -10, -10, -5, -5, -10, -10, -20,
}

var kingPST = [64]int{
	-30, -40, -40, -50, -50, -40, -40, -30,
	-30, -40, -40, -50, -50, -40, -40, -30,
	-30, -40, -40, -50, -50, -40, -40, -30,
	-30, -40, -40, -50, -50, -40, -40, -30,
	-20, -30, -30, -40, -40, -30, -30, -20,
	-10, -20, -20, -20, -20, -20, -20, -10,
	20, 20, 0, 0, 0, 0, 20, 20,
	20, 30, 10, 0, 0, 10, 30, 20,
}

// Evaluate returns a score for the current board position.
// The score is positive if the side to move is better, negative if worse.
func Evaluate(b *board.Board) int {
	whiteScore := evaluateColor(b, board.White)
	blackScore := evaluateColor(b, board.Black)

	score := whiteScore - blackScore

	if b.SideToMove == board.Black {
		return -score
	}
	return score
}

func evaluateColor(b *board.Board, c board.Color) int {
	score := 0

	// Piece bitboards copies (PopLSB will not affect original board)
	pawns := b.Pieces[c][board.Pawn]
	knights := b.Pieces[c][board.Knight]
	bishops := b.Pieces[c][board.Bishop]
	rooks := b.Pieces[c][board.Rook]
	queens := b.Pieces[c][board.Queen]
	king := b.Pieces[c][board.King]

	score += pawns.Count() * PawnValue
	pawnCopy := pawns
	for pawnCopy != 0 {
		sq := pawnCopy.PopLSB()
		score += getPSTValue(board.Pawn, sq, c)

		// Doubled pawns: more than one pawn on this file
		if (pawns & (board.FileA << sq.File())).Count() > 1 {
			score -= 10
		}

		// Isolated pawns: no pawns on adjacent files
		isIsolated := true
		if sq.File() > 0 && (pawns&(board.FileA<<(sq.File()-1))) != 0 {
			isIsolated = false
		}
		if sq.File() < 7 && (pawns&(board.FileA<<(sq.File()+1))) != 0 {
			isIsolated = false
		}
		if isIsolated {
			score -= 20
		}
	}

	score += knights.Count() * KnightValue
	for knights != 0 {
		sq := knights.PopLSB()
		score += getPSTValue(board.Knight, sq, c)
	}

	score += bishops.Count() * BishopValue
	for bishops != 0 {
		sq := bishops.PopLSB()
		score += getPSTValue(board.Bishop, sq, c)
	}

	score += rooks.Count() * RookValue
	for rooks != 0 {
		sq := rooks.PopLSB()
		score += getPSTValue(board.Rook, sq, c)
	}

	score += queens.Count() * QueenValue
	for queens != 0 {
		sq := queens.PopLSB()
		score += getPSTValue(board.Queen, sq, c)
	}

	if !king.IsEmpty() {
		sq := king.LSB()
		score += getPSTValue(board.King, sq, c)
	}

	return score
}

func getPSTValue(pt board.PieceType, sq board.Square, c board.Color) int {
	index := int(sq)
	if c == board.Black {
		// Flip rank for black (A1 <-> A8, etc)
		// Squares are 0-63 (A1..H8)
		rank := int(sq) / 8
		file := int(sq) % 8
		index = (7-rank)*8 + file
	}

	switch pt {
	case board.Pawn:
		return pawnPST[index]
	case board.Knight:
		return knightPST[index]
	case board.Bishop:
		return bishopPST[index]
	case board.Rook:
		return rookPST[index]
	case board.Queen:
		return queenPST[index]
	case board.King:
		return kingPST[index]
	}
	return 0
}
