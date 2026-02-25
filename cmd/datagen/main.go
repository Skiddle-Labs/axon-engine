package main

import (
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
)

var (
	numGames    = flag.Int("games", 100, "Number of games to generate")
	numThreads  = flag.Int("threads", 1, "Number of parallel games")
	searchDepth = flag.Int("depth", 8, "Search depth for each move")
	randomMoves = flag.Int("random", 8, "Number of random moves at start")
	bookFile    = flag.String("book", "", "Path to Polyglot book file (optional)")
	outputFile  = flag.String("out", "data.epd", "Output file for training data")
)

type GameResult int

const (
	ResultWin  GameResult = 1
	ResultDraw GameResult = 0
	ResultLoss GameResult = -1
)

type Config struct {
	NumGames    int
	NumThreads  int
	SearchDepth int
	RandomMoves int
	BookFile    string
	OutputFile  string
}

func main() {
	flag.Parse()

	fmt.Printf("Axon Datagen - Generating %d games using %d threads\n", *numGames, *numThreads)
	fmt.Printf("Settings: Depth %d, Random Moves %d, Output %s\n", *searchDepth, *randomMoves, *outputFile)

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

	// Ensure TT is initialized
	search.GlobalTT = search.NewTranspositionTable(64)

	file, err := os.OpenFile(*outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error opening output file: %v\n", err)
		return
	}
	defer file.Close()

	var wg sync.WaitGroup
	gamesRemaining := int32(*numGames)
	totalPositions := uint64(0)

	startTime := time.Now()

	for i := 0; i < *numThreads; i++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()
			for atomic.AddInt32(&gamesRemaining, -1) >= 0 {
				positions, result := PlaySingleGame(book, *searchDepth, *randomMoves)
				if len(positions) > 0 {
					SaveGame(file, positions, result)
					atomic.AddUint64(&totalPositions, uint64(len(positions)))
				}

				rem := atomic.LoadInt32(&gamesRemaining)
				if rem%10 == 0 && rem >= 0 {
					fmt.Printf("Thread %d: %d games left...\n", threadID, rem)
				}
			}
		}(i)
	}

	wg.Wait()

	duration := time.Since(startTime).Seconds()
	posCount := atomic.LoadUint64(&totalPositions)
	fmt.Printf("\nFinished! Generated %d positions in %.2f seconds (%.0f pos/sec)\n", posCount, duration, float64(posCount)/duration)
}

func PlaySingleGame(book *engine.PolyglotBook, searchDepth int, randomMoves int) ([]string, GameResult) {
	board := engine.NewBoard()
	board.SetFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	// 1. Play moves from book or random moves to start
	for i := 0; i < randomMoves; i++ {
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
			return nil, ResultDraw // Rare for startpos
		}

		move := legalMoves[rand.Intn(len(legalMoves))]
		board.MakeMove(move)
	}

	var positions []string

	// 2. Self-play until end
	for ply := 0; ply < 400; ply++ {
		// Detect terminal states
		ml := board.GenerateMoves()
		legalCount := 0
		for i := 0; i < ml.Count; i++ {
			if board.MakeMove(ml.Moves[i]) {
				board.UnmakeMove(ml.Moves[i])
				legalCount++
			}
		}

		if legalCount == 0 {
			inCheck := board.IsSquareAttacked(board.Pieces[board.SideToMove][engine.King].LSB(), board.SideToMove^1)
			if inCheck {
				if board.SideToMove == engine.White {
					return positions, ResultLoss // Black wins
				}
				return positions, ResultWin // White wins
			}
			return positions, ResultDraw // Stalemate
		}

		// Draw detections
		if board.HalfMoveClock >= 100 {
			return positions, ResultDraw
		}

		// 3-fold repetition check
		reps := 0
		for i := 0; i < board.Ply; i++ {
			if board.History[i].Hash == board.Hash {
				reps++
			}
		}
		if reps >= 2 {
			return positions, ResultDraw
		}

		// Record position (only if not in check and not too early/late for diversity)
		// Positions reached via book are not recorded.
		if board.Ply > randomMoves {
			// We store the FEN from the perspective of the result.
			// Tuner expects: FEN [1.0/0.5/0.0] where result is for the side in the FEN.
			fen := GetFENWithoutCounters(board)
			positions = append(positions, fen)
		}

		// Search for move
		eng := search.NewEngine(board)
		move := eng.Search(searchDepth)

		if move == engine.NoMove {
			// Fallback to first legal move
			for i := 0; i < ml.Count; i++ {
				if board.MakeMove(ml.Moves[i]) {
					board.UnmakeMove(ml.Moves[i])
					move = ml.Moves[i]
					break
				}
			}
		}

		if !board.MakeMove(move) {
			break // Should not happen
		}
	}

	return positions, ResultDraw
}

func GetFENWithoutCounters(b *engine.Board) string {
	// We want a clean FEN for the tuner.
	// The tuner usually cares about the side to move and the pieces.
	// Standard EPD format is common.
	fields := make([]string, 0, 4)

	// Pieces
	var pieces strings.Builder
	for r := 7; r >= 0; r-- {
		empty := 0
		for f := 0; f < 8; f++ {
			p := b.PieceAt(engine.NewSquare(f, r))
			if p == engine.NoPiece {
				empty++
			} else {
				if empty > 0 {
					pieces.WriteString(fmt.Sprintf("%d", empty))
					empty = 0
				}
				pieces.WriteString(GetPieceChar(p))
			}
		}
		if empty > 0 {
			pieces.WriteString(fmt.Sprintf("%d", empty))
		}
		if r > 0 {
			pieces.WriteByte('/')
		}
	}
	fields = append(fields, pieces.String())

	// Side to move
	if b.SideToMove == engine.White {
		fields = append(fields, "w")
	} else {
		fields = append(fields, "b")
	}

	// Castling
	castling := ""
	if b.Castling&engine.WhiteKingside != 0 {
		castling += "K"
	}
	if b.Castling&engine.WhiteQueenside != 0 {
		castling += "Q"
	}
	if b.Castling&engine.BlackKingside != 0 {
		castling += "k"
	}
	if b.Castling&engine.BlackQueenside != 0 {
		castling += "q"
	}
	if castling == "" {
		castling = "-"
	}
	fields = append(fields, castling)

	// EP
	if b.EnPassant != engine.NoSquare {
		fields = append(fields, b.EnPassant.String())
	} else {
		fields = append(fields, "-")
	}

	return strings.Join(fields, " ")
}

func GetPieceChar(p engine.Piece) string {
	chars := ".PNBRQKpnbrqk"
	return string(chars[int(p)])
}

var fileMutex sync.Mutex

func SaveGame(file *os.File, positions []string, result GameResult) {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	resStr := "0.5"
	if result == ResultWin {
		resStr = "1.0"
	} else if result == ResultLoss {
		resStr = "0.0"
	}

	for _, fen := range positions {
		// The result in the data file should be from the perspective of the player in the FEN.
		// If result is 1.0 (White won) and FEN is Black to move, then for Black the result is 0.0.
		// However, tuner's loadEntries handles this:
		// actualResult := e.result
		// if e.board.SideToMove == engine.Black { actualResult = 1.0 - actualResult }
		// So we always store the result relative to White.

		fmt.Fprintf(file, "%s [%s]\n", fen, resStr)
	}
}
