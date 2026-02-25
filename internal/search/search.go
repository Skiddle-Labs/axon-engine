package search

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Skiddle-Labs/axon-engine/internal/engine"
	"github.com/Skiddle-Labs/axon-engine/internal/eval"
)

const (
	Infinity  = 30000
	MateScore = 20000
)

var GlobalTT = NewTranspositionTable(64)

// Engine handles the search process.
type Engine struct {
	Board      *engine.Board
	Nodes      *uint64
	StartTime  time.Time
	TimeLimit  time.Duration // Hard limit: stop immediately
	SoftLimit  time.Duration // Soft limit: don't start new depth
	NodesLimit uint64
	Stopped    *int32
	TT         *TranspositionTable

	Threads int
	MultiPV int

	localNodes   uint64
	KillerMoves  [64][2]engine.Move
	HistoryTable [2][64][64]int
	CounterMoves [64][64]engine.Move
}

func (e *Engine) syncNodes() {
	if e.localNodes > 0 {
		atomic.AddUint64(e.Nodes, e.localNodes)
		e.localNodes = 0
	}
}

// NewEngine creates a new search engine instance.
func NewEngine(b *engine.Board) *Engine {
	nodes := uint64(0)
	stopped := int32(0)
	return &Engine{
		Board:   b,
		TT:      GlobalTT,
		Nodes:   &nodes,
		Stopped: &stopped,
		Threads: 1,
		MultiPV: 1,
	}
}

// Search finds the best move for the current position using iterative deepening.
func (e *Engine) Search(maxDepth int) engine.Move {
	atomic.StoreUint64(e.Nodes, 0)
	e.localNodes = 0
	e.StartTime = time.Now()
	atomic.StoreInt32(e.Stopped, 0)

	globalBestMove := engine.NoMove
	lastBestMove := engine.NoMove
	lastScore := 0
	stability := 0

	// Check for legal moves to avoid searching in terminal positions
	ml := e.Board.GenerateMoves()
	legalCount := 0
	for i := 0; i < ml.Count; i++ {
		if e.Board.MakeMove(ml.Moves[i]) {
			e.Board.UnmakeMove(ml.Moves[i])
			legalCount++
			break
		}
	}

	if legalCount == 0 {
		return engine.NoMove
	}

	// If SoftLimit is not explicitly set, use 60% of TimeLimit as a default.
	if e.SoftLimit == 0 && e.TimeLimit > 0 {
		e.SoftLimit = (e.TimeLimit * 6) / 10
	}

	// Launch helper threads for Lazy SMP
	var wg sync.WaitGroup
	for t := 1; t < e.Threads; t++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()
			bCopy := *e.Board
			helper := NewEngine(&bCopy)
			helper.TT = e.TT
			helper.Nodes = e.Nodes
			helper.Stopped = e.Stopped

			for d := 1; d <= maxDepth+2; d++ {
				if atomic.LoadInt32(e.Stopped) != 0 {
					break
				}

				depth := d
				if threadID%2 != 0 {
					depth = d + 1
				} else if threadID > 2 {
					depth = d - 1
				}

				if depth < 1 {
					depth = 1
				}

				helper.negamax(depth, -Infinity, Infinity, engine.NoMove)
			}
			helper.syncNodes()
		}(t)
	}

	defer func() {
		atomic.StoreInt32(e.Stopped, 1)
		wg.Wait()
		e.syncNodes()
	}()

	for depth := 1; depth <= maxDepth; depth++ {
		adjustedSoftLimit := e.SoftLimit
		if depth > 5 && stability >= 3 {
			adjustedSoftLimit /= 2
		}

		if depth > 1 && adjustedSoftLimit > 0 && time.Since(e.StartTime) >= adjustedSoftLimit {
			break
		}

		multiPVBestMoves := make([]engine.Move, 0, e.MultiPV)

		for pv := 1; pv <= e.MultiPV; pv++ {
			alpha := -Infinity
			beta := Infinity
			delta := 15

			if depth > 4 && pv == 1 {
				alpha = lastScore - delta
				beta = lastScore + delta
			}

			bestMoveAtDepth := engine.NoMove
			scoreAtDepth := -Infinity

			for {
				if alpha < -MateScore {
					alpha = -Infinity
				}
				if beta > MateScore {
					beta = Infinity
				}

				scoreAtDepth = -Infinity
				bestMoveAtDepth = engine.NoMove

				ml := e.Board.GenerateMoves()
				_, ttMove, _ := e.TT.Probe(e.Board.Hash, depth, alpha, beta, e.Board.Ply)
				e.orderMoves(&ml, ttMove)

				currAlpha := alpha
				legalMoves := 0
				for i := 0; i < ml.Count; i++ {
					move := ml.Moves[i]

					skip := false
					for _, m := range multiPVBestMoves {
						if move == m {
							skip = true
							break
						}
					}
					if skip {
						continue
					}

					if !e.Board.MakeMove(move) {
						continue
					}
					legalMoves++

					var s int
					if legalMoves == 1 {
						s = -e.negamax(depth-1, -beta, -currAlpha, engine.NoMove)
					} else {
						reduction := 0
						if depth >= 3 && i >= 6 && move.Flags()&engine.CaptureFlag == 0 && move.Flags()&0x8000 == 0 {
							reduction = 1
						}

						s = -e.negamax(depth-1-reduction, -(currAlpha + 1), -currAlpha, engine.NoMove)

						if s > currAlpha && reduction > 0 {
							s = -e.negamax(depth-1, -(currAlpha + 1), -currAlpha, engine.NoMove)
						}
						if s > currAlpha && s < beta {
							s = -e.negamax(depth-1, -beta, -currAlpha, engine.NoMove)
						}
					}

					e.Board.UnmakeMove(move)

					if atomic.LoadInt32(e.Stopped) != 0 {
						break
					}

					if s > scoreAtDepth {
						scoreAtDepth = s
						bestMoveAtDepth = move
					}
					if s > currAlpha {
						currAlpha = s
					}
				}

				if atomic.LoadInt32(e.Stopped) != 0 {
					break
				}

				if scoreAtDepth <= alpha {
					alpha -= delta
					delta *= 2
				} else if scoreAtDepth >= beta {
					beta += delta
					delta *= 2
				} else {
					if pv == 1 {
						lastScore = scoreAtDepth
						if bestMoveAtDepth != engine.NoMove {
							if bestMoveAtDepth == lastBestMove {
								stability++
							} else {
								stability = 0
								lastBestMove = bestMoveAtDepth
							}
							globalBestMove = bestMoveAtDepth
						}
					}
					if bestMoveAtDepth != engine.NoMove {
						multiPVBestMoves = append(multiPVBestMoves, bestMoveAtDepth)
						e.printInfo(depth, scoreAtDepth, bestMoveAtDepth, pv)
					}
					break
				}
			}
		}

		if atomic.LoadInt32(e.Stopped) != 0 {
			break
		}
	}

	return globalBestMove
}

