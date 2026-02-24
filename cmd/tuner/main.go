package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/personal-github/axon-engine/internal/engine"
	"github.com/personal-github/axon-engine/internal/eval"
)

// Entry represents a single position and its game result.
type Entry struct {
	board  *engine.Board
	result float64 // 1.0 for Win, 0.5 for Draw, 0.0 for Loss
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Axon Tuner - Texel Method")
		fmt.Println("Usage: tuner <datafile.epd>")
		fmt.Println("Data format: FEN [result]")
		return
	}

	entries, err := loadEntries(os.Args[1])
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
	bestK := findBestK(entries)
	fmt.Printf("Done. Best K: %.4f\n", bestK)

	// Step 2: Run the optimization loop.
	runTuning(entries, bestK)
}

func loadEntries(path string) ([]Entry, error) {
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

		// Expected format: <FEN> [<result>]
		parts := strings.Split(line, "[")
		if len(parts) < 2 {
			continue
		}

		fen := strings.TrimSpace(parts[0])
		resultPart := strings.Trim(parts[1], " ]")

		var result float64
		switch resultPart {
		case "1.0":
			result = 1.0
		case "0.5":
			result = 0.5
		case "0.0":
			result = 0.0
		default:
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

// sigmoid maps an evaluation score to a predicted game result (0.0 to 1.0).
func sigmoid(score, k float64) float64 {
	return 1.0 / (1.0 + math.Pow(10, -k*score/400.0))
}

// calculateMSE computes the Mean Squared Error between static evaluations and game results.
func calculateMSE(entries []Entry, k float64) float64 {
	errorSum := 0.0
	for _, e := range entries {
		// Static evaluation from the perspective of the side to move.
		score := float64(eval.Evaluate(e.board))

		// If side to move is black, the result needs to be inverted for comparison.
		actualResult := e.result
		if e.board.SideToMove == engine.Black {
			actualResult = 1.0 - actualResult
		}

		prediction := sigmoid(score, k)
		errorSum += math.Pow(actualResult-prediction, 2)
	}
	return errorSum / float64(len(entries))
}

func findBestK(entries []Entry) float64 {
	bestK := 0.0
	minError := math.MaxFloat64

	// Search for K that minimizes MSE
	for k := 0.1; k <= 2.0; k += 0.01 {
		err := calculateMSE(entries, k)
		if err < minError {
			minError = err
			bestK = k
		}
	}
	return bestK
}

func runTuning(entries []Entry, k float64) {
	params, names := getTunableParams()
	bestMSE := calculateMSE(entries, k)

	fmt.Printf("Initial MSE: %.10f\n", bestMSE)
	fmt.Println("Starting Local Search optimization...")

	iteration := 1
	for {
		improved := false
		fmt.Printf("Iteration %d | Current MSE: %.10f\n", iteration, bestMSE)

		for _, p := range params {
			oldVal := *p

			// Try increasing
			*p = oldVal + 1
			newMSE := calculateMSE(entries, k)
			if newMSE < bestMSE {
				bestMSE = newMSE
				improved = true
				continue
			}

			// Try decreasing
			*p = oldVal - 1
			newMSE = calculateMSE(entries, k)
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
