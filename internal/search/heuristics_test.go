package search

import (
	"testing"

	"github.com/Skiddle-Labs/axon-engine/internal/engine"
)

// TestCaptureHistory verifies that the CaptureHistory table is populated during search.
func TestCaptureHistory(t *testing.T) {
	b := engine.NewBoard()
	// Position with multiple captures to ensure CaptureHistory is exercised.
	b.SetFEN("r1bqkbnr/pppp1ppp/2n5/4p3/3P4/5N2/PPP1PPPP/RNBQKB1R w KQkq - 0 1")

	GlobalTT.Clear()
	e := NewEngine(b)

	// Depth enough to trigger cutoffs and history updates.
	e.Search(6)

	found := false
Loop:
	for c := 0; c < 2; c++ {
		for pt := 0; pt < 7; pt++ {
			for cpt := 0; cpt < 7; cpt++ {
				for sq := 0; sq < 64; sq++ {
					if e.CaptureHistory[c][pt][cpt][sq] != 0 {
						found = true
						break Loop
					}
				}
			}
		}
	}

	if !found {
		t.Error("CaptureHistory was not updated during search")
	}
}

// TestCorrectionHistory verifies that the CorrectionTable is populated during search.
// Correction history learns from search results to correct static evaluation errors.
func TestCorrectionHistory(t *testing.T) {
	b := engine.NewBoard()
	b.SetFEN(engine.StartFEN)

	GlobalTT.Clear()
	e := NewEngine(b)

	// Run search. Correction history updates usually happen on TT stores at non-PV nodes.
	e.Search(8)

	found := false
Loop:
	for c := 0; c < 2; c++ {
		for i := 0; i < 16384; i++ {
			if e.CorrectionTable[c][i] != 0 {
				found = true
				break Loop
			}
		}
	}

	if !found {
		t.Error("CorrectionTable was not updated during search")
	}
}

// TestHistoryTable verifies that the main HistoryTable is populated.
func TestHistoryTable(t *testing.T) {
	b := engine.NewBoard()
	b.SetFEN(engine.StartFEN)

	GlobalTT.Clear()
	e := NewEngine(b)
	e.Search(6)

	found := false
Loop:
	for c := 0; c < 2; c++ {
		for pt := 0; pt < 7; pt++ {
			for sq := 0; sq < 64; sq++ {
				if e.HistoryTable[c][pt][sq] != 0 {
					found = true
					break Loop
				}
			}
		}
	}

	if !found {
		t.Error("HistoryTable was not updated during search")
	}
}
