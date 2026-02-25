package search

import (
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Skiddle-Labs/axon-engine/internal/engine"
)

const (
	Infinity  = 32000
	MateScore = 31000
)

// GlobalTT is the shared transposition table used by all engine instances.
var GlobalTT = NewTranspositionTable(64)

// Search Parameters (UCI Knobs)
var (
	AspirationDelta = 15
	RFPMargin       = 75
	FPMargin        = 100
	NMPBase         = 3
	NMPDivisor      = 6
)

// Engine represents a search instance.
type Engine struct {
	Board      *engine.Board
	Nodes      *uint64
	StartTime  time.Time
	TimeLimit  time.Duration
	SoftLimit  time.Duration
	NodesLimit uint64
	Stopped    *int32
	TT         *TranspositionTable

	Threads int
	MultiPV int
	Silent  bool

	RootExcludedMoves []engine.Move

	// Time management tracking
	BestMoveChanges int
	LastDepthScore  int
	LastDepthMove   engine.Move

	localNodes   uint64
	KillerMoves  [128][2]engine.Move
	HistoryTable *[2][7][64]int
	CounterMoves *[64][64]engine.Move
}

// NewEngine creates a new search instance for a board.
func NewEngine(b *engine.Board) *Engine {
	nodes := uint64(0)
	stopped := int32(0)
	return &Engine{
		Board:        b,
		Nodes:        &nodes,
		Stopped:      &stopped,
		TT:           GlobalTT,
		Threads:      1,
		MultiPV:      1,
		HistoryTable: &[2][7][64]int{},
		CounterMoves: &[64][64]engine.Move{},
	}
}

// ResetSearchParameters restores all search knobs to their default values.
func ResetSearchParameters() {
	AspirationDelta = 15
	RFPMargin = 75
	FPMargin = 100
	NMPBase = 3
	NMPDivisor = 6
	UpdateLMR(0.75, 2.25)
}

// syncNodes flushes local node counts to the global atomic counter.
func (e *Engine) syncNodes() {
	atomic.AddUint64(e.Nodes, e.localNodes)
	e.localNodes = 0
}

// Search performs Iterative Deepening to find the best move for the current position.
func (e *Engine) Search(maxDepth int) engine.Move {
	atomic.StoreUint64(e.Nodes, 0)
	e.localNodes = 0
	e.StartTime = time.Now()
	atomic.StoreInt32(e.Stopped, 0)

	globalBestMove := engine.NoMove
	lastScore := 0
	e.BestMoveChanges = 0
	e.LastDepthScore = 0
	e.LastDepthMove = engine.NoMove

	// Safe fallback: pick the first legal move as a baseline
	ml := e.Board.GenerateMoves()
	for i := 0; i < ml.Count; i++ {
		if e.Board.MakeMove(ml.Moves[i]) {
			e.Board.UnmakeMove(ml.Moves[i])
			globalBestMove = ml.Moves[i]
			break
		}
	}

	if globalBestMove == engine.NoMove {
		return engine.NoMove
	}

	// Default soft limit to 60% of total time if not set
	if e.SoftLimit == 0 && e.TimeLimit > 0 {
		e.SoftLimit = (e.TimeLimit * 6) / 10
	}

	var wg sync.WaitGroup
	// Lazy SMP helper threads
	for t := 1; t < e.Threads; t++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()
			bCopy := *e.Board
			helper := NewEngine(&bCopy)
			helper.HistoryTable = e.HistoryTable
			helper.CounterMoves = e.CounterMoves
			helper.TT = e.TT
			helper.Nodes = e.Nodes
			helper.Stopped = e.Stopped
			for d := 1; d <= maxDepth; d++ {
				searchDepth := d
				if threadID%2 != 0 {
					searchDepth = d + 1
				}
				helper.negamax(searchDepth, -Infinity, Infinity, 0, engine.NoMove)
				if atomic.LoadInt32(e.Stopped) != 0 {
					break
				}
			}
		}(t)
	}

	defer func() {
		atomic.StoreInt32(e.Stopped, 1)
		wg.Wait()
		e.syncNodes()
	}()

	// Iterative Deepening loop
	for depth := 1; depth <= maxDepth; depth++ {
		// Dynamic Time Management: adjust soft limit based on move stability
		currentSoftLimit := e.SoftLimit
		if depth > 5 && e.BestMoveChanges > 0 && currentSoftLimit > 0 {
			// Extend time if the best move is unstable (frequently changing)
			currentSoftLimit = (currentSoftLimit * 15) / 10
			if currentSoftLimit > e.TimeLimit && e.TimeLimit > 0 {
				currentSoftLimit = e.TimeLimit
			}
		}

		if depth > 1 && currentSoftLimit > 0 && time.Since(e.StartTime) >= currentSoftLimit {
			break
		}

		e.RootExcludedMoves = nil

		for pvIdx := 1; pvIdx <= e.MultiPV; pvIdx++ {
			alpha := -Infinity
			beta := Infinity
			delta := AspirationDelta

			// Aspiration Windows: restrict search bounds around previous score to prune more nodes
			// Only for the first PV; secondary PVs usually search with full window.
			if depth >= 5 && pvIdx == 1 {
				alpha = lastScore - delta
				beta = lastScore + delta
			}

			for {
				score := e.negamax(depth, alpha, beta, 0, engine.NoMove)

				if atomic.LoadInt32(e.Stopped) != 0 {
					break
				}

				if score <= alpha {
					// Fail Low: Score is worse than expected, widen window down
					if e.TimeLimit > 0 && e.SoftLimit < e.TimeLimit {
						e.SoftLimit += e.SoftLimit / 10
					}
					alpha = int(math.Max(float64(alpha-delta), -Infinity))
					beta = (alpha + beta) / 2
					delta += delta/2 + 5
				} else if score >= beta {
					// Fail High: Score is better than expected, widen window up
					if e.TimeLimit > 0 && e.SoftLimit < e.TimeLimit {
						e.SoftLimit += e.SoftLimit / 10
					}
					beta = int(math.Min(float64(beta+delta), Infinity))
					delta += delta/2 + 5
				} else {
					if pvIdx == 1 {
						lastScore = score
					}
					break
				}

				// Revert to full window if it gets too wide
				if delta > 1000 {
					alpha = -Infinity
					beta = Infinity
				}
			}

			if atomic.LoadInt32(e.Stopped) != 0 {
				break
			}

			// Update best move and check stability from the Transposition Table
			_, ttMove, found := e.TT.Probe(e.Board.Hash, depth, -Infinity, Infinity, 0)
			if found && ttMove != engine.NoMove {
				if pvIdx == 1 {
					if ttMove != e.LastDepthMove {
						if e.LastDepthMove != engine.NoMove {
							e.BestMoveChanges++
						}
						e.LastDepthMove = ttMove
					}

					// If score dropped significantly compared to last depth, extend search time
					if depth > 5 && lastScore < e.LastDepthScore-20 {
						if e.TimeLimit > 0 && e.SoftLimit < e.TimeLimit {
							e.SoftLimit += e.SoftLimit / 10
						}
					}
					e.LastDepthScore = lastScore
					globalBestMove = ttMove
				}

				e.printInfo(depth, lastScore, ttMove, pvIdx)
				e.RootExcludedMoves = append(e.RootExcludedMoves, ttMove)
			} else {
				break
			}
		}

		// Terminate if a mate is found in the primary line
		if lastScore > MateScore-500 || lastScore < -MateScore+500 {
			break
		}
	}

	return globalBestMove
}

