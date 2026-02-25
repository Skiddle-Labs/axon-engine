package main

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"time"

	"github.com/Skiddle-Labs/axon-engine/internal/engine"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: shuffler <output.bin> <input1.bin> [input2.bin ...]")
		os.Exit(1)
	}

	outputPath := os.Args[1]
	inputPaths := os.Args[2:]

	var allPositions []engine.PackedPos

	for _, path := range inputPaths {
		fmt.Printf("Reading %s...\n", path)
		positions, err := readBinaryFile(path)
		if err != nil {
			fmt.Printf("Error reading %s: %v\n", path, err)
			continue
		}
		allPositions = append(allPositions, positions...)
		fmt.Printf("Loaded %d positions (Total: %d)\n", len(positions), len(allPositions))
	}

	if len(allPositions) == 0 {
		fmt.Println("No positions loaded. Exiting.")
		return
	}

	fmt.Printf("Shuffling %d positions...\n", len(allPositions))
	shuffle(allPositions)

	fmt.Printf("Writing to %s...\n", outputPath)
	err := writeBinaryFile(outputPath, allPositions)
	if err != nil {
		fmt.Printf("Error writing output: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Done!")
}

func readBinaryFile(path string) ([]engine.PackedPos, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var positions []engine.PackedPos
	for {
		pos, err := engine.Deserialize(file)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		positions = append(positions, pos)
	}

	return positions, nil
}

func writeBinaryFile(path string, positions []engine.PackedPos) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, pos := range positions {
		err := pos.Serialize(file)
		if err != nil {
			return err
		}
	}

	return nil
}

func shuffle(positions []engine.PackedPos) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	rng.Shuffle(len(positions), func(i, j int) {
		positions[i], positions[j] = positions[j], positions[i]
	})
}
