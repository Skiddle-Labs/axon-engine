package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Skiddle-Labs/axon-engine/internal/engine"
)

var (
	inputFile   = flag.String("in", "", "Input file path")
	outputFile  = flag.String("out", "processed.bin", "Output file path")
	inputFormat = flag.String("if", "epd", "Input format (epd, bin)")
	outFormat   = flag.String("of", "bin", "Output format (epd, bin)")
	shuffle     = flag.Bool("shuffle", false, "Shuffle the positions (requires loading all into memory)")
	limit       = flag.Int("limit", 0, "Limit the number of positions to process")
)

func main() {
	flag.Parse()

	if *inputFile == "" {
		fmt.Println("Input file required: -in <path>")
		flag.Usage()
		return
	}

	start := time.Now()
	var entries []engine.PackedPos

	fmt.Printf("Reading %s (%s)...\n", *inputFile, *inputFormat)
	if *inputFormat == "epd" {
		entries = readEPD(*inputFile)
	} else if *inputFormat == "bin" {
		entries = readBin(*inputFile)
	} else {
		fmt.Printf("Unknown input format: %s\n", *inputFormat)
		return
	}

	if len(entries) == 0 {
		fmt.Println("No entries found or failed to parse.")
		return
	}

	fmt.Printf("Loaded %d positions.\n", len(entries))

	if *shuffle {
		fmt.Println("Shuffling...")
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		rng.Shuffle(len(entries), func(i, j int) {
			entries[i], entries[j] = entries[j], entries[i]
		})
	}

	if *limit > 0 && *limit < len(entries) {
		fmt.Printf("Limiting to %d positions.\n", *limit)
		entries = entries[:*limit]
	}

	fmt.Printf("Writing to %s (%s)...\n", *outputFile, *outFormat)
	if *outFormat == "bin" {
		writeBin(*outputFile, entries)
	} else if *outFormat == "epd" {
		writeEPD(*outputFile, entries)
	} else {
		fmt.Printf("Unknown output format: %s\n", *outFormat)
		return
	}

	fmt.Printf("Done! Processed %d positions in %v\n", len(entries), time.Since(start))
}

func readEPD(path string) []engine.PackedPos {
	file, err := os.Open(path)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return nil
	}
	defer file.Close()

	var entries []engine.PackedPos
	scanner := bufio.NewScanner(file)
	board := engine.NewBoard()

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Handle various EPD formats
		// Basic: FEN [result]
		// Extended: FEN; ce 123; [result]

		result := 0
		if strings.Contains(line, "[1.0]") || strings.Contains(line, "\"1.0\"") {
			result = 1
		} else if strings.Contains(line, "[0.0]") || strings.Contains(line, "\"0.0\"") {
			result = -1
		}

		score := 0
		if idx := strings.Index(line, "ce "); idx != -1 {
			scoreStr := ""
			for i := idx + 3; i < len(line); i++ {
				if line[i] == ';' || line[i] == ' ' || line[i] == '"' {
					break
				}
				scoreStr += string(line[i])
			}
			if s, err := strconv.Atoi(scoreStr); err == nil {
				score = s
			}
		}

		// Strip everything after first semicolon or result bracket for FEN parsing
		fen := line
		if idx := strings.Index(fen, ";"); idx != -1 {
			fen = fen[:idx]
		}
		if idx := strings.Index(fen, " ["); idx != -1 {
			fen = fen[:idx]
		}
		fen = strings.TrimSpace(fen)

		if err := board.SetFEN(fen); err == nil {
			entries = append(entries, board.Pack(score, result))
		}
	}
	return entries
}

func readBin(path string) []engine.PackedPos {
	file, err := os.Open(path)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return nil
	}
	defer file.Close()

	var entries []engine.PackedPos
	for {
		p, err := engine.Deserialize(file)
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		entries = append(entries, p)
	}
	return entries
}

func writeBin(path string, entries []engine.PackedPos) {
	file, err := os.Create(path)
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return
	}
	defer file.Close()
	writer := bufio.NewWriterSize(file, 1024*1024)

	for _, entry := range entries {
		entry.Serialize(writer)
	}
	writer.Flush()
}

func writeEPD(path string, entries []engine.PackedPos) {
	file, err := os.Create(path)
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return
	}
	defer file.Close()
	writer := bufio.NewWriter(file)

	for _, entry := range entries {
		board, score, result := entry.Unpack()
		resStr := " [0.5]"
		if result == 1 {
			resStr = " [1.0]"
		} else if result == -1 {
			resStr = " [0.0]"
		}

		// Use 4-field FEN for training EPD
		parts := strings.Fields(board.FEN())
		epd := strings.Join(parts[:4], " ")

		fmt.Fprintf(writer, "%s; ce %d;%s\n", epd, score, resStr)
	}
	writer.Flush()
}
