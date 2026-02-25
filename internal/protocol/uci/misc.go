package uci

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/Skiddle-Labs/axon-engine/internal/engine"
	"github.com/Skiddle-Labs/axon-engine/internal/eval"
	"github.com/Skiddle-Labs/axon-engine/internal/nnue"
	"github.com/Skiddle-Labs/axon-engine/internal/search"
)

func (u *UCI) handleIsReady() {
	u.send("info string NNUE: " + nnue.NetworkName)
	u.send("readyok")
}

func (u *UCI) handleUCINewGame() {
	if u.engine != nil {
		atomic.StoreInt32(u.engine.Stopped, 1)
	}
	search.GlobalTT.Clear()
	u.board = engine.NewBoard()
	u.historyTable = [2][7][64]int{}
	u.counterMoves = [64][64]engine.Move{}
}

func (u *UCI) handleEval() {
	score := eval.Evaluate(u.board)
	u.send(fmt.Sprintf("Evaluation: %d", score))
}

func (u *UCI) handleBench(fields []string) {
	depth := 10
	if len(fields) > 1 {
		if d, err := strconv.Atoi(fields[1]); err == nil {
			depth = d
		}
	}

	u.board.SetFEN(engine.StartFEN)
	eng := search.NewEngine(u.board)
	eng.Silent = true

	start := time.Now()
	eng.Search(depth)
	elapsed := time.Since(start)

	nodes := atomic.LoadUint64(eng.Nodes)
	nps := uint64(0)
	if elapsed.Seconds() > 0.001 {
		nps = uint64(float64(nodes) / elapsed.Seconds())
	}

	u.send(fmt.Sprintf("Bench: %d nodes, %d nps, %v time", nodes, nps, elapsed))
}

func (u *UCI) handleCount(fields []string) {
	filename := "training_data.epd"
	if len(fields) > 1 {
		filename = fields[1]
	}

	file, err := os.Open(filename)
	if err != nil {
		u.send(fmt.Sprintf("info string Error: could not open %s", filename))
		return
	}
	defer file.Close()

	count := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		count++
	}

	if err := scanner.Err(); err != nil {
		u.send(fmt.Sprintf("info string Error reading %s: %v", filename, err))
		return
	}

	u.send(fmt.Sprintf("info string Total positions in %s: %d", filename, count))
}

func (u *UCI) handlePerft(fields []string) {
	depth := 5
	if len(fields) > 1 {
		if d, err := strconv.Atoi(fields[1]); err == nil {
			depth = d
		}
	}

	start := time.Now()
	ml := u.board.GenerateMoves()
	var totalNodes uint64

	for i := 0; i < ml.Count; i++ {
		move := ml.Moves[i]
		if u.board.MakeMove(move) {
			nodes := u.board.Perft(depth - 1)
			u.board.UnmakeMove(move)
			u.send(fmt.Sprintf("%s: %d", move.String(), nodes))
			totalNodes += nodes
		}
	}

	elapsed := time.Since(start)
	nps := uint64(0)
	if elapsed.Seconds() > 0 {
		nps = uint64(float64(totalNodes) / elapsed.Seconds())
	}

	u.send(fmt.Sprintf("\nDepth: %d\nNodes: %d\nTime: %v\nNPS: %d", depth, totalNodes, elapsed, nps))
}
