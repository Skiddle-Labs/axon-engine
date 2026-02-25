package protocol

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Skiddle-Labs/axon-engine/internal/engine"
	"github.com/Skiddle-Labs/axon-engine/internal/eval"
	"github.com/Skiddle-Labs/axon-engine/internal/logger"
	"github.com/Skiddle-Labs/axon-engine/internal/search"
)

const startFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

// Protocol manages UCI protocol communication.
type Protocol struct {
	reader           *bufio.Scanner
	writer           io.Writer
	board            *engine.Board
	search           *search.Engine
	threads          int
	multiPV          int
	moveOverhead     int
	slowMover        int
	analyseMode      bool
	isPondering      bool
	pendingTimeLimit time.Duration
	book             *engine.PolyglotBook
	bookBestMove     bool
	bookDepth        int

	// Persistent Search Heuristics
	historyTable [2][7][64]int
	counterMoves [64][64]engine.Move
}

// NewProtocol creates a new Protocol handler.
func NewProtocol(input io.Reader, output io.Writer) *Protocol {
	b := engine.NewBoard()
	b.SetFEN(startFEN)

	return &Protocol{
		reader:       bufio.NewScanner(input),
		writer:       output,
		board:        b,
		threads:      1,
		multiPV:      1,
		moveOverhead: 10,
		slowMover:    100,
		bookDepth:    255,
	}
}

// Start begins the main loop for processing UCI commands.
func (p *Protocol) Start() {
	logger.Info("UCI Protocol handler started")
	for p.reader.Scan() {
		line := strings.TrimSpace(p.reader.Text())
		if line == "" {
			continue
		}

		logger.Debug("<< %s", line)
		parts := strings.Fields(line)
		command := parts[0]

		switch command {
		case "uci":
			p.handleUCI()
		case "isready":
			p.handleIsReady()
		case "bench":
			p.handleBench()
		case "position":
			p.handlePosition(parts)
		case "go":
			p.handleGo(parts)
		case "stop":
			p.handleStop()
		case "ponderhit":
			p.handlePonderHit()
		case "ucinewgame":
			p.handleUCINewGame()
		case "setoption":
			p.handleSetOption(parts)
		case "d":
			p.handleDisplay()
		case "eval":
			p.handleEval()
		case "quit":
			return
		default:
			// Ignore unknown commands for now
		}
	}
}

func (p *Protocol) handleUCI() {
	p.send("id name Axon Engine")
	p.send("id author Axon Team")
	p.send("option name Hash type spin default 64 min 1 max 65536")
	p.send("option name Threads type spin default 1 min 1 max 128")
	p.send("option name MultiPV type spin default 1 min 1 max 128")
	p.send("option name Ponder type check default false")
	p.send("option name Move Overhead type spin default 10 min 0 max 5000")
	p.send("option name Slow Mover type spin default 100 min 10 max 1000")
	p.send("option name Clear Hash type button")
	p.send("option name UCI_AnalyseMode type check default false")
	p.send("option name UCI_Opponent type string")
	p.send("option name Book File type string default <none>")
	p.send("option name Book Best Move type check default false")
	p.send("option name Book Depth type spin default 255 min 0 max 255")
	p.send("option name Log File type string default <none>")
	p.send("uciok")
}

func (p *Protocol) handleIsReady() {
	p.send("readyok")
}

func (p *Protocol) handlePosition(parts []string) {
	if len(parts) < 2 {
		return
	}

	var fen string
	moveIndex := -1

	for i, part := range parts {
		if part == "moves" {
			moveIndex = i
			break
		}
	}

	if parts[1] == "startpos" {
		fen = startFEN
	} else if parts[1] == "fen" {
		endIndex := len(parts)
		if moveIndex != -1 {
			endIndex = moveIndex
		}
		if endIndex <= 2 {
			return
		}
		fen = strings.Join(parts[2:endIndex], " ")
	} else {
		return
	}

	p.board.SetFEN(fen)

	if moveIndex != -1 {
		for i := moveIndex + 1; i < len(parts); i++ {
			move := p.parseMove(parts[i])
			if move != engine.NoMove {
				p.board.MakeMove(move)
			}
		}
	}
}

