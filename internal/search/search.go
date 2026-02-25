package search

import (
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Skiddle-Labs/axon-engine/internal/engine"
	"github.com/Skiddle-Labs/axon-engine/internal/eval"
)

const (
	Infinity  = 32000
	MateScore = 31000
)

var GlobalTT = NewTranspositionTable(64)

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

	// Time management tracking
	BestMoveChanges int
	LastDepthScore  int
	LastDepthMove   engine.Move

	localNodes   uint64
	KillerMoves  [128][2]engine.Move
	HistoryTable *[2][7][64]int
	CounterMoves *[64][64]engine.Move
}

func (e *Engine) syncNodes() {
	atomic.AddUint64(e.Nodes, e.localNodes)
	e.localNodes = 0
}

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

	// Safe fallback: pick the first legal move
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

	for depth := 1; depth <= maxDepth; depth++ {
		// Dynamic Time Management: adjust soft limit based on move stability
		currentSoftLimit := e.SoftLimit
		if depth > 5 && e.BestMoveChanges > 0 && currentSoftLimit > 0 {
			// Extend time if the best move is unstable
			currentSoftLimit = (currentSoftLimit * 15) / 10
			if currentSoftLimit > e.TimeLimit && e.TimeLimit > 0 {
				currentSoftLimit = e.TimeLimit
			}
		}

		if depth > 1 && currentSoftLimit > 0 && time.Since(e.StartTime) >= currentSoftLimit {
			break
		}

		alpha := -Infinity
		beta := Infinity
		delta := 15

		// Aspiration Windows
		if depth >= 5 {
			alpha = lastScore - delta
			beta = lastScore + delta
		}

		for {
			score := e.negamax(depth, alpha, beta, 0, engine.NoMove)

			if atomic.LoadInt32(e.Stopped) != 0 {
				break
			}

			if score <= alpha {
				// Fail Low: Extend time slightly to resolve the drop
				if e.TimeLimit > 0 && e.SoftLimit < e.TimeLimit {
					e.SoftLimit += e.SoftLimit / 10
				}
				alpha = int(math.Max(float64(alpha-delta), -Infinity))
				beta = (alpha + beta) / 2
				delta += delta/2 + 5
			} else if score >= beta {
				// Fail High: Extend time to find the best continuation
				if e.TimeLimit > 0 && e.SoftLimit < e.TimeLimit {
					e.SoftLimit += e.SoftLimit / 10
				}
				beta = int(math.Min(float64(beta+delta), Infinity))
				delta += delta/2 + 5
			} else {
				lastScore = score
				break
			}

			if delta > 1000 {
				alpha = -Infinity
				beta = Infinity
			}
		}

		if atomic.LoadInt32(e.Stopped) != 0 {
			break
		}

		// Update best move from TT
		_, ttMove, found := e.TT.Probe(e.Board.Hash, depth, -Infinity, Infinity, 0)
		if found && ttMove != engine.NoMove {
			if ttMove != e.LastDepthMove {
				if e.LastDepthMove != engine.NoMove {
					e.BestMoveChanges++
				}
				e.LastDepthMove = ttMove
			}

			// Score stability: if score dropped significantly, think more
			if depth > 5 && lastScore < e.LastDepthScore-20 {
				if e.TimeLimit > 0 && e.SoftLimit < e.TimeLimit {
					e.SoftLimit += e.SoftLimit / 10
				}
			}
			e.LastDepthScore = lastScore

			globalBestMove = ttMove
		}

		e.printInfo(depth, lastScore, globalBestMove, 1)

		if lastScore > MateScore-500 || lastScore < -MateScore+500 {
			break
		}
	}

	return globalBestMove
}

