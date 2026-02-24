package search

import (
	"fmt"
	"time"

	"github.com/personal-github/axon-engine/internal/board"
	"github.com/personal-github/axon-engine/internal/eval"
)

const (
	Infinity  = 30000
	MateScore = 20000
)

var GlobalTT = NewTranspositionTable(64)

// Engine handles the search process.
type Engine struct {
	Board     *board.Board
	Nodes     uint64
	StartTime time.Time
	TimeLimit time.Duration
	Stopped   bool
	TT        *TranspositionTable

	KillerMoves  [64][2]board.Move
	HistoryTable [2][64][64]int
}

// NewEngine creates a new search engine instance.
func NewEngine(b *board.Board) *Engine {
	return &Engine{
		Board: b,
		TT:    GlobalTT,
	}
}

// Search finds the best move for the current position using iterative deepening.
func (e *Engine) Search(maxDepth int) board.Move {
	e.Nodes = 0
	e.StartTime = time.Now()
	e.Stopped = false
	globalBestMove := board.NoMove

	lastScore := 0
	for depth := 1; depth <= maxDepth; depth++ {
		alpha := -Infinity
		beta := Infinity
		delta := 50

		if depth > 4 {
			alpha = lastScore - delta
			beta = lastScore + delta
		}

		for {
			bestMove := board.NoMove
			score := -Infinity

			ml := e.Board.GenerateMoves()
			// Probe TT for the best move to order it first
			_, ttMove, _ := e.TT.Probe(e.Board.Hash, depth, alpha, beta, e.Board.Ply)
			e.orderMoves(&ml, ttMove)

			currAlpha := alpha
			for i := 0; i < ml.Count; i++ {
				move := ml.Moves[i]
				if !e.Board.MakeMove(move) {
					continue
				}

				s := -e.negamax(depth-1, -beta, -currAlpha)
				e.Board.UnmakeMove(move)

				if e.Stopped {
					break
				}

				if s > score {
					score = s
					bestMove = move
				}
				if s > currAlpha {
					currAlpha = s
				}
			}

			if e.Stopped {
				break
			}

			if score <= alpha {
				alpha -= delta
				delta *= 2
				if alpha < -Infinity {
					alpha = -Infinity
				}
			} else if score >= beta {
				beta += delta
				delta *= 2
				if beta > Infinity {
					beta = Infinity
				}
			} else {
				lastScore = score
				if bestMove != board.NoMove {
					globalBestMove = bestMove
					duration := time.Since(e.StartTime).Seconds()
					nps := uint64(0)
					if duration > 0.001 {
						nps = uint64(float64(e.Nodes) / duration)
					}

					fmt.Printf("info depth %d score cp %d nodes %d nps %d time %d pv %s\n",
						depth, lastScore, e.Nodes, nps, int(duration*1000), globalBestMove.String())
				}
				break
			}

			if alpha == -Infinity && beta == Infinity {
				lastScore = score
				if bestMove != board.NoMove {
					globalBestMove = bestMove
					duration := time.Since(e.StartTime).Seconds()
					nps := uint64(0)
					if duration > 0.001 {
						nps = uint64(float64(e.Nodes) / duration)
					}

					fmt.Printf("info depth %d score cp %d nodes %d nps %d time %d pv %s\n",
						depth, lastScore, e.Nodes, nps, int(duration*1000), globalBestMove.String())
				}
				break
			}
		}

		if e.Stopped {
			break
		}
	}

	return globalBestMove
}

// negamax is the core search algorithm with alpha-beta pruning.
func (e *Engine) negamax(depth, alpha, beta int) int {
	if e.Nodes&2047 == 0 && e.TimeLimit > 0 && time.Since(e.StartTime) >= e.TimeLimit {
		e.Stopped = true
	}

	if e.Stopped {
		return 0
	}

	e.Nodes++

	// 1. TT Probe
	ttScore, ttMove, found := e.TT.Probe(e.Board.Hash, depth, alpha, beta, e.Board.Ply)
	if found {
		return ttScore
	}

	inCheck := e.Board.IsSquareAttacked(e.Board.Pieces[e.Board.SideToMove][board.King].LSB(), e.Board.SideToMove^1)
	if inCheck {
		depth++
	}

	// Base case: reach depth 0, enter quiescence search to avoid horizon effect.
	if depth <= 0 {
		return e.quiescence(alpha, beta)
	}

	// Null Move Pruning (NMP)
	// If the current side has major pieces and is not in check, try skipping a turn.
	// If the resulting score is still >= beta, we assume the position is strong enough to fail-high.
	if depth >= 3 && !inCheck && e.Board.HasMajorPieces(e.Board.SideToMove) {
		e.Board.MakeNullMove()
		score := -e.negamax(depth-3, -beta, -beta+1)
		e.Board.UnmakeNullMove()

		if score >= beta {
			return beta
		}
	}

	alphaOrig := alpha
	bestMove := board.NoMove
	bestScore := -Infinity

	ml := e.Board.GenerateMoves()
	e.orderMoves(&ml, ttMove)

	legalMoves := 0
	for i := 0; i < ml.Count; i++ {
		move := ml.Moves[i]
		if !e.Board.MakeMove(move) {
			continue
		}

		legalMoves++
		score := -e.negamax(depth-1, -beta, -alpha)
		e.Board.UnmakeMove(move)

		if score > bestScore {
			bestScore = score
			bestMove = move
		}

		if score >= beta {
			// Store killer moves and history heuristic for quiet moves
			if move.Flags()&board.CaptureFlag == 0 && e.Board.Ply < 64 {
				if move != e.KillerMoves[e.Board.Ply][0] {
					e.KillerMoves[e.Board.Ply][1] = e.KillerMoves[e.Board.Ply][0]
					e.KillerMoves[e.Board.Ply][0] = move
				}
				e.HistoryTable[e.Board.SideToMove][move.From()][move.To()] += depth * depth
			}

			bestScore = beta
			break // Fail-high, beta cutoff
		}
		if score > alpha {
			alpha = score
		}
	}

	// Handle terminal nodes (Checkmate/Stalemate)
	if legalMoves == 0 {
		if inCheck {
			// Checkmate: return mate score adjusted by distance from root (ply)
			return -MateScore + e.Board.Ply
		}
		// Stalemate
		return 0
	}

	// TT Store
	flag := ExactFlag
	if bestScore <= alphaOrig {
		flag = AlphaFlag
	} else if bestScore >= beta {
		flag = BetaFlag
	}
	e.TT.Store(e.Board.Hash, depth, bestScore, flag, bestMove, e.Board.Ply)

	return bestScore
}

