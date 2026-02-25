package search

import "math"

// lmrTable stores precomputed Late Move Reduction values.
// We use a 64x64 table to cover most practical search depths and move counts.
var (
	lmrTable   [64][64]int
	LMRBase    = 0.75
	LMRDivisor = 2.25
)

func init() {
	UpdateLMR(LMRBase, LMRDivisor)
}

// UpdateLMR recomputes the LMR table with new scaling parameters.
func UpdateLMR(base, divisor float64) {
	LMRBase = base
	LMRDivisor = divisor
	for d := 1; d < 64; d++ {
		for i := 1; i < 64; i++ {
			// Standard LMR formula: R = base + ln(depth) * ln(move_index) / divisor
			reduction := LMRBase + math.Log(float64(d))*math.Log(float64(i))/LMRDivisor
			lmrTable[d][i] = int(reduction)
		}
	}
}

// getReduction returns the precomputed reduction value for a given depth and move index.
// moveIndex should be the 0-based index of the move in the ordered move list.
func getReduction(depth, moveIndex int) int {
	if depth < 0 {
		depth = 0
	}
	if depth >= 64 {
		depth = 63
	}
	if moveIndex < 0 {
		moveIndex = 0
	}
	if moveIndex >= 64 {
		moveIndex = 63
	}
	return lmrTable[depth][moveIndex]
}