func (p *Protocol) parseMove(moveStr string) engine.Move {
	ml := p.board.GenerateMoves()
	for i := 0; i < ml.Count; i++ {
		m := ml.Moves[i]

		// Verify legality by making and unmaking the move
		if !p.board.MakeMove(m) {
			continue
		}
		p.board.UnmakeMove(m)

		s := fmt.Sprintf("%s%s", m.From().String(), m.To().String())

		if len(moveStr) == 5 {
			var pStr string
			switch m.Flags() & 0xB000 {
			case engine.PromoQueen:
				pStr = "q"
			case engine.PromoRook:
				pStr = "r"
			case engine.PromoBishop:
				pStr = "b"
			case engine.PromoKnight:
				pStr = "n"
			}
			if s+pStr == moveStr {
				return m
			}
		} else if s == moveStr {
			return m
		}
	}
	return engine.NoMove
}

func (p *Protocol) handleGo(parts []string) {
	// Probe opening book
	if p.board.Ply <= p.bookDepth {
		if move, ok := p.book.GetMove(p.board); ok {
			p.send(fmt.Sprintf("info depth 1 score cp 0 nodes 0 pv %s", move.String()))
			p.send(fmt.Sprintf("bestmove %s", move.String()))
			return
		}
	}

	if p.search != nil {
		atomic.StoreInt32(p.search.Stopped, 1)
	}

	p.search = search.NewEngine(p.board)
	p.search.Threads = p.threads
	p.search.MultiPV = p.multiPV
	p.search.HistoryTable = &p.historyTable
	p.search.CounterMoves = &p.counterMoves
	p.isPondering = false
	p.pendingTimeLimit = 0

	ml := p.board.GenerateMoves()
	legalCount := 0
	var lastLegal engine.Move
	for i := 0; i < ml.Count; i++ {
		if p.board.MakeMove(ml.Moves[i]) {
			p.board.UnmakeMove(ml.Moves[i])
			legalCount++
			lastLegal = ml.Moves[i]
		}
	}

	// Optimization: Instant move if only one legal move is available and not pondering
	isPonder := false
	for _, part := range parts {
		if part == "ponder" {
			isPonder = true
			break
		}
	}

	if legalCount == 1 && !isPonder {
		score := eval.Evaluate(p.board)
		p.send(fmt.Sprintf("info depth 1 score cp %d nodes 0 pv %s", score, lastLegal.String()))
		p.send(fmt.Sprintf("bestmove %s", lastLegal.String()))
		return
	}

	p.send(fmt.Sprintf("info string searching with %d threads", p.threads))

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
			p.isPondering = true
		case "depth":
			if i+1 < len(parts) {
				if d, err := strconv.Atoi(parts[i+1]); err == nil {
					depth = d
				}
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
				if n, err := strconv.ParseUint(parts[i+1], 10, 64); err == nil {
					nodesLimit = n
				}
				i++
			}
		case "infinite":
			depth = 128
		}
	}

	if movetime > 0 {
		timeLimit = time.Duration(movetime) * time.Millisecond
	} else {
		myTime := wtime
		myInc := winc
		if p.board.SideToMove == engine.Black {
			myTime = btime
			myInc = binc
		}

		if myTime >= 0 {
			mtg := movestogo
			if mtg <= 0 {
				// Dynamic moves-to-go allocation: be more aggressive in midgame
				// (higher piece count -> lower mtg) and more conservative in endgame
				// (lower piece count -> higher mtg) to maintain a safer reserve.
				occCount := p.board.Occupancy().Count()
				mtg = 50 - occCount
				if mtg < 20 {
					mtg = 20
				}
			}

			// Allocate time, leaving a safety buffer from the increment.
			timeLimit = time.Duration(myTime/mtg)*time.Millisecond + time.Duration(myInc)*8/10*time.Millisecond
		}
	}

	if timeLimit > 0 {
		timeLimit = (timeLimit * time.Duration(p.slowMover)) / 100
		timeLimit -= time.Duration(p.moveOverhead) * time.Millisecond
		if timeLimit < 1*time.Millisecond {
			timeLimit = 1 * time.Millisecond
		}

		logger.Info("Time allocation: %v (Soft limit: %v)", timeLimit, (timeLimit*6)/10)

		if !p.isPondering {
			p.search.TimeLimit = timeLimit
			p.search.SoftLimit = (timeLimit * 6) / 10
		} else {
			p.pendingTimeLimit = timeLimit
		}
	}

	p.search.NodesLimit = nodesLimit

	go func(e *search.Engine, d int) {
		bestMove := e.Search(d)

		for p.isPondering {
			time.Sleep(10 * time.Millisecond)
			if atomic.LoadInt32(e.Stopped) != 0 {
				break
			}
		}

		if bestMove == engine.NoMove {
			ml := e.Board.GenerateMoves()
			if ml.Count > 0 {
				bestMove = ml.Moves[0]
				score := eval.Evaluate(e.Board)
				p.send(fmt.Sprintf("info depth 1 score cp %d nodes %d pv %s", score, atomic.LoadUint64(e.Nodes), bestMove.String()))
			} else {
				return
			}
		}

		moveStr := bestMove.String()

		// Get ponder move from TT
		ponderMoveStr := ""
		tempBoard := *e.Board
		if tempBoard.MakeMove(bestMove) {
			_, ponderMove, found := search.GlobalTT.Probe(tempBoard.Hash, 0, -search.Infinity, search.Infinity, tempBoard.Ply)
			if found && ponderMove != engine.NoMove {
				// Verify legality
				ml := tempBoard.GenerateMoves()
				for i := 0; i < ml.Count; i++ {
					if ml.Moves[i] == ponderMove {
						if tempBoard.MakeMove(ponderMove) {
							ponderMoveStr = ponderMove.String()
						}
						break
					}
				}
			}
		}

		if ponderMoveStr != "" {
			p.send(fmt.Sprintf("bestmove %s ponder %s", moveStr, ponderMoveStr))
		} else {
			p.send(fmt.Sprintf("bestmove %s", moveStr))
		}
		logger.Info("Search finished. Best move: %s", moveStr)

		// Age persistent history table to favor more recent search results
		for c := 0; c < 2; c++ {
			for pt := 0; pt < 7; pt++ {
				for sq := 0; sq < 64; sq++ {
					p.historyTable[c][pt][sq] /= 2
				}
			}
		}
	}(p.search, depth)
}

