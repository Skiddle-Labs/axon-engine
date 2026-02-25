package main

import (
	"bufio"
	"flag"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/Skiddle-Labs/axon-engine/internal/engine"
	"github.com/Skiddle-Labs/axon-engine/internal/eval"
)

// Entry represents a single position and its game result.
type Entry struct {
	board  *engine.Board
	result float64 // 1.0 for Win, 0.5 for Draw, 0.0 for Loss
}

var (
	dataFile = flag.String("file", "", "Path to the training data file (EPD format)")
	maxIters = flag.Int("iterations", 0, "Number of iterations (0 for until no improvement)")
)

func main() {
	flag.Parse()

	filePath := *dataFile
	if filePath == "" && len(flag.Args()) > 0 {
		filePath = flag.Arg(0)
	}

	if filePath == "" {
		fmt.Println("Axon Tuner - Texel Method")
		fmt.Println("Usage: tuner -file <datafile.epd> [-iterations <n>]")
		flag.PrintDefaults()
		return
	}

	entries, err := LoadEntries(filePath)
	if err != nil {
		fmt.Printf("Error loading entries: %v\n", err)
		return
	}

	if len(entries) == 0 {
		fmt.Println("No valid entries found in file.")
		return
	}

	fmt.Printf("Loaded %d positions for tuning.\n", len(entries))

	// Step 1: Find the optimal scaling constant K for the sigmoid function.
	// This constant maps centipawn scores to expected game results.
	fmt.Print("Calculating optimal K... ")
	bestK := FindBestK(entries)
	fmt.Printf("Done. Best K: %.4f\n", bestK)

	// Step 2: Run the optimization loop.
	RunTuning(entries, bestK, *maxIters)
}

func LoadEntries(path string) ([]Entry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []Entry
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var fen string
		var result float64
		found := false

		if strings.Contains(line, "[") {
			// Format: <FEN> [1.0]
			parts := strings.Split(line, "[")
			fen = strings.TrimSpace(parts[0])
			resStr := strings.Trim(parts[1], " ]")
			switch resStr {
			case "1.0":
				result, found = 1.0, true
			case "0.5":
				result, found = 0.5, true
			case "0.0":
				result, found = 0.0, true
			}
		} else {
			// Format: <FEN> ... "1-0"; or "1/2-1/2" or "0-1"
			if strings.Contains(line, "\"1-0\"") {
				result, found = 1.0, true
			} else if strings.Contains(line, "\"1/2-1/2\"") {
				result, found = 0.5, true
			} else if strings.Contains(line, "\"0-1\"") {
				result, found = 0.0, true
			}

			if found {
				fen = line
				if idx := strings.Index(line, "c9"); idx != -1 {
					fen = strings.TrimSpace(line[:idx])
				} else if idx := strings.Index(line, ";"); idx != -1 {
					fen = strings.TrimSpace(line[:idx])
				}
			}
		}

		if !found {
			continue
		}

		b := engine.NewBoard()
		if err := b.SetFEN(fen); err != nil {
			continue
		}

		entries = append(entries, Entry{board: b, result: result})
	}

	return entries, scanner.Err()
}

// Sigmoid maps an evaluation score to a predicted game result (0.0 to 1.0).
func Sigmoid(score, k float64) float64 {
	return 1.0 / (1.0 + math.Pow(10, -k*score/400.0))
}

// CalculateMSE computes the Mean Squared Error between static evaluations and game results.
func CalculateMSE(entries []Entry, k float64) float64 {
	errorSum := 0.0
	for _, e := range entries {
		// Static evaluation from the perspective of the side to move.
		score := float64(eval.Evaluate(e.board))

		// If side to move is black, the result needs to be inverted for comparison.
		actualResult := e.result
		if e.board.SideToMove == engine.Black {
			actualResult = 1.0 - actualResult
		}

		prediction := Sigmoid(score, k)
		errorSum += math.Pow(actualResult-prediction, 2)
	}
	return errorSum / float64(len(entries))
}

func FindBestK(entries []Entry) float64 {
	bestK := 0.0
	minError := math.MaxFloat64

	// Search for K that minimizes MSE
	for k := 0.1; k <= 2.0; k += 0.01 {
		err := CalculateMSE(entries, k)
		if err < minError {
			minError = err
			bestK = k
		}
	}
	return bestK
}

func RunTuning(entries []Entry, k float64, maxIterations int) {
	params, names := getTunableParams()
	bestMSE := CalculateMSE(entries, k)

	fmt.Printf("Initial MSE: %.10f\n", bestMSE)
	fmt.Println("Starting Local Search optimization...")

	iteration := 1
	for {
		if maxIterations > 0 && iteration > maxIterations {
			fmt.Printf("\nReached maximum iterations: %d\n", maxIterations)
			printParams(params, names)
			break
		}

		improved := false
		fmt.Printf("Iteration %d | Current MSE: %.10f\n", iteration, bestMSE)

		for _, p := range params {
			oldVal := *p

			// Try increasing
			*p = oldVal + 1
			newMSE := CalculateMSE(entries, k)
			if newMSE < bestMSE {
				bestMSE = newMSE
				improved = true
				continue
			}

			// Try decreasing
			*p = oldVal - 1
			newMSE = CalculateMSE(entries, k)
			if newMSE < bestMSE {
				bestMSE = newMSE
				improved = true
				continue
			}

			// Restore if no improvement
			*p = oldVal
		}

		if !improved {
			fmt.Println("\nOptimization complete. No further improvements found.")
			printParams(params, names)
			break
		}

		if iteration%10 == 0 {
			printParams(params, names)
		}
		iteration++
	}
}

func getTunableParams() ([]*int, []string) {
	var params []*int
	var names []string

	// Material
	params = append(params, &eval.PawnMG, &eval.PawnEG, &eval.KnightMG, &eval.KnightEG)
	names = append(names, "PawnMG", "PawnEG", "KnightMG", "KnightEG")
	params = append(params, &eval.BishopMG, &eval.BishopEG, &eval.RookMG, &eval.RookEG)
	names = append(names, "BishopMG", "BishopEG", "RookMG", "RookEG")
	params = append(params, &eval.QueenMG, &eval.QueenEG)
	names = append(names, "QueenMG", "QueenEG")

	// PSTs
	typeNames := []string{"None", "Pawn", "Knight", "Bishop", "Rook", "Queen", "King"}
	for pt := engine.Pawn; pt <= engine.King; pt++ {
		for i := 0; i < 64; i++ {
			params = append(params, &eval.MgPST[pt][i])
			names = append(names, fmt.Sprintf("MgPST[%s][%d]", typeNames[pt], i))
			params = append(params, &eval.EgPST[pt][i])
			names = append(names, fmt.Sprintf("EgPST[%s][%d]", typeNames[pt], i))
		}
	}

	// King Safety
	for pt := engine.Knight; pt <= engine.Queen; pt++ {
		params = append(params, &eval.KingAttackerWeight[pt])
		names = append(names, fmt.Sprintf("KingAttackerWeight[%s]", typeNames[pt]))
	}
	for i := 0; i < 100; i++ {
		params = append(params, &eval.SafetyTable[i])
		names = append(names, fmt.Sprintf("SafetyTable[%d]", i))
	}

	return params, names
}

func printParams(params []*int, names []string) {
	fmt.Println("\n--- Current Parameter Values ---")
	for i := 0; i < len(params); i++ {
		// Only print major values or non-zero changes if too many
		if i < 10 {
			fmt.Printf("%s: %d\n", names[i], *params[i])
		}
	}
	fmt.Println("--------------------------------")
}
