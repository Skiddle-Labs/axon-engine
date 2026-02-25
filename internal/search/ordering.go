package search

import (
	"github.com/Skiddle-Labs/axon-engine/internal/engine"
	"github.com/Skiddle-Labs/axon-engine/internal/types"
)

var mvvLva = [7][7]int{
	{0, 0, 0, 0, 0, 0, 0},
	{0, 15, 14, 13, 12, 11, 10},
	{0, 25, 24, 23, 22, 21, 20},
	{0, 35, 34, 33, 32, 31, 30},
	{0, 45, 44, 43, 42, 41, 40},
	{0, 55, 54, 53, 52, 51, 50},
	{0, 65, 64, 63, 62, 61, 60},
}

func (e *Engine) orderMoves(ml *engine.MoveList, ttMove engine.Move, ply int) {
	var scores [256]int

	counterMove := engine.NoMove
	if e.Board.Ply > 0 && e.CounterMoves != nil {
		prevMove := e.Board.History[e.Board.Ply-1].Move
		counterMove = e.CounterMoves[prevMove.From()][prevMove.To()]
	}

	for i := 0; i < ml.Count; i++ {
		move := ml.Moves[i]

		if move == ttMove {
			scores[i] = 30000
			continue
		}

		flags := move.Flags()
		piece := e.Board.PieceAt(move.From()).Type()

		if flags&engine.CaptureFlag != 0 {
			victim := e.Board.PieceAt(move.To()).Type()
			if flags == engine.EnPassantFlag {
				victim = types.Pawn
			}
			if e.Board.SEE(move) >= 0 {
				scores[i] = 25000 + mvvLva[victim][piece]
			} else {
				scores[i] = 10000 + mvvLva[victim][piece]
			}

			if e.CaptureHistory != nil {
				scores[i] += e.CaptureHistory[e.Board.SideToMove][piece][victim][move.To()]
			}
		} else if flags&0x8000 != 0 {
			scores[i] = 20000
		} else if ply < 128 && move == e.KillerMoves[ply][0] {
			scores[i] = 15000
		} else if ply < 128 && move == e.KillerMoves[ply][1] {
			scores[i] = 14000
		} else if move == counterMove {
			scores[i] = 13000
		} else if e.HistoryTable != nil {
			scores[i] = e.HistoryTable[e.Board.SideToMove][piece][move.To()]
		}
	}

	for i := 0; i < ml.Count-1; i++ {
		bestIdx := i
		for j := i + 1; j < ml.Count; j++ {
			if scores[j] > scores[bestIdx] {
				bestIdx = j
			}
		}
		ml.Moves[i], ml.Moves[bestIdx] = ml.Moves[bestIdx], ml.Moves[i]
		scores[i], scores[bestIdx] = scores[bestIdx], scores[i]
	}
}