// getPV extracts the Principal Variation string from the Transposition Table.
func (e *Engine) getPV(depth int) string {
	var pv []engine.Move
	tempBoard := *e.Board

	for i := 0; i < depth; i++ {
		_, move, found := e.TT.Probe(tempBoard.Hash, 0, -Infinity, Infinity, 0)
		if !found || move == engine.NoMove {
			break
		}

		legal := false
		ml := tempBoard.GenerateMoves()
		for j := 0; j < ml.Count; j++ {
			if ml.Moves[j] == move {
				if tempBoard.MakeMove(move) {
					legal = true
					break
				}
			}
		}

		if !legal {
			break
		}
		pv = append(pv, move)
	}

	res := ""
	for i, m := range pv {
		if i > 0 {
			res += " "
		}
		res += m.String()
	}
	return res
}

// printInfo outputs search information in UCI format.
func (e *Engine) printInfo(depth, score int, bestMove engine.Move, multipv int) {
	e.syncNodes()
	if e.Silent {
		return
	}
	duration := time.Since(e.StartTime).Seconds()
	nps := uint64(0)
	nodes := atomic.LoadUint64(e.Nodes)
	if duration > 0.001 {
		nps = uint64(float64(nodes) / duration)
	}

	hashfull := e.TT.HashFull()
	pvStr := e.getPV(depth)
	if pvStr == "" && bestMove != engine.NoMove {
		pvStr = bestMove.String()
	}

	if score > MateScore-500 {
		mateIn := (MateScore - score + 1) / 2
		fmt.Printf("info depth %d multipv %d score mate %d nodes %d nps %d hashfull %d time %d pv %s\n",
			depth, multipv, mateIn, nodes, nps, hashfull, int(duration*1000), pvStr)
	} else if score < -MateScore+500 {
		mateIn := (MateScore + score + 1) / 2
		fmt.Printf("info depth %d multipv %d score mate -%d nodes %d nps %d hashfull %d time %d pv %s\n",
			depth, multipv, mateIn, nodes, nps, hashfull, int(duration*1000), pvStr)
	} else {
		fmt.Printf("info depth %d multipv %d score cp %d nodes %d nps %d hashfull %d time %d pv %s\n",
			depth, multipv, score, nodes, nps, hashfull, int(duration*1000), pvStr)
	}
}
