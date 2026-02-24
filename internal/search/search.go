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

// Engine handles the search process.
type Engine struct {
	Board     *board.Board
	Nodes     uint64
	StartTime time.Time
}

// NewEngine creates a new search engine instance.
func NewEngine(b *board.Board) *Engine {
	return &Engine{Board: b}
}

// Search finds the best move for the current position using iterative deepening.
func (e *Engine) Search(maxDepth int) board.Move {
	e.Nodes = 0
	e.StartTime = time.Now()
	globalBestMove := board.NoMove

	for depth := 1; depth <= maxDepth; depth++ {
		alpha := -Infinity
		beta := Infinity
		bestMove := board.NoMove

		ml := e.Board.GenerateMoves()
		e.orderMoves(&ml)

		for i := 0; i < ml.Count; i++ {
			move := ml.Moves[i]
			if !e.Board.MakeMove(move) {
				continue
			}

			score := -e.negamax(depth-1, -beta, -alpha)
			e.Board.UnmakeMove(move)

			if score > alpha {
				alpha = score
				bestMove = move
			}
		}

		if bestMove != board.NoMove {
			globalBestMove = bestMove
			duration := time.Since(e.StartTime).Seconds()
			nps := uint64(0)
			if duration > 0.001 {
				nps = uint64(float64(e.Nodes) / duration)
			}

			fmt.Printf("info depth %d score cp %d nodes %d nps %d time %d pv %s\n",
				depth, alpha, e.Nodes, nps, int(duration*1000), globalBestMove.String())
		}
	}

	return globalBestMove
}

// negamax is the core search algorithm with alpha-beta pruning.
func (e *Engine) negamax(depth, alpha, beta int) int {
	e.Nodes++

	// Base case: reach depth 0, enter quiescence search to avoid horizon effect.
	if depth <= 0 {
		return e.quiescence(alpha, beta)
	}

	ml := e.Board.GenerateMoves()
	e.orderMoves(&ml)

	legalMoves := 0
	for i := 0; i < ml.Count; i++ {
		move := ml.Moves[i]
		if !e.Board.MakeMove(move) {
			continue
		}

		legalMoves++
		score := -e.negamax(depth-1, -beta, -alpha)
		e.Board.UnmakeMove(move)

		if score >= beta {
			return beta // Fail-high, beta cutoff
		}
		if score > alpha {
			alpha = score
		}
	}

	// Handle terminal nodes (Checkmate/Stalemate)
	if legalMoves == 0 {
		kingSq := e.Board.Pieces[e.Board.SideToMove][board.King].LSB()
		if e.Board.IsSquareAttacked(kingSq, e.Board.SideToMove^1) {
			// Checkmate: return mate score adjusted by distance from root (ply)
			return -MateScore + e.Board.Ply
		}
		// Stalemate
		return 0
	}

	return alpha
}

// quiescence search evaluates only "noisy" positions (captures) to stabilize the evaluation.
func (e *Engine) quiescence(alpha, beta int) int {
	e.Nodes++
	standingPat := eval.Evaluate(e.Board)

	if standingPat >= beta {
		return beta
	}
	if standingPat > alpha {
		alpha = standingPat
	}

	ml := e.Board.GenerateMoves()
	e.orderMoves(&ml)

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
func (e *Engine) orderMoves(ml *board.MoveList) {
	scores := make([]int, ml.Count)

	for i := 0; i < ml.Count; i++ {
		move := ml.Moves[i]
		flags := move.Flags()

		if flags&board.CaptureFlag != 0 {
			attacker := e.Board.PieceAt(move.From()).Type()
			victim := e.Board.PieceAt(move.To()).Type()

			if flags == board.EnPassantFlag {
				victim = board.Pawn
			}

			// Prioritize captures using MVV-LVA
			scores[i] = 10000 + mvvLva[victim][attacker]
		} else if flags&0x8000 != 0 {
			// Prioritize promotions
			scores[i] = 9000
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
