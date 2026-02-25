package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/Skiddle-Labs/axon-engine/internal/engine"
	"github.com/Skiddle-Labs/axon-engine/internal/eval"
	"github.com/Skiddle-Labs/axon-engine/internal/types"
)

// Entry represents a single position and its game result.
type Entry struct {
	board  *engine.Board
	result float64 // 1.0 for Win, 0.5 for Draw, 0.0 for Loss
}

var (
	dataFile = flag.String("file", "", "Path to the training data file (EPD format)")
	maxIters = flag.Int("iterations", 0, "Number of iterations (0 for until no improvement)")
	threads  = flag.Int("threads", 0, "Number of threads to use for MSE calculation (defaults to 80% of CPUs)")
	saveFile = flag.String("save", "tuned_params.txt", "Path to save the optimized parameters")
	method   = flag.String("method", "local", "Optimization method (local or spsa)")
	lossType = flag.String("loss", "mse", "Loss function to minimize (mse or logloss)")
	binInput = flag.Bool("bin", false, "Use binary input format (.bin)")
	analyze  = flag.Bool("analyze", false, "Analyze K-factor and score buckets")
)

func main() {
	flag.Parse()

	filePath := *dataFile
	if filePath == "" && len(flag.Args()) > 0 {
		filePath = flag.Arg(0)
	}

	if filePath == "" {
		fmt.Println("Axon Tuner - Texel Method")
		fmt.Println("Usage: tuner -file <datafile.epd> [-iterations <n>] [-threads <t>] [-save <file>] [-method <local|spsa>] [-loss <mse|logloss>] [-bin]")
		flag.PrintDefaults()
		return
	}

	var entries []Entry
	var err error
	if *binInput {
		entries, err = LoadBinaryEntries(filePath)
	} else {
		entries, err = LoadEntries(filePath)
	}
	if err != nil {
		fmt.Printf("Error loading entries: %v\n", err)
		return
	}

	if len(entries) == 0 {
		fmt.Println("No valid entries found in file.")
		return
	}

	fmt.Printf("Loaded %d positions for tuning.\n", len(entries))

	if *threads <= 0 {
		t := int(float64(runtime.NumCPU()) * 0.8)
		if t < 1 {
			t = 1
		}
		*threads = t
	}
	fmt.Printf("Using %d threads.\n", *threads)

	fmt.Print("Precomputing features... ")
	precomputed := PrecomputeEntries(entries)
	fmt.Println("Done.")

	// Step 1: Find the optimal scaling constant K for the sigmoid function.
	fmt.Print("Calculating optimal K... ")
	bestK := FindBestK(precomputed)
	fmt.Printf("Done. Best K: %.4f\n", bestK)

	if *analyze {
		AnalyzeK(precomputed, bestK)
		return
	}

	// Step 2: Run the optimization loop.
	if strings.ToLower(*method) == "spsa" {
		if *maxIters == 0 {
			*maxIters = 1000
		}
		RunSPSA(precomputed, bestK, *maxIters)
	} else {
		RunTuning(precomputed, bestK, *maxIters)
	}
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
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		var fen string
		var result float64
		found := false

		// 1. Try to find the result in common EPD/Texel/PGN formats
		if strings.Contains(line, "[1.0]") || strings.Contains(line, "\"1-0\"") || strings.Contains(line, " 1-0") || strings.Contains(line, " 1.0") {
			result, found = 1.0, true
		} else if strings.Contains(line, "[0.5]") || strings.Contains(line, "\"1/2-1/2\"") || strings.Contains(line, " 1/2-1/2") || strings.Contains(line, " 0.5") {
			result, found = 0.5, true
		} else if strings.Contains(line, "[0.0]") || strings.Contains(line, "\"0-1\"") || strings.Contains(line, " 0-1") || strings.Contains(line, " 0.0") {
			result, found = 0.0, true
		}

		// 2. Fallback: Search all fields for a result marker or evaluation tags
		fields := strings.Fields(line)
		if !found && len(fields) > 0 {
			// Check the very last field first (common in some EPD formats)
			lastField := strings.Trim(fields[len(fields)-1], "\";,[]")
			switch lastField {
			case "1-0", "1.0", "1":
				result, found = 1.0, true
			case "1/2-1/2", "0.5", "1/2":
				result, found = 0.5, true
			case "0-1", "0.0", "0":
				result, found = 0.0, true
			}

			for i, f := range fields {
				if found {
					break
				}
				clean := strings.Trim(f, "\";,[]")
				switch clean {
				case "1-0", "1.0", "1":
					result, found = 1.0, true
				case "1/2-1/2", "0.5", "1/2":
					result, found = 0.5, true
				case "0-1", "0.0", "0":
					result, found = 0.0, true
				case "ce", "v":
					// If we find an evaluation tag (ce or v), use it to generate a synthetic result
					if i+1 < len(fields) {
						var evalScore float64
						_, err := fmt.Sscanf(strings.Trim(fields[i+1], "\";,[]"), "%f", &evalScore)
						if err == nil {
							// Use a standard sigmoid to convert centipawns to win probability
							// K=0.75 is a reasonable default for engine evals
							result = Sigmoid(evalScore, 0.75)
							found = true
						}
					}
				}
				if found {
					break
				}
			}
		}

		// 3. Extract FEN
		// Usually the first 4 or 6 fields are the FEN.
		// We'll try to join fields until we have a valid FEN or hit a marker.
		fenIdx := 0
		for i, field := range fields {
			clean := strings.Trim(field, "\";,[]")
			if strings.ContainsAny(field, "[;\"") ||
				clean == "1-0" || clean == "1/2-1/2" || clean == "0-1" ||
				clean == "1.0" || clean == "0.5" || clean == "0.0" ||
				clean == "1" || clean == "0" || clean == "1/2" ||
				clean == "ce" || clean == "v" {
				fenIdx = i
				break
			}
			if i == 5 { // Standard FEN has 6 fields
				fenIdx = 6
				break
			}
			fenIdx = i + 1
		}

		if fenIdx < 4 {
			// Minimal FEN is pieces, side, castling, ep
			continue
		}
		fen = strings.Join(fields[:fenIdx], " ")

		// Final check: If no result was found via tags, we can't use it for Texel tuning
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

func Sigmoid(score, k float64) float64 {
	return 1.0 / (1.0 + math.Exp(-k*score/400.0*math.Ln10))
}

func LoadBinaryEntries(path string) ([]Entry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []Entry
	for {
		p, err := engine.Deserialize(file)
		if err == io.EOF {
			break
		}
		if err != nil {
			return entries, err
		}
		board, _, result := p.Unpack()
		res := 0.5
		if result == 1 {
			res = 1.0
		} else if result == -1 {
			res = 0.0
		}
		entries = append(entries, Entry{board: board, result: res})
	}
	return entries, nil
}

// CalculateMSEParallel computes the Mean Squared Error using multiple threads.
func CalculateMSEParallel(entries []PrecomputedEntry, k float64) float64 {
	numThreads := *threads
	if numThreads <= 0 {
		numThreads = 1
	}

	useLogLoss := strings.ToLower(*lossType) == "logloss"
	chunkSize := (len(entries) + numThreads - 1) / numThreads
	var totalError uint64 // Using bits for atomic storage of float64

	var wg sync.WaitGroup
	for i := 0; i < numThreads; i++ {
		start := i * chunkSize
		if start >= len(entries) {
			break
		}
		end := start + chunkSize
		if end > len(entries) {
			end = len(entries)
		}

		wg.Add(1)
		go func(s, e int) {
			defer wg.Done()
			localError := 0.0
			for j := s; j < e; j++ {
				entry := entries[j]
				score := float64(entry.Evaluate())
				actualResult := entry.Result
				if entry.SideToMove == types.Black {
					actualResult = 1.0 - actualResult
				}
				prediction := Sigmoid(score, k)

				if useLogLoss {
					// Log Loss (Cross-Entropy)
					// Avoid log(0) or log(1) with small epsilon
					const epsilon = 1e-15
					p := math.Max(epsilon, math.Min(1.0-epsilon, prediction))
					localError -= actualResult*math.Log(p) + (1.0-actualResult)*math.Log(1.0-p)
				} else {
					// Mean Squared Error
					localError += math.Pow(actualResult-prediction, 2)
				}
			}

			// Atomic add for float64
			for {
				oldBits := atomic.LoadUint64(&totalError)
				newBits := math.Float64bits(math.Float64frombits(oldBits) + localError)
				if atomic.CompareAndSwapUint64(&totalError, oldBits, newBits) {
					break
				}
			}
		}(start, end)
	}
	wg.Wait()

	return math.Float64frombits(atomic.LoadUint64(&totalError)) / float64(len(entries))
}

func FindBestK(entries []PrecomputedEntry) float64 {
	bestK := 0.0
	minError := math.MaxFloat64

	for k := 0.1; k <= 2.0; k += 0.01 {
		err := CalculateMSEParallel(entries, k)
		if err < minError {
			minError = err
			bestK = k
		}
	}
	return bestK
}

func RunTuning(entries []PrecomputedEntry, k float64, maxIterations int) {
	params, names := getTunableParams()
	bestMSE := CalculateMSEParallel(entries, k)

	lossName := strings.ToUpper(*lossType)
	fmt.Printf("Initial %s: %.10f\n", lossName, bestMSE)
	fmt.Println("Starting Local Search optimization...")

	iteration := 1
	for {
		if maxIterations > 0 && iteration > maxIterations {
			fmt.Printf("\nReached maximum iterations: %d\n", maxIterations)
			printParams(params, names)
			break
		}

		improved := false
		fmt.Printf("Iteration %d | Current %s: %.10f\n", iteration, lossName, bestMSE)

		for i, p := range params {
			oldVal := *p

			// Try increasing
			*p = oldVal + 1
			newMSE := CalculateMSEParallel(entries, k)
			if newMSE < bestMSE {
				bestMSE = newMSE
				improved = true
				fmt.Printf("  %s: %d -> %d (%s: %.10f)\n", names[i], oldVal, *p, lossName, bestMSE)
				saveParams(*saveFile, params, names)
				continue
			}

			// Try decreasing
			*p = oldVal - 1
			newMSE = CalculateMSEParallel(entries, k)
			if newMSE < bestMSE {
				bestMSE = newMSE
				improved = true
				fmt.Printf("  %s: %d -> %d (%s: %.10f)\n", names[i], oldVal, *p, lossName, bestMSE)
				saveParams(*saveFile, params, names)
				continue
			}

			// Restore if no improvement
			*p = oldVal
		}

		if !improved {
			fmt.Println("\nOptimization complete. No further improvements found.")
			printParams(params, names)
			saveParams(*saveFile, params, names)
			break
		}

		// Full iteration summary
		if iteration%10 == 0 {
			printParams(params, names)
		}
		iteration++
	}
}

func saveParams(path string, params []*int, names []string) {
	file, err := os.Create(path)
	if err != nil {
		fmt.Printf("Error creating save file: %v\n", err)
		return
	}
	defer file.Close()

	fmt.Fprintf(file, "// Axon Tuned Parameters - Saved automatically\n\n")
	for i := 0; i < len(params); i++ {
		fmt.Fprintf(file, "%s = %d\n", names[i], *params[i])
	}
	fmt.Printf("Parameters saved to %s\n", path)
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

	// Pawn Structure
	params = append(params, &eval.PawnDoubledMG, &eval.PawnDoubledEG)
	names = append(names, "PawnDoubledMG", "PawnDoubledEG")
	params = append(params, &eval.PawnIsolatedMG, &eval.PawnIsolatedEG)
	names = append(names, "PawnIsolatedMG", "PawnIsolatedEG")
	params = append(params, &eval.PawnSupportedMG, &eval.PawnSupportedEG)
	names = append(names, "PawnSupportedMG", "PawnSupportedEG")
	params = append(params, &eval.PawnPhalanxMG, &eval.PawnPhalanxEG)
	names = append(names, "PawnPhalanxMG", "PawnPhalanxEG")
	params = append(params, &eval.PawnBackwardMG, &eval.PawnBackwardEG)
	names = append(names, "PawnBackwardMG", "PawnBackwardEG")
	params = append(params, &eval.PawnPassedMG, &eval.PawnPassedEG)
	names = append(names, "PawnPassedMG", "PawnPassedEG")

	// Mobility
	for i := range eval.KnightMobilityMG {
		params = append(params, &eval.KnightMobilityMG[i])
		names = append(names, fmt.Sprintf("KnightMobilityMG[%d]", i))
	}
	for i := range eval.KnightMobilityEG {
		params = append(params, &eval.KnightMobilityEG[i])
		names = append(names, fmt.Sprintf("KnightMobilityEG[%d]", i))
	}
	for i := range eval.BishopMobilityMG {
		params = append(params, &eval.BishopMobilityMG[i])
		names = append(names, fmt.Sprintf("BishopMobilityMG[%d]", i))
	}
	for i := range eval.BishopMobilityEG {
		params = append(params, &eval.BishopMobilityEG[i])
		names = append(names, fmt.Sprintf("BishopMobilityEG[%d]", i))
	}
	for i := range eval.RookMobilityMG {
		params = append(params, &eval.RookMobilityMG[i])
		names = append(names, fmt.Sprintf("RookMobilityMG[%d]", i))
	}
	for i := range eval.RookMobilityEG {
		params = append(params, &eval.RookMobilityEG[i])
		names = append(names, fmt.Sprintf("RookMobilityEG[%d]", i))
	}
	for i := range eval.QueenMobilityMG {
		params = append(params, &eval.QueenMobilityMG[i])
		names = append(names, fmt.Sprintf("QueenMobilityMG[%d]", i))
	}
	for i := range eval.QueenMobilityEG {
		params = append(params, &eval.QueenMobilityEG[i])
		names = append(names, fmt.Sprintf("QueenMobilityEG[%d]", i))
	}

	params = append(params, &eval.VirtualMobilityMG, &eval.VirtualMobilityEG)
	names = append(names, "VirtualMobilityMG", "VirtualMobilityEG")

	// Other
	params = append(params, &eval.BishopPairMG, &eval.BishopPairEG)
	names = append(names, "BishopPairMG", "BishopPairEG")
	params = append(params, &eval.WeakAttackerMG, &eval.WeakAttackerEG)
	names = append(names, "WeakAttackerMG", "WeakAttackerEG")
	params = append(params, &eval.HangingDivisorMG, &eval.HangingDivisorEG)
	names = append(names, "HangingDivisorMG", "HangingDivisorEG")

	// Positional
	params = append(params, &eval.KnightOutpostMG, &eval.KnightOutpostEG)
	names = append(names, "KnightOutpostMG", "KnightOutpostEG")
	params = append(params, &eval.BishopOutpostMG, &eval.BishopOutpostEG)
	names = append(names, "BishopOutpostMG", "BishopOutpostEG")
	params = append(params, &eval.RookOpenFileMG, &eval.RookOpenFileEG)
	names = append(names, "RookOpenFileMG", "RookOpenFileEG")
	params = append(params, &eval.RookHalfOpenFileMG, &eval.RookHalfOpenFileEG)
	names = append(names, "RookHalfOpenFileMG", "RookHalfOpenFileEG")

	// PSTs
	typeNames := []string{"None", "Pawn", "Knight", "Bishop", "Rook", "Queen", "King"}
	for pt := types.Pawn; pt <= types.King; pt++ {
		for i := 0; i < 64; i++ {
			params = append(params, &eval.MgPST[pt][i])
			names = append(names, fmt.Sprintf("MgPST[%s][%d]", typeNames[pt], i))
			params = append(params, &eval.EgPST[pt][i])
			names = append(names, fmt.Sprintf("EgPST[%s][%d]", typeNames[pt], i))
		}
	}

	// King Safety
	params = append(params, &eval.KingShieldClose, &eval.KingShieldFar, &eval.KingShieldMissing)
	names = append(names, "KingShieldClose", "KingShieldFar", "KingShieldMissing")
	params = append(params, &eval.PawnStormMG, &eval.PawnStormEG)
	names = append(names, "PawnStormMG", "PawnStormEG")

	for pt := types.Knight; pt <= types.Queen; pt++ {
		params = append(params, &eval.KingAttackerWeight[pt])
		names = append(names, fmt.Sprintf("KingAttackerWeight[%s]", typeNames[pt]))
	}
	for i := 0; i < 100; i++ {
		params = append(params, &eval.SafetyTable[i])
		names = append(names, fmt.Sprintf("SafetyTable[%d]", i))
	}

	return params, names
}

func AnalyzeK(entries []PrecomputedEntry, k float64) {
	fmt.Printf("\n--- Sigmoid Scaling (K=%.4f) Analysis ---\n", k)
	fmt.Printf("%-15s %-10s %-15s %-15s %-10s\n", "Score Range", "Count", "Actual Win %", "Pred Win %", "Diff")
	fmt.Println(strings.Repeat("-", 70))

	buckets := 24
	step := 100

	for i := -buckets / 2; i < buckets/2; i++ {
		min := i * step
		max := (i + 1) * step

		var count int
		var totalResult float64
		var totalPred float64

		for _, entry := range entries {
			score := float64(entry.Evaluate())
			actual := entry.Result
			if entry.SideToMove == types.Black {
				actual = 1.0 - actual
			}

			if int(score) >= min && int(score) < max {
				count++
				totalResult += actual
				totalPred += Sigmoid(score, k)
			}
		}

		if count > 0 {
			actualPct := totalResult / float64(count)
			predPct := totalPred / float64(count)
			fmt.Printf("[%4d, %4d) %-10d %-15.4f %-15.4f %-10.4f\n", min, max, count, actualPct, predPct, actualPct-predPct)
		}
	}
}

func printParams(params []*int, names []string) {
	fmt.Println("\n--- Current Material & Key Parameters ---")
	// Print the first few interesting parameters (Material, Pawn Structure, etc.)
	for i := 0; i < 35 && i < len(params); i++ {
		fmt.Printf("%s: %d\n", names[i], *params[i])
	}
	fmt.Println("-----------------------------------------")
}