func (e *Engine) negamax(depth, alpha, beta, ply int, excludedMove engine.Move) int {
	e.localNodes++
	if e.localNodes >= 2048 {
		e.syncNodes()
		nodes := atomic.LoadUint64(e.Nodes)
		if (e.TimeLimit > 0 && time.Since(e.StartTime) >= e.TimeLimit) ||
			(e.NodesLimit > 0 && nodes >= e.NodesLimit) {
			atomic.StoreInt32(e.Stopped, 1)
		}
	}

	if atomic.LoadInt32(e.Stopped) != 0 {
		return 0
	}

	if ply > 0 {
		// Repetition detection
		for i := e.Board.Ply - 2; i >= e.Board.Ply-int(e.Board.HalfMoveClock); i -= 2 {
			if i >= 0 && e.Board.History[i].Hash == e.Board.Hash {
				return 0
			}
		}
	}

	// TT Probe
	ttScore, ttMove, found := e.TT.Probe(e.Board.Hash, depth, alpha, beta, ply)
	if found && excludedMove == engine.NoMove {
		return ttScore
	}

	inCheck := e.Board.IsSquareAttacked(e.Board.Pieces[e.Board.SideToMove][engine.King].LSB(), e.Board.SideToMove^1)
	if inCheck {
		depth++
	}

	if depth <= 0 {
		return e.quiescence(alpha, beta, ply)
	}

	staticEval := eval.Evaluate(e.Board)

	// Reverse Futility Pruning (RFP)
	if depth < 5 && !inCheck && excludedMove == engine.NoMove && ply > 0 && beta < MateScore-1000 {
		margin := depth * 75
		if staticEval-margin >= beta {
			return beta
		}
	}

	// Internal Iterative Reductions (IIR)
	if depth >= 3 && ttMove == engine.NoMove && !inCheck && ply > 0 {
		depth--
	}

	// Singular Extensions
	extension := 0
	if depth >= 8 && ttMove != engine.NoMove && excludedMove == engine.NoMove && found {
		singularBeta := ttScore - 2*depth
		score := e.negamax((depth-1)/2, singularBeta-1, singularBeta, ply, ttMove)
		if score < singularBeta {
			extension = 1
		}
	}

	// Null Move Pruning
	if depth >= 3 && !inCheck && ply > 0 && excludedMove == engine.NoMove && e.Board.HasMajorPieces(e.Board.SideToMove) {
		e.Board.MakeNullMove()
		score := -e.negamax(depth-3, -beta, -beta+1, ply+1, engine.NoMove)
		e.Board.UnmakeNullMove()
		if score >= beta {
			return beta
		}
	}

	ml := e.Board.GenerateMoves()
	e.orderMoves(&ml, ttMove, ply)

	alphaOrig := alpha
	bestMove := engine.NoMove
	bestScore := -Infinity
	legalMoves := 0

	for i := 0; i < ml.Count; i++ {
		move := ml.Moves[i]
		if move == excludedMove {
			continue
		}

		if !e.Board.MakeMove(move) {
			continue
		}

		// Late Move Pruning (LMP)
		if depth <= 4 && !inCheck && legalMoves > (5+depth*depth) &&
			(move.Flags()&(engine.CaptureFlag|0x8000) == 0) {

			them := e.Board.SideToMove
			kingSq := e.Board.Pieces[them][engine.King].LSB()
			if !e.Board.IsSquareAttacked(kingSq, them^1) {
				e.Board.UnmakeMove(move)
				continue
			}
		}

		legalMoves++

		var score int
		if legalMoves == 1 {
			score = -e.negamax(depth-1+extension, -beta, -alpha, ply+1, engine.NoMove)
		} else {
			reduction := 0
			if depth >= 3 && legalMoves > 4 && !inCheck && move.Flags()&engine.CaptureFlag == 0 && move.Flags()&0x8000 == 0 {
				reduction = 1
				if legalMoves > 12 {
					reduction++
				}
			}

			score = -e.negamax(depth-1-reduction, -(alpha + 1), -alpha, ply+1, engine.NoMove)
			if score > alpha && reduction > 0 {
				score = -e.negamax(depth-1, -(alpha + 1), -alpha, ply+1, engine.NoMove)
			}
			if score > alpha && score < beta {
				score = -e.negamax(depth-1, -beta, -alpha, ply+1, engine.NoMove)
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

		if score > alpha {
			alpha = score
		}

		if score >= beta {
			// Update heuristics on fail-high
			if move.Flags()&engine.CaptureFlag == 0 && ply < 128 {
				if move != e.KillerMoves[ply][0] {
					e.KillerMoves[ply][1] = e.KillerMoves[ply][0]
					e.KillerMoves[ply][0] = move
				}
				piece := e.Board.PieceAt(move.From()).Type()
				if e.HistoryTable != nil {
					e.HistoryTable[e.Board.SideToMove][piece][move.To()] += depth * depth
				}

				if e.Board.Ply > 0 && e.CounterMoves != nil {
					prevMove := e.Board.History[e.Board.Ply-1].Move
					e.CounterMoves[prevMove.From()][prevMove.To()] = move
				}
			}
			break
		}
	}

	if legalMoves == 0 {
		if inCheck {
			return -MateScore + ply
		}
		return 0
	}

	if excludedMove == engine.NoMove {
		flag := ExactFlag
		if bestScore <= alphaOrig {
			flag = AlphaFlag
		} else if bestScore >= beta {
			flag = BetaFlag
		}
		e.TT.Store(e.Board.Hash, depth, bestScore, flag, bestMove, ply)
	}

	return bestScore
}

func (e *Engine) quiescence(alpha, beta, ply int) int {
	e.localNodes++
	if e.localNodes >= 2048 {
		e.syncNodes()
	}

	if atomic.LoadInt32(e.Stopped) != 0 {
		return 0
	}

	standingPat := eval.Evaluate(e.Board)
	if standingPat >= beta {
		return beta
	}
	if standingPat > alpha {
		alpha = standingPat
	}

	ml := e.Board.GenerateCaptures()
	e.orderMoves(&ml, engine.NoMove, ply)

	for i := 0; i < ml.Count; i++ {
		move := ml.Moves[i]
		if e.Board.SEE(move) < 0 {
			continue
		}

		if !e.Board.MakeMove(move) {
			continue
		}
		score := -e.quiescence(-beta, -alpha, ply+1)
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

func (e *Engine) printInfo(depth, score int, bestMove engine.Move, multipv int) {
	e.syncNodes()
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
		fmt.Printf("info depth %d score mate %d nodes %d nps %d hashfull %d time %d pv %s\n",
			depth, mateIn, nodes, nps, hashfull, int(duration*1000), pvStr)
	} else if score < -MateScore+500 {
		mateIn := (MateScore + score + 1) / 2
		fmt.Printf("info depth %d score mate -%d nodes %d nps %d hashfull %d time %d pv %s\n",
			depth, mateIn, nodes, nps, hashfull, int(duration*1000), pvStr)
	} else {
		fmt.Printf("info depth %d score cp %d nodes %d nps %d hashfull %d time %d pv %s\n",
			depth, score, nodes, nps, hashfull, int(duration*1000), pvStr)
	}
}

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
				victim = engine.Pawn
			}
			if e.Board.SEE(move) >= 0 {
				scores[i] = 25000 + mvvLva[victim][piece]
			} else {
				scores[i] = 10000 + mvvLva[victim][piece]
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
