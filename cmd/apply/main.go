package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Axon Parameter Applier")
		fmt.Println("Usage: go run cmd/apply/main.go <tuned_params.txt>")
		return
	}

	tunedFilePath := os.Args[1]
	paramsFilePath := "internal/eval/params.go"

	// 1. Load tuned parameters into a map
	tunedParams, err := loadTunedParams(tunedFilePath)
	if err != nil {
		fmt.Printf("Error loading tuned params: %v\n", err)
		return
	}
	fmt.Printf("Loaded %d parameters from %s\n", len(tunedParams), tunedFilePath)

	// 2. Read the params.go file
	content, err := os.ReadFile(paramsFilePath)
	if err != nil {
		fmt.Printf("Error reading %s: %v\n", paramsFilePath, err)
		return
	}

	// 3. Update the content
	newContent := updateParams(string(content), tunedParams)

	// 4. Write back to params.go
	err = os.WriteFile(paramsFilePath, []byte(newContent), 0644)
	if err != nil {
		fmt.Printf("Error writing to %s: %v\n", paramsFilePath, err)
		return
	}

	fmt.Println("Successfully applied tuned parameters to internal/eval/params.go")
	fmt.Println("Don't forget to rebuild the engine with: go build -o axon.exe .")
}

func loadTunedParams(path string) (map[string]int, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	params := make(map[string]int)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		parts := strings.Split(line, "=")
		if len(parts) != 2 {
			continue
		}

		name := strings.TrimSpace(parts[0])
		valStr := strings.TrimSpace(parts[1])
		val, err := strconv.Atoi(valStr)
		if err != nil {
			continue
		}

		params[name] = val
	}

	return params, scanner.Err()
}

func updateParams(content string, tunedParams map[string]int) string {
	lines := strings.Split(content, "\n")
	updatedCount := 0

	// Regex to match: Name = Value
	// Handles scalars: PawnMG = 85 or var PawnMG = 85
	scalarRegex := regexp.MustCompile(`^(\s*(?:var\s+)?)([A-Za-z0-9_]+)(\s*=\s*)(-?[0-9]+)(.*)$`)

	// Regex to match PST/Table entries: value, value, value
	// We need to be careful here because PSTs are multi-line arrays.
	// We use a simple line-by-line replacement for PST indices like MgPST[Pawn][0]

	// Because PSTs in params.go are formatted as:
	// engine.Pawn: {
	//    0, 0, 0, ...
	// }
	// The tuner names them as MgPST[Pawn][0].

	type pstContext struct {
		prefix string // "MgPST" or "EgPST" or "SafetyTable"
		piece  string // "Pawn", "Knight", etc.
		index  int
	}

	var ctx pstContext

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect table starts
		if strings.HasPrefix(trimmed, "var MgPST") {
			ctx.prefix = "MgPST"
			continue
		} else if strings.HasPrefix(trimmed, "var EgPST") {
			ctx.prefix = "EgPST"
			continue
		} else if strings.HasPrefix(trimmed, "var SafetyTable") {
			ctx.prefix = "SafetyTable"
			ctx.index = 0
			continue
		}

		// Detect Piece sub-blocks in PST
		if ctx.prefix != "" && strings.Contains(trimmed, "engine.") {
			pieceName := strings.TrimSuffix(strings.TrimPrefix(trimmed, "engine."), ": {")
			ctx.piece = pieceName
			ctx.index = 0
			continue
		}

		// Close contexts
		if trimmed == "}," || trimmed == "}" {
			ctx.piece = ""
			if trimmed == "}" {
				ctx.prefix = ""
			}
			continue
		}

		// Handle Scalar replacements
		if ctx.prefix == "" {
			matches := scalarRegex.FindStringSubmatch(line)
			if len(matches) > 0 {
				name := matches[2]
				if newVal, exists := tunedParams[name]; exists {
					lines[i] = fmt.Sprintf("%s%s%s%d%s", matches[1], name, matches[3], newVal, matches[5])
					updatedCount++
				}
				continue
			}
		}

		// Handle Array/Table entry replacements
		if ctx.prefix != "" {
			// Find all numbers in the line
			parts := strings.Split(line, ",")
			for j, part := range parts {
				valTrim := strings.TrimSpace(part)
				if valTrim == "" || valTrim == "{" || valTrim == "}" {
					continue
				}

				// Verify it's a number
				if _, err := strconv.Atoi(valTrim); err != nil {
					continue
				}

				paramName := ""
				if ctx.prefix == "SafetyTable" {
					paramName = fmt.Sprintf("SafetyTable[%d]", ctx.index)
				} else {
					paramName = fmt.Sprintf("%s[%s][%d]", ctx.prefix, ctx.piece, ctx.index)
				}

				if newVal, exists := tunedParams[paramName]; exists {
					// Replace the number while preserving spacing
					parts[j] = strings.Replace(part, valTrim, strconv.Itoa(newVal), 1)
					updatedCount++
				}
				ctx.index++
			}
			lines[i] = strings.Join(parts, ",")
		}
	}

	fmt.Printf("Updated %d values in source code.\n", updatedCount)
	return strings.Join(lines, "\n")
}