func (p *Protocol) handleStop() {
	p.isPondering = false
	if p.search != nil {
		atomic.StoreInt32(p.search.Stopped, 1)
	}
}

func (p *Protocol) handlePonderHit() {
	if !p.isPondering {
		return
	}
	p.isPondering = false
	if p.search != nil && p.pendingTimeLimit > 0 {
		p.search.TimeLimit = p.pendingTimeLimit
		p.search.StartTime = time.Now()
	}
}

func (p *Protocol) handleUCINewGame() {
	p.board.Clear()
	p.board.SetFEN(startFEN)
	search.GlobalTT.Clear()

	// Reset persistent heuristics for a new game
	p.historyTable = [2][7][64]int{}
	p.counterMoves = [64][64]engine.Move{}
}

func (p *Protocol) handleSetOption(parts []string) {
	logger.Info("SetOption: %v", parts)
	namePart := ""
	valuePart := ""
	parsingName := false
	parsingValue := false

	for i := 0; i < len(parts); i++ {
		if parts[i] == "name" {
			parsingName = true
			parsingValue = false
			continue
		}
		if parts[i] == "value" {
			parsingName = false
			parsingValue = true
			continue
		}

		if parsingName {
			if namePart != "" {
				namePart += " "
			}
			namePart += parts[i]
		} else if parsingValue {
			if valuePart != "" {
				valuePart += " "
			}
			valuePart += parts[i]
		}
	}

	name := strings.ToLower(namePart)
	value := valuePart

	if name == "hash" {
		if size, err := strconv.Atoi(value); err == nil {
			search.GlobalTT = search.NewTranspositionTable(size)
		}
	} else if name == "threads" {
		if t, err := strconv.Atoi(value); err == nil {
			p.threads = t
		}
	} else if name == "multipv" {
		if m, err := strconv.Atoi(value); err == nil {
			p.multiPV = m
		}
	} else if name == "move overhead" {
		if v, err := strconv.Atoi(value); err == nil {
			p.moveOverhead = v
		}
	} else if name == "slow mover" {
		if v, err := strconv.Atoi(value); err == nil {
			p.slowMover = v
		}
	} else if name == "clear hash" {
		search.GlobalTT.Clear()
	} else if name == "uci_analysemode" {
		p.analyseMode = value == "true"
	} else if name == "uci_opponent" {
		// Standard UCI option
	} else if name == "book file" {
		if p.book != nil {
			p.book.Close()
		}
		if value != "<none>" && value != "" {
			var err error
			p.book, err = engine.OpenBook(value)
			if err != nil {
				p.send(fmt.Sprintf("info string Error opening book: %v", err))
			} else if p.book != nil {
				p.book.Options.BestMove = p.bookBestMove
			}
		} else {
			p.book = nil
		}
	} else if name == "book best move" {
		p.bookBestMove = value == "true"
		if p.book != nil {
			p.book.Options.BestMove = p.bookBestMove
		}
	} else if name == "book depth" {
		if v, err := strconv.Atoi(value); err == nil {
			p.bookDepth = v
		}
	} else if name == "log file" {
		if err := logger.SetFile(value); err != nil {
			p.send(fmt.Sprintf("info string Error opening log file: %v", err))
		} else {
			logger.SetEnabled(true)
			logger.Info("Logging enabled to file: %s", value)
		}
	}
}

