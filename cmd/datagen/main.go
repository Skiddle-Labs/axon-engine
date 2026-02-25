package main

import (
	"bufio"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Skiddle-Labs/axon-engine/internal/engine"
	"github.com/Skiddle-Labs/axon-engine/internal/search"
	"github.com/Skiddle-Labs/axon-engine/internal/types"
)

var (
	numGames    = flag.Int("games", 100, "Number of games to generate")
	numThreads  = flag.Int("threads", 1, "Number of parallel games")
	searchDepth = flag.Int("depth", 8, "Search depth for each move")
	randomMoves = flag.Int("random", 8, "Number of random moves at start")
	bookFile    = flag.String("book", "", "Path to Polyglot book file (optional)")
	outputFile  = flag.String("out", "data.epd", "Output file for training data")
	inputEPD    = flag.String("input", "", "Path to input EPD file for starting positions (optional)")
	minPly      = flag.Int("minply", 16, "Minimum ply to start recording positions")
	maxPly      = flag.Int("maxply", 200, "Maximum ply to stop recording positions")
	adjScore    = flag.Int("adj-score", 1000, "Adjudication score (centipawns) to end games early")
	adjCount    = flag.Int("adj-count", 4, "Number of consecutive moves above adj-score to adjudicate")
)

type GameResult int

const (
	ResultWin  GameResult = 1
	ResultDraw GameResult = 0
	ResultLoss GameResult = -1
)

type GameResultData struct {
	Positions []string
	Result    GameResult
}

func main() {
	flag.Parse()

	fmt.Printf("Axon Bulk Datagen - Target: %d games, Threads: %d\n", *numGames, *numThreads)
	fmt.Printf("Search Depth: %d | Random Moves: %d | Range: %d-%d ply\n", *searchDepth, *randomMoves, *minPly, *maxPly)

	var book *engine.PolyglotBook
	if *bookFile != "" {
		var err error
		book, err = engine.OpenBook(*bookFile)
		if err != nil {
			fmt.Printf("Error opening book: %v\n", err)
			return
		}
		defer book.Close()
		fmt.Printf("Using opening book: %s\n", *bookFile)
	}

	// Load input EPDs if provided
	var inputFens []string
	if *inputEPD != "" {
		data, err := os.ReadFile(*inputEPD)
		if err != nil {
			fmt.Printf("Error reading input EPD: %v\n", err)
			return
		}
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				inputFens = append(inputFens, line)
			}
		}
		fmt.Printf("Loaded %d starting positions from %s\n", len(inputFens), *inputEPD)
		if *numGames > len(inputFens) {
			*numGames = len(inputFens)
		}
	}

	// Shared TT with adequate size for high-concurrency shallow searches
	search.GlobalTT = search.NewTranspositionTable(256)

	file, err := os.OpenFile(*outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error opening output file: %v\n", err)
		return
	}
	defer file.Close()
	writer := bufio.NewWriterSize(file, 1024*1024)

	var wg sync.WaitGroup
	results := make(chan GameResultData, *numThreads*2)
	writerDone := make(chan struct{})

	// Dedicated writer goroutine to avoid mutex contention
	go func() {
		for res := range results {
			SaveToDisk(writer, res.Positions, res.Result)
		}
		writer.Flush()
		close(writerDone)
	}()

	gamesRemaining := int32(*numGames)
	totalPositions := uint64(0)
	totalGames := uint64(0)

	startTime := time.Now()

	for i := 0; i < *numThreads; i++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()
			// Local random source to avoid global mutex contention
			rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(threadID)))

			for {
				idx := int(atomic.AddInt32(&gamesRemaining, -1))
				if idx < 0 {
					break
				}

				startFen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
				if len(inputFens) > 0 {
					startFen = inputFens[idx]
				}

				positions, result := PlaySingleGame(startFen, book, rng)
				if len(positions) > 0 {
					results <- GameResultData{Positions: positions, Result: result}
					atomic.AddUint64(&totalPositions, uint64(len(positions)))
				}

				count := atomic.AddUint64(&totalGames, 1)
				if count%10 == 0 {
					elapsed := time.Since(startTime).Seconds()
					posCount := atomic.LoadUint64(&totalPositions)
					fmt.Printf("\rGames: %d/%d | Positions: %d | Pos/sec: %.0f",
						count, *numGames, posCount, float64(posCount)/elapsed)
				}
			}
		}(i)
	}

	wg.Wait()
	close(results)
	<-writerDone

	duration := time.Since(startTime).Seconds()
	posCount := atomic.LoadUint64(&totalPositions)
	fmt.Printf("\n\nFinished!\nTotal Positions: %d\nTotal Time: %.2f seconds\nFinal Throughput: %.0f pos/sec\n",
		posCount, duration, float64(posCount)/duration)
}

