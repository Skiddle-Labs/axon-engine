package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Skiddle-Labs/axon-engine/internal/engine"
)

var (
	inputFile  = flag.String("input", "chessData.csv", "Path to the input CSV file")
	outputFile = flag.String("output", "converted.bin", "Path to the output binary file")
	limit      = flag.Int("limit", 0, "Limit number of positions (0 = no limit)")
)

func main() {
	flag.Parse()

	fmt.Printf("Converting CSV: %s -> %s\n", *inputFile, *outputFile)

	file, err := os.Open(*inputFile)
	if err != nil {
		fmt.Printf("Error opening input: %v\n", err)
		return
	}
	defer file.Close()

	outFile, err := os.Create(*outputFile)
	if err != nil {
		fmt.Printf("Error creating output: %v\n", err)
		return
	}
	defer outFile.Close()

	reader := csv.NewReader(bufio.NewReaderSize(file, 1024*1024))
	// Large datasets often have many columns or vary in formatting;
	// we only care about the first two columns (FEN, Evaluation)
	reader.FieldsPerRecord = -1

	writer := bufio.NewWriterSize(outFile, 1024*1024)
	defer writer.Flush()

	// Skip header
	_, err = reader.Read()
	if err != nil {
		fmt.Printf("Error reading header: %v\n", err)
		return
	}

	board := engine.NewBoard()
	count := 0
	startTime := time.Now()

	for {
		if *limit > 0 && count >= *limit {
			break
		}

		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("Error reading CSV record: %v\n", err)
			continue
		}

		if len(record) < 2 {
			continue
		}

		fen := record[0]
		scoreStr := record[1]

		// Parse score
		// Some datasets have '+' or '#', we handle basic integers here
		scoreStr = strings.TrimPrefix(scoreStr, "+")
		score, err := strconv.Atoi(scoreStr)
		if err != nil {
			// Handle mate scores (e.g., #5 or -#2) by mapping to high values
			if strings.Contains(scoreStr, "#") {
				if strings.Contains(scoreStr, "-") {
					score = -10000
				} else {
					score = 10000
				}
			} else {
				continue
			}
		}

		// Set up board to pack
		err = board.SetFEN(fen)
		if err != nil {
			continue
		}

		// Since we don't have the final game result in the CSV,
		// we use 0 (Draw) as the result indicator.
		// Bullet trainer can handle training on scores alone.
		packed := board.Pack(score, 0)
		err = packed.Serialize(writer)
		if err != nil {
			fmt.Printf("Error writing binary: %v\n", err)
			break
		}

		count++
		if count%100000 == 0 {
			elapsed := time.Since(startTime).Seconds()
			fmt.Printf("\rProcessed: %d | Speed: %.0f pos/sec", count, float64(count)/elapsed)
		}
	}

	elapsed := time.Since(startTime).Seconds()
	fmt.Printf("\n\nFinished!\nTotal: %d positions\nTime: %.2f seconds\nFinal Speed: %.0f pos/sec\n",
		count, elapsed, float64(count)/elapsed)
}
