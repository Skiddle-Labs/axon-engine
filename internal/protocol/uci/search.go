package uci

import (
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/Skiddle-Labs/axon-engine/internal/engine"
	"github.com/Skiddle-Labs/axon-engine/internal/eval"
	"github.com/Skiddle-Labs/axon-engine/internal/search"
	"github.com/Skiddle-Labs/axon-engine/internal/types"
)

func (u *UCI) handleGo(parts []string) {
	// 1. Check book
	if u.board.Ply <= u.bookDepth && u.book != nil {
		if move, ok := u.book.GetMove(u.board); ok {
			u.send(fmt.Sprintf("info depth 1 score cp 0 nodes 0 pv %s", move.String()))
			u.send(fmt.Sprintf("bestmove %s", move.String()))
			return
		}
	}

	// 2. Optimization: Instant move if only one legal move is available and not pondering
	isPonder := false
	for _, part := range parts {
		if part == "ponder" {
			isPonder = true
			break
		}
	}

	ml := u.board.GenerateMoves()
	legalCount := 0
	var lastLegal engine.Move
	for i := 0; i < ml.Count; i++ {
		if u.board.MakeMove(ml.Moves[i]) {
			u.board.UnmakeMove(ml.Moves[i])
			legalCount++
			lastLegal = ml.Moves[i]
		}
	}

	if legalCount == 1 && !isPonder {
		score := eval.Evaluate(u.board)
		u.send(fmt.Sprintf("info depth 1 score cp %d nodes 0 pv %s", score, lastLegal.String()))
		u.send(fmt.Sprintf("bestmove %s", lastLegal.String()))
		return
	}

	// 3. Stop previous search
	if u.engine != nil {
		atomic.StoreInt32(u.engine.Stopped, 1)
	}

	// 4. Setup engine instance
	u.engine = search.NewEngine(u.board)
	u.engine.Threads = u.threads
	u.engine.MultiPV = u.multiPV
	u.engine.ShowWDL = u.showWDL
	u.engine.HistoryTable = &u.historyTable
	u.engine.CounterMoves = &u.counterMoves
	u.isPondering = false
	u.pendingTimeLimit = 0

	// 5. Parse 'go' arguments
	depth := 128
	var timeLimit time.Duration
	wtime, btime := -1, -1
	winc, binc := 0, 0
	movestogo := 0
	movetime := -1
	nodesLimit := uint64(0)

	for i := 1; i < len(parts); i++ {
		switch parts[i] {
		case "ponder":
			u.isPondering = true
		case "depth":
			if i+1 < len(parts) {
				depth, _ = strconv.Atoi(parts[i+1])
				i++
			}
		case "wtime":
			if i+1 < len(parts) {
				wtime, _ = strconv.Atoi(parts[i+1])
				i++
			}
		case "btime":
			if i+1 < len(parts) {
				btime, _ = strconv.Atoi(parts[i+1])
				i++
			}
		case "winc":
			if i+1 < len(parts) {
				winc, _ = strconv.Atoi(parts[i+1])
				i++
			}
		case "binc":
			if i+1 < len(parts) {
				binc, _ = strconv.Atoi(parts[i+1])
				i++
			}
		case "movestogo":
			if i+1 < len(parts) {
				movestogo, _ = strconv.Atoi(parts[i+1])
				i++
			}
		case "movetime":
			if i+1 < len(parts) {
				movetime, _ = strconv.Atoi(parts[i+1])
				i++
			}
		case "nodes":
			if i+1 < len(parts) {
				n, _ := strconv.ParseUint(parts[i+1], 10, 64)
				nodesLimit = n
				i++
			}
		case "infinite":
			depth = 128
		}
	}

	// 6. Time Management
	if movetime > 0 {
		timeLimit = time.Duration(movetime) * time.Millisecond
	} else {
		myTime := wtime
		myInc := winc
		if u.board.SideToMove == types.Black {
			myTime = btime
			myInc = binc
		}

		if myTime >= 0 {
			mtg := movestogo
			if mtg <= 0 {
				occCount := u.board.Occupancy().Count()
				mtg = 50 - occCount
				if mtg < 20 {
					mtg = 20
				}
			}
			timeLimit = time.Duration(myTime/mtg)*time.Millisecond + time.Duration(myInc)*8/10*time.Millisecond
		}
	}

	if timeLimit > 0 {
		timeLimit = (timeLimit * time.Duration(u.slowMover)) / 100
		timeLimit -= time.Duration(u.moveOverhead) * time.Millisecond
		if timeLimit < 1*time.Millisecond {
			timeLimit = 1 * time.Millisecond
		}

		if !u.isPondering {
			u.engine.TimeLimit = timeLimit
			u.engine.SoftLimit = (timeLimit * 6) / 10
		} else {
			u.pendingTimeLimit = timeLimit
		}
	}

	u.engine.NodesLimit = nodesLimit

	// 7. Launch Search in a background goroutine
	go func(e *search.Engine, d int) {
		bestMove := e.Search(d)

		// If we are in ponder mode, wait for ponderhit or stop.
		// We use a small sleep to avoid busy-waiting.
		for u.isPondering && atomic.LoadInt32(e.Stopped) == 0 {
			time.Sleep(10 * time.Millisecond)
		}

		// Don't report bestmove if this search instance was superseded by a new 'go' command
		if atomic.LoadInt32(e.Stopped) != 0 && u.engine != e {
			return
		}

		if bestMove == engine.NoMove {
			ml := e.Board.GenerateMoves()
			if ml.Count > 0 {
				bestMove = ml.Moves[0]
				// Send a final info string for the fallback move to satisfy engine controllers
				score := eval.Evaluate(e.Board)
				u.send(fmt.Sprintf("info depth 1 score cp %d nodes %d pv %s", score, atomic.LoadUint64(e.Nodes), bestMove.String()))
			} else {
				return // Terminal state (mate/stalemate)
			}
		}

		// Probing the TT for a ponder move
		ponderMoveStr := ""
		tempBoard := *e.Board
		if tempBoard.MakeMove(bestMove) {
			_, pMove, found := search.GlobalTT.Probe(tempBoard.Hash, 0, -search.Infinity, search.Infinity, 0)
			if found && pMove != engine.NoMove {
				// Verify legality of the ponder move
				pml := tempBoard.GenerateMoves()
				for i := 0; i < pml.Count; i++ {
					if pml.Moves[i] == pMove {
						if tempBoard.MakeMove(pMove) {
							ponderMoveStr = pMove.String()
						}
						break
					}
				}
			}
		}

		if ponderMoveStr != "" {
			u.send(fmt.Sprintf("bestmove %s ponder %s", bestMove.String(), ponderMoveStr))
		} else {
			u.send(fmt.Sprintf("bestmove %s", bestMove.String()))
		}

		// Age history tables for the next search
		for c := 0; c < 2; c++ {
			for pt := 0; pt < 7; pt++ {
				for sq := 0; sq < 64; sq++ {
					u.historyTable[c][pt][sq] /= 2
				}
			}
		}
	}(u.engine, depth)
}

func (u *UCI) handleStop() {
	if u.engine != nil {
		atomic.StoreInt32(u.engine.Stopped, 1)
	}
	u.isPondering = false
}

func (u *UCI) handlePonderHit() {
	u.isPondering = false
	if u.engine != nil && u.pendingTimeLimit > 0 {
		u.engine.TimeLimit = u.pendingTimeLimit
		u.engine.SoftLimit = (u.pendingTimeLimit * 6) / 10
	}
}
