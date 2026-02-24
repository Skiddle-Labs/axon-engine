package search

import (
	"fmt"
	"time"

	"github.com/personal-github/axon-engine/internal/engine"
	"github.com/personal-github/axon-engine/internal/eval"
)

const (
	Infinity  = 30000
	MateScore = 20000
)

var GlobalTT = NewTranspositionTable(64)

// Engine handles the search process.
type Engine struct {
	Board     *engine.Board
	Nodes     uint64
	StartTime time.Time
	TimeLimit time.Duration
	Stopped   bool
	TT        *TranspositionTable

	KillerMoves  [64][2]engine.Move
	HistoryTable [2][64][64]int
}

// NewEngine creates a new search engine instance.
func NewEngine(b *engine.Board) *Engine {
	return &Engine{
		Board: b,
		TT:    GlobalTT,
	}
}

// Search finds the best move for the current position using iterative deepening.
func (e *Engine) Search(maxDepth int) engine.Move {
	e.Nodes = 0
	e.StartTime = time.Now()
	e.Stopped = false
	globalBestMove := engine.NoMove

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
			bestMove := engine.NoMove
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

				var s int
				if i == 0 {
					// Full search for first move
					s = -e.negamax(depth-1, -beta, -currAlpha)
				} else {
					// LMR at root (mild)
					reduction := 0
					if depth >= 3 && i >= 6 && move.Flags()&engine.CaptureFlag == 0 && move.Flags()&0x8000 == 0 {
						reduction = 1
					}

					// Principal Variation Search (PVS) with null window
					s = -e.negamax(depth-1-reduction, -(currAlpha + 1), -currAlpha)

					// Re-search if reduced move beats alpha
					if s > currAlpha && reduction > 0 {
						s = -e.negamax(depth-1, -(currAlpha + 1), -currAlpha)
					}
					// Re-search with full window if null window search beats alpha
					if s > currAlpha && s < beta {
						s = -e.negamax(depth-1, -beta, -currAlpha)
					}
				}

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
				if bestMove != engine.NoMove {
					globalBestMove = bestMove
					e.printInfo(depth, score, globalBestMove)
				}
				break
			}

			if alpha == -Infinity && beta == Infinity {
				lastScore = score
				if bestMove != engine.NoMove {
					globalBestMove = bestMove
					e.printInfo(depth, score, globalBestMove)
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

	// 2. Repetition Detection
	for i := e.Board.Ply - 2; i >= e.Board.Ply-int(e.Board.HalfMoveClock); i -= 2 {
		if i >= 0 && e.Board.History[i].Hash == e.Board.Hash {
			return 0
		}
	}

	inCheck := e.Board.IsSquareAttacked(e.Board.Pieces[e.Board.SideToMove][engine.King].LSB(), e.Board.SideToMove^1)
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

	// Internal Iterative Deepening (IID)
	// If we don't have a best move from the TT, perform a shallow search to find one.
	if depth >= 5 && ttMove == engine.NoMove {
		e.negamax(depth-2, alpha, beta)
		_, ttMove, _ = e.TT.Probe(e.Board.Hash, depth, alpha, beta, e.Board.Ply)
	}

	alphaOrig := alpha
	bestMove := engine.NoMove
	bestScore := -Infinity
	standingPat := eval.Evaluate(e.Board)

	ml := e.Board.GenerateMoves()
	e.orderMoves(&ml, ttMove)

	legalMoves := 0
	for i := 0; i < ml.Count; i++ {
		move := ml.Moves[i]

		// Futility Pruning: Skip quiet moves at low depth if they can't improve alpha
		if depth <= 3 && !inCheck && i > 0 && move.Flags()&engine.CaptureFlag == 0 && move.Flags()&0x8000 == 0 {
			if standingPat+depth*150 < alpha {
				continue
			}
		}

		// Bad Capture Pruning: skip captures that lose material at low depths
		if depth <= 2 && !inCheck && move.Flags()&engine.CaptureFlag != 0 && e.Board.SEE(move) < 0 {
			continue
		}

		if !e.Board.MakeMove(move) {
			continue
		}

		legalMoves++

		var score int
		if i == 0 {
			// Principal Variation move, search with full window
			score = -e.negamax(depth-1, -beta, -alpha)
		} else {
			// Late Move Reductions (LMR)
			reduction := 0
			if depth >= 3 && i >= 4 && !inCheck && move.Flags()&engine.CaptureFlag == 0 && move.Flags()&0x8000 == 0 {
				reduction = 1
				if i >= 10 {
					reduction = 2
				}
			}

			// Principal Variation Search (PVS) with null window
			score = -e.negamax(depth-1-reduction, -(alpha + 1), -alpha)

			// Re-search if reduced move beats alpha
			if score > alpha && reduction > 0 {
				score = -e.negamax(depth-1, -(alpha + 1), -alpha)
			}

			// Re-search with full window if null window search beats alpha
			if score > alpha && score < beta {
				score = -e.negamax(depth-1, -beta, -alpha)
			}
		}

		e.Board.UnmakeMove(move)

		if e.Stopped {
			return 0
		}

		if score > bestScore {
			bestScore = score
			bestMove = move
		}

		if score >= beta {
			// Store killer moves and history heuristic for quiet moves
			if move.Flags()&engine.CaptureFlag == 0 && e.Board.Ply < 64 {
				if move != e.KillerMoves[e.Board.Ply][0] {
					e.KillerMoves[e.Board.Ply][1] = e.KillerMoves[e.Board.Ply][0]
					e.KillerMoves[e.Board.Ply][0] = move
				}
				e.HistoryTable[e.Board.SideToMove][move.From()][move.To()] += depth * depth
			}

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

	inCheck := e.Board.IsSquareAttacked(e.Board.Pieces[e.Board.SideToMove][engine.King].LSB(), e.Board.SideToMove^1)

	alphaOrig := alpha
	standingPat := -Infinity

	if !inCheck {
		standingPat = eval.Evaluate(e.Board)
		if standingPat >= beta {
			return standingPat
		}
		if standingPat > alpha {
			alpha = standingPat
		}
	}

	var ml engine.MoveList
	if inCheck {
		ml = e.Board.GenerateMoves()
	} else {
		ml = e.Board.GenerateCaptures()
	}

	// Order moves in quiescence search too (captures only)
	e.orderMoves(&ml, engine.NoMove)

	bestScore := standingPat
	legalMoves := 0

	for i := 0; i < ml.Count; i++ {
		move := ml.Moves[i]

		if !inCheck {
			// Delta Pruning
			if move.Flags()&0x8000 == 0 {
				victim := e.Board.PieceAt(move.To()).Type()
				if move.Flags() == engine.EnPassantFlag {
					victim = engine.Pawn
				}

				// Rough piece values for delta pruning
				victimValue := 0
				switch victim {
				case engine.Pawn:
					victimValue = 100
				case engine.Knight:
					victimValue = 300
				case engine.Bishop:
					victimValue = 300
				case engine.Rook:
					victimValue = 500
				case engine.Queen:
					victimValue = 900
				}

				if standingPat+victimValue+200 < alpha {
					continue
				}
			}

			// SEE Pruning: Don't search losing captures in quiescence
			if e.Board.SEE(move) < 0 {
				continue
			}
		}

		if !e.Board.MakeMove(move) {
			continue
		}
		legalMoves++

		score := -e.quiescence(-beta, -alpha)
		e.Board.UnmakeMove(move)

		if score > bestScore {
			bestScore = score
			if score >= beta {
				break
			}
			if score > alpha {
				alpha = score
			}
		}
	}

	if inCheck && legalMoves == 0 {
		bestScore = -MateScore + e.Board.Ply
	}

	// TT Store
	flag := ExactFlag
	if bestScore <= alphaOrig {
		flag = AlphaFlag
	} else if bestScore >= beta {
		flag = BetaFlag
	}
	e.TT.Store(e.Board.Hash, 0, bestScore, flag, engine.NoMove, e.Board.Ply)

	return bestScore
}

func (e *Engine) printInfo(depth, score int, bestMove engine.Move) {
	duration := time.Since(e.StartTime).Seconds()
	nps := uint64(0)
	if duration > 0.001 {
		nps = uint64(float64(e.Nodes) / duration)
	}

	if score > MateScore-1000 {
		mateIn := (MateScore - score + 1) / 2
		fmt.Printf("info depth %d score mate %d nodes %d nps %d time %d pv %s\n",
			depth, mateIn, e.Nodes, nps, int(duration*1000), bestMove.String())
	} else if score < -MateScore+1000 {
		mateIn := (MateScore + score + 1) / 2
		fmt.Printf("info depth %d score mate -%d nodes %d nps %d time %d pv %s\n",
			depth, mateIn, e.Nodes, nps, int(duration*1000), bestMove.String())
	} else {
		fmt.Printf("info depth %d score cp %d nodes %d nps %d time %d pv %s\n",
			depth, score, e.Nodes, nps, int(duration*1000), bestMove.String())
	}
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
func (e *Engine) orderMoves(ml *engine.MoveList, ttMove engine.Move) {
	scores := make([]int, ml.Count)

	for i := 0; i < ml.Count; i++ {
		move := ml.Moves[i]

		// TT move gets highest priority
		if move == ttMove {
			scores[i] = 30000
			continue
		}

		flags := move.Flags()

		if flags&engine.CaptureFlag != 0 {
			attacker := e.Board.PieceAt(move.From()).Type()
			victim := e.Board.PieceAt(move.To()).Type()

			if flags == engine.EnPassantFlag {
				victim = engine.Pawn
			}

			// Prioritize winning/equal captures using SEE
			if e.Board.SEE(move) >= 0 {
				scores[i] = 25000 + mvvLva[victim][attacker]
			} else {
				scores[i] = 11000 + mvvLva[victim][attacker]
			}
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