// quiescence search evaluates only "noisy" positions (captures) to stabilize the evaluation.
func (e *Engine) quiescence(alpha, beta int) int {
	if e.Nodes&2047 == 0 && e.TimeLimit > 0 && time.Since(e.StartTime) >= e.TimeLimit {
		e.Stopped = true
	}

	if e.Stopped {
		return 0
	}

	e.Nodes++

	// TT Probe
	ttScore, _, found := e.TT.Probe(e.Board.Hash, 0, alpha, beta, e.Board.Ply)
	if found {
		return ttScore
	}

	standingPat := eval.Evaluate(e.Board)

	if standingPat >= beta {
		return beta
	}
	if standingPat > alpha {
		alpha = standingPat
	}

	ml := e.Board.GenerateMoves()
	// Order moves in quiescence search too (captures only)
	e.orderMoves(&ml, board.NoMove)

	for i := 0; i < ml.Count; i++ {
		move := ml.Moves[i]
		// In quiescence, we only look at captures.
		if move.Flags()&board.CaptureFlag == 0 {
			continue
		}

		if !e.Board.MakeMove(move) {
			continue
		}

		score := -e.quiescence(-beta, -alpha)
		e.Board.UnmakeMove(move)

		if score >= beta {
			return beta
		}
		if score > alpha {
			alpha = score
		}
	}

	return alpha
}

// MVV-LVA (Most Valuable Victim - Least Valuable Aggressor) table.
// P=1, N=2, B=3, R=4, Q=5, K=6
var mvvLva = [7][7]int{
	// Aggressor:  None, P,  N,  B,  R,  Q,  K
	{0, 0, 0, 0, 0, 0, 0},       // Victim: None
	{0, 15, 14, 13, 12, 11, 10}, // Victim: Pawn
	{0, 25, 24, 23, 22, 21, 20}, // Victim: Knight
	{0, 35, 34, 33, 32, 31, 30}, // Victim: Bishop
	{0, 45, 44, 43, 42, 41, 40}, // Victim: Rook
	{0, 55, 54, 53, 52, 51, 50}, // Victim: Queen
	{0, 65, 64, 63, 62, 61, 60}, // Victim: King
}

// orderMoves sorts moves to improve alpha-beta pruning efficiency.
func (e *Engine) orderMoves(ml *board.MoveList, ttMove board.Move) {
	scores := make([]int, ml.Count)

	for i := 0; i < ml.Count; i++ {
		move := ml.Moves[i]

		// TT move gets highest priority
		if move == ttMove {
			scores[i] = 30000
			continue
		}

		flags := move.Flags()

		if flags&board.CaptureFlag != 0 {
			attacker := e.Board.PieceAt(move.From()).Type()
			victim := e.Board.PieceAt(move.To()).Type()

			if flags == board.EnPassantFlag {
				victim = board.Pawn
			}

			// Prioritize captures using MVV-LVA
			scores[i] = 20000 + mvvLva[victim][attacker]
		} else if flags&0x8000 != 0 {
			// Prioritize promotions
			scores[i] = 15000
		} else if e.Board.Ply < 64 && move == e.KillerMoves[e.Board.Ply][0] {
			scores[i] = 10000
		} else if e.Board.Ply < 64 && move == e.KillerMoves[e.Board.Ply][1] {
			scores[i] = 9000
		} else {
			scores[i] = e.HistoryTable[e.Board.SideToMove][move.From()][move.To()]
		}
	}

	// Selection sort (simple and avoids heap allocations for small lists)
	for i := 0; i < ml.Count-1; i++ {
		maxIdx := i
		for j := i + 1; j < ml.Count; j++ {
			if scores[j] > scores[maxIdx] {
				maxIdx = j
			}
		}
		if maxIdx != i {
			scores[i], scores[maxIdx] = scores[maxIdx], scores[i]
			ml.Moves[i], ml.Moves[maxIdx] = ml.Moves[maxIdx], ml.Moves[i]
		}
	}
}
