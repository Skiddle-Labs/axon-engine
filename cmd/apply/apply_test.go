package main

import (
	"os"
	"strings"
	"testing"
)

func TestUpdateParams(t *testing.T) {
	content := `package eval

var PawnMG = 85
var (
	KnightMG = 349
	Unchanged = 42
)

var SafetyTable = [10]int{
	1, 2, 3, 4, 5,
	6, 7, 8, 9, 10,
}

var MgPST = [7][64]int{
	engine.Pawn: {
		0, 1, 2, 3,
	},
	engine.Knight: {
		10, 20, 30, 40,
	},
}

var EgPST = [7][64]int{
	engine.Pawn: {
		5, 6, 7, 8,
	},
}`

	tunedParams := map[string]int{
		"PawnMG":           100,
		"KnightMG":         400,
		"SafetyTable[0]":   11,
		"SafetyTable[5]":   66,
		"MgPST[Pawn][1]":   111,
		"MgPST[Pawn][3]":   333,
		"MgPST[Knight][0]": 99,
		"EgPST[Pawn][2]":   77,
	}

	newContent := updateParams(content, tunedParams)

	// Check Scalars
	if !strings.Contains(newContent, "var PawnMG = 100") {
		t.Errorf("Expected var PawnMG = 100, not found or incorrect")
	}
	if !strings.Contains(newContent, "KnightMG = 400") {
		t.Errorf("Expected KnightMG = 400, not found or incorrect")
	}
	if !strings.Contains(newContent, "Unchanged = 42") {
		t.Errorf("Expected Unchanged = 42 to be preserved")
	}

	// Check SafetyTable
	// Note: the applier preserves commas and spacing
	if !strings.Contains(newContent, "11, 2, 3, 4, 5,") {
		t.Errorf("Expected SafetyTable[0] to be 11, content: %s", newContent)
	}
	if !strings.Contains(newContent, "66, 7, 8, 9, 10,") {
		t.Errorf("Expected SafetyTable[5] to be 66, content: %s", newContent)
	}

	// Check MgPST
	if !strings.Contains(newContent, "0, 111, 2, 333,") {
		t.Errorf("Expected MgPST[Pawn] updates, content: %s", newContent)
	}
	if !strings.Contains(newContent, "99, 20, 30, 40,") {
		t.Errorf("Expected MgPST[Knight] updates, content: %s", newContent)
	}

	// Check EgPST
	if !strings.Contains(newContent, "5, 6, 77, 8,") {
		t.Errorf("Expected EgPST[Pawn] updates, content: %s", newContent)
	}
}

func TestLoadTunedParams(t *testing.T) {
	// Create a temporary parameters file
	tmpFile := "test_params.txt"
	content := `
PawnMG = 90
KnightMG = 350
// This is a comment
InvalidLine
MgPST[Pawn][0] = -5
`
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile)

	params, err := loadTunedParams(tmpFile)
	if err != nil {
		t.Fatalf("Failed to load params: %v", err)
	}

	expected := map[string]int{
		"PawnMG":         90,
		"KnightMG":       350,
		"MgPST[Pawn][0]": -5,
	}

	if len(params) != len(expected) {
		t.Errorf("Expected %d params, got %d", len(expected), len(params))
	}

	for k, v := range expected {
		if params[k] != v {
			t.Errorf("Param %s: expected %d, got %d", k, v, params[k])
		}
	}
}