func PlaySingleGame(fen string, book *engine.PolyglotBook, rng *rand.Rand) ([]string, GameResult) {
	board := engine.NewBoard()
	if err := board.SetFEN(fen); err != nil {
		return nil, ResultDraw
	}

	// 1. Randomization phase (Book + Random moves)
	for i := 0; i < *randomMoves; i++ {
		if book != nil {
			if move, ok := book.GetMove(board); ok {
				board.MakeMove(move)
				continue
			}
		}

		ml := board.GenerateMoves()
		legalMoves := make([]engine.Move, 0, ml.Count)
		for j := 0; j < ml.Count; j++ {
			if board.MakeMove(ml.Moves[j]) {
				board.UnmakeMove(ml.Moves[j])
				legalMoves = append(legalMoves, ml.Moves[j])
			}
		}

		if len(legalMoves) == 0 {
			return nil, ResultDraw
		}

		move := legalMoves[rng.Intn(len(legalMoves))]
		board.MakeMove(move)
	}

	var positions []string
	highScoreCounter := 0
	lastScore := 0

	// 2. Self-play with search
	eng := search.NewEngine(board)
	eng.Silent = true

	for ply := 0; ply < 400; ply++ {
		// Adjudication & Draw detections
		if board.HalfMoveClock >= 100 {
			return positions, ResultDraw
		}

		// Simplified 3-fold repetition
		reps := 0
		for i := board.Ply - 4; i >= 0 && i >= board.Ply-int(board.HalfMoveClock); i -= 2 {
			if board.History[i].Hash == board.Hash {
				reps++
				if reps >= 2 {
					return positions, ResultDraw
				}
			}
		}

		// Search for move
		move := eng.Search(*searchDepth)

		if move == engine.NoMove {
			kingSq := board.Pieces[board.SideToMove][types.King].LSB()
			inCheck := board.IsSquareAttacked(kingSq, board.SideToMove^1)
			if inCheck {
				if board.SideToMove == types.White {
					return positions, ResultLoss
				}
				return positions, ResultWin
			}
			return positions, ResultDraw
		}

		// Extract score from TT for adjudication
		score, _, found := search.GlobalTT.Probe(board.Hash, *searchDepth, -search.Infinity, search.Infinity, 0)
		if found {
			absScore := score
			if absScore < 0 {
				absScore = -absScore
			}

			if absScore >= *adjScore {
				highScoreCounter++
			} else {
				highScoreCounter = 0
			}

			// End game if score is consistently huge
			if highScoreCounter >= *adjCount {
				if score > 0 {
					if board.SideToMove == types.White {
						return positions, ResultWin
					}
					return positions, ResultLoss
				} else {
					if board.SideToMove == types.White {
						return positions, ResultLoss
					}
					return positions, ResultWin
				}
			}
			lastScore = score
		}

		// Filter and record position
		if board.Ply >= *minPly && board.Ply <= *maxPly {
			// Skip positions that are in check or have too high/low eval (unstable)
			kingSq := board.Pieces[board.SideToMove][types.King].LSB()
			inCheck := board.IsSquareAttacked(kingSq, board.SideToMove^1)
			if !inCheck && lastScore < 2000 && lastScore > -2000 {
				fen := GetEPDFEN(board)
				positions = append(positions, fen)
			}
		}

		if !board.MakeMove(move) {
			break
		}
	}

	return positions, ResultDraw
}

func GetEPDFEN(b *engine.Board) string {
	var sb strings.Builder
	sb.Grow(90)

	// 1. Piece placement
	for r := 7; r >= 0; r-- {
		empty := 0
		for f := 0; f < 8; f++ {
			p := b.PieceAt(types.Square(r<<3 | f))
			if p == types.NoPiece {
				empty++
			} else {
				if empty > 0 {
					sb.WriteByte(byte('0' + empty))
					empty = 0
				}
				sb.WriteByte(".PNBRQKpnbrqk"[p])
			}
		}
		if empty > 0 {
			sb.WriteByte(byte('0' + empty))
		}
		if r > 0 {
			sb.WriteByte('/')
		}
	}

	// 2. Side to move
	sb.WriteByte(' ')
	if b.SideToMove == types.White {
		sb.WriteByte('w')
	} else {
		sb.WriteByte('b')
	}

	// 3. Castling rights
	sb.WriteByte(' ')
	if b.Castling == 0 {
		sb.WriteByte('-')
	} else {
		if b.Castling&engine.WhiteKingside != 0 {
			sb.WriteByte('K')
		}
		if b.Castling&engine.WhiteQueenside != 0 {
			sb.WriteByte('Q')
		}
		if b.Castling&engine.BlackKingside != 0 {
			sb.WriteByte('k')
		}
		if b.Castling&engine.BlackQueenside != 0 {
			sb.WriteByte('q')
		}
	}

	// 4. En passant square
	sb.WriteByte(' ')
	if b.EnPassant == types.NoSquare {
		sb.WriteByte('-')
	} else {
		sq := b.EnPassant
		sb.WriteByte(byte('a' + (sq & 7)))
		sb.WriteByte(byte('1' + (sq >> 3)))
	}

	return sb.String()
}

func GetPieceChar(p types.Piece) string {
	chars := ".PNBRQKpnbrqk"
	return string(chars[int(p)])
}

func SaveToDisk(writer *bufio.Writer, positions []string, result GameResult) {
	resStr := " [0.5]\n"
	if result == ResultWin {
		resStr = " [1.0]\n"
	} else if result == ResultLoss {
		resStr = " [0.0]\n"
	}

	for _, fen := range positions {
		writer.WriteString(fen)
		writer.WriteString(resStr)
	}
}