// negamax is the core search algorithm with alpha-beta pruning.
func (e *Engine) negamax(depth, alpha, beta int, excludedMove engine.Move) int {
	e.localNodes++
	if e.localNodes >= 2048 {
		nodes := atomic.AddUint64(e.Nodes, e.localNodes)
		e.localNodes = 0
		if (e.TimeLimit > 0 && time.Since(e.StartTime) >= e.TimeLimit) ||
			(e.NodesLimit > 0 && nodes >= e.NodesLimit) {
			atomic.StoreInt32(e.Stopped, 1)
		}
	}

	if atomic.LoadInt32(e.Stopped) != 0 {
		return 0
	}

	// 2. TT Probe
	ttScore, ttMove, found := e.TT.Probe(e.Board.Hash, depth, alpha, beta, e.Board.Ply)
	if found && excludedMove == engine.NoMove {
		return ttScore
	}

	// 3. Repetition Detection
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

	// Singular Extensions
	// If we have a very strong move from the TT, verify that no other move is nearly as good.
	extension := 0
	if depth >= 8 && ttMove != engine.NoMove && excludedMove == engine.NoMove {
		// We use a lower depth to verify singularity
		_, _, ttFound := e.TT.Probe(e.Board.Hash, depth, alpha, beta, e.Board.Ply)
		if ttFound {
			// Search all moves except the TT move with a reduced window
			singularBeta := ttScore - 2*depth
			singularDepth := (depth - 1) / 2

			score := e.negamax(singularDepth, singularBeta-1, singularBeta, ttMove)

			if score < singularBeta {
				extension = 1
			}
		}
	}

	// Static Null Move Pruning (Reverse Futility Pruning)
	// If static evaluation is significantly above beta, we can prune the node.
	if depth <= 5 && !inCheck {
		staticEval := eval.Evaluate(e.Board)
		margin := 120 * depth
		if staticEval-margin >= beta {
			return staticEval - margin
		}
	}

	// Null Move Pruning (NMP)
	// If the current side has major pieces and is not in check, try skipping a turn.
	// If the resulting score is still >= beta, we assume the position is strong enough to fail-high.
	if depth >= 3 && !inCheck && e.Board.HasMajorPieces(e.Board.SideToMove) {
		e.Board.MakeNullMove()
		score := -e.negamax(depth-3, -beta, -beta+1, engine.NoMove)
		e.Board.UnmakeNullMove()

		if score >= beta {
			return beta
		}
	}

	// Internal Iterative Deepening (IID)
	// If we don't have a best move from the TT, perform a shallow search to find one.
	if depth >= 5 && ttMove == engine.NoMove {
		e.negamax(depth-2, alpha, beta, engine.NoMove)
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

		if move == excludedMove {
			continue
		}

		// Futility Pruning: Skip quiet moves at low depth if they can't improve alpha
		if depth <= 3 && !inCheck && i > 0 && move.Flags()&engine.CaptureFlag == 0 && move.Flags()&0x8000 == 0 {
			if standingPat+depth*150 < alpha {
				continue
			}
		}

		// Late Move Pruning (LMP): Skip quiet moves if we've searched enough at low depth
		if depth <= 4 && !inCheck && i >= (4+2*depth*depth) && move.Flags()&engine.CaptureFlag == 0 && move.Flags()&0x8000 == 0 {
			continue
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
			score = -e.negamax(depth-1+extension, -beta, -alpha, engine.NoMove)
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
			score = -e.negamax(depth-1-reduction, -(alpha + 1), -alpha, engine.NoMove)

			// Re-search if reduced move beats alpha
			if score > alpha && reduction > 0 {
				score = -e.negamax(depth-1, -(alpha + 1), -alpha, engine.NoMove)
			}

			// Re-search with full window if null window search beats alpha
			if score > alpha && score < beta {
				score = -e.negamax(depth-1, -beta, -alpha, engine.NoMove)
			}
		}

		e.Board.UnmakeMove(move)

		if atomic.LoadInt32(e.Stopped) != 0 {
			return 0
		}

		if score > bestScore {
			bestScore = score
			bestMove = move
		}

		if score >= beta {
			if atomic.LoadInt32(e.Stopped) != 0 {
				return 0
			}
			// Store killer moves, history heuristic, and countermoves for quiet moves
			if move.Flags()&engine.CaptureFlag == 0 && e.Board.Ply < 64 {
				if move != e.KillerMoves[e.Board.Ply][0] {
					e.KillerMoves[e.Board.Ply][1] = e.KillerMoves[e.Board.Ply][0]
					e.KillerMoves[e.Board.Ply][0] = move
				}
				e.HistoryTable[e.Board.SideToMove][move.From()][move.To()] += depth * depth

				if e.Board.Ply > 0 {
					prevMove := e.Board.History[e.Board.Ply-1].Move
					e.CounterMoves[prevMove.From()][prevMove.To()] = move
				}
			}

			break // Fail-high, beta cutoff
		}
		if score > alpha {
			alpha = score
		}
	}

	if atomic.LoadInt32(e.Stopped) != 0 {
		return 0
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
	e.localNodes++
	if e.localNodes >= 2048 {
		nodes := atomic.AddUint64(e.Nodes, e.localNodes)
		e.localNodes = 0
		if (e.TimeLimit > 0 && time.Since(e.StartTime) >= e.TimeLimit) ||
			(e.NodesLimit > 0 && nodes >= e.NodesLimit) {
			atomic.StoreInt32(e.Stopped, 1)
		}
	}

	if atomic.LoadInt32(e.Stopped) != 0 {
		return 0
	}

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

		if atomic.LoadInt32(e.Stopped) != 0 {
			return 0
		}

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

func (e *Engine) getPV(depth int) string {
	var pv []string
	tempBoard := *e.Board
	// Extract PV from Transposition Table
	for i := 0; i < depth && i < 100; i++ {
		_, move, found := e.TT.Probe(tempBoard.Hash, 0, -Infinity, Infinity, tempBoard.Ply)
		if !found || move == engine.NoMove {
			break
		}

		// Verify legality before adding to PV
		ml := tempBoard.GenerateMoves()
		legal := false
		for j := 0; j < ml.Count; j++ {
			if ml.Moves[j] == move {
				if tempBoard.MakeMove(move) {
					legal = true
				}
				break
			}
		}

		if !legal {
			break
		}
		pv = append(pv, move.String())
	}
	return strings.Join(pv, " ")
}

func (e *Engine) printInfo(depth, score int, bestMove engine.Move, multipv int) {
	e.syncNodes()
	duration := time.Since(e.StartTime).Seconds()
	nps := uint64(0)
	nodes := atomic.LoadUint64(e.Nodes)
	if duration > 0.001 {
		nps = uint64(float64(nodes) / duration)
	}

	multipvStr := ""
	if e.MultiPV > 1 {
		multipvStr = fmt.Sprintf(" multipv %d", multipv)
	}

	hashfull := e.TT.HashFull()
	pvStr := e.getPV(depth)
	if pvStr == "" && bestMove != engine.NoMove {
		pvStr = bestMove.String()
	}

	if score > MateScore-1000 {
		mateIn := (MateScore - score + 1) / 2
		fmt.Printf("info depth %d%s score mate %d nodes %d nps %d hashfull %d time %d pv %s\n",
			depth, multipvStr, mateIn, nodes, nps, hashfull, int(duration*1000), pvStr)
	} else if score < -MateScore+1000 {
		mateIn := (MateScore + score + 1) / 2
		fmt.Printf("info depth %d%s score mate -%d nodes %d nps %d hashfull %d time %d pv %s\n",
			depth, multipvStr, mateIn, nodes, nps, hashfull, int(duration*1000), pvStr)
	} else {
		fmt.Printf("info depth %d%s score cp %d nodes %d nps %d hashfull %d time %d pv %s\n",
			depth, multipvStr, score, nodes, nps, hashfull, int(duration*1000), pvStr)
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
	var scores [256]int

	counterMove := engine.NoMove
	if e.Board.Ply > 0 {
		prevMove := e.Board.History[e.Board.Ply-1].Move
		counterMove = e.CounterMoves[prevMove.From()][prevMove.To()]
	}

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
		} else if move == counterMove {
			scores[i] = 8500
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