func (p *Protocol) handleDisplay() {
	p.send(p.board.String())
}

func (p *Protocol) handleEval() {
	score := eval.Evaluate(p.board)
	p.send(fmt.Sprintf("Evaluation: %d cp", score))

	typeNames := []string{"None", "Pawn", "Knight", "Bishop", "Rook", "Queen", "King"}

	p.send("Material breakdown:")
	whiteMg, whiteEg := 0, 0
	blackMg, blackEg := 0, 0

	for pt := engine.Pawn; pt <= engine.Queen; pt++ {
		wCount := p.board.Pieces[engine.White][pt].Count()
		bCount := p.board.Pieces[engine.Black][pt].Count()

		var mg, eg int
		switch pt {
		case engine.Pawn:
			mg, eg = eval.PawnMG, eval.PawnEG
		case engine.Knight:
			mg, eg = eval.KnightMG, eval.KnightEG
		case engine.Bishop:
			mg, eg = eval.BishopMG, eval.BishopEG
		case engine.Rook:
			mg, eg = eval.RookMG, eval.RookEG
		case engine.Queen:
			mg, eg = eval.QueenMG, eval.QueenEG
		}

		whiteMg += wCount * mg
		whiteEg += wCount * eg
		blackMg += bCount * mg
		blackEg += bCount * eg

		if wCount > 0 || bCount > 0 {
			p.send(fmt.Sprintf("  %-8s | White: %2d (MG: %4d, EG: %4d) | Black: %2d (MG: %4d, EG: %4d)",
				typeNames[pt], wCount, wCount*mg, wCount*eg, bCount, bCount*mg, bCount*eg))
		}
	}

	p.send(fmt.Sprintf("  %-8s | White: (MG: %4d, EG: %4d) | Black: (MG: %4d, EG: %4d)",
		"Total", whiteMg, whiteEg, blackMg, blackEg))
}

func (p *Protocol) handleBench() {
	positions := []string{
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		"r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1",
		"8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1",
		"r3k2r/Pppp1ppp/1b3nbN/nP6/BBP1P3/q4N2/Pp1P2PP/R2Q1RK1 w kq - 0 1",
		"rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 1 8",
		"r4rk1/1pp1qppp/p1np1n2/2b1p1B1/2B1P1b1/P1NP1N2/1PP1QPPP/R4RK1 w - - 0 10",
	}

	totalNodes := uint64(0)
	startTime := time.Now()

	for _, fen := range positions {
		p.board.SetFEN(fen)
		p.search = search.NewEngine(p.board)
		p.search.Threads = p.threads
		p.search.NodesLimit = 0
		p.search.TimeLimit = 0

		p.send(fmt.Sprintf("Benchmarking position: %s", fen))
		p.search.Search(10) // Search to depth 10
		totalNodes += atomic.LoadUint64(p.search.Nodes)
	}

	duration := time.Since(startTime).Seconds()
	nps := uint64(0)
	if duration > 0 {
		nps = uint64(float64(totalNodes) / duration)
	}

	p.send(fmt.Sprintf("\nTotal nodes: %d", totalNodes))
	p.send(fmt.Sprintf("Time: %.3f s", duration))
	p.send(fmt.Sprintf("Nodes per second: %d", nps))
}

func (p *Protocol) send(msg string) {
	logger.Debug(">> %s", msg)
	fmt.Fprintln(p.writer, msg)
}
