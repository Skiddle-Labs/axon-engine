package search

import (
	"sync"
	"sync/atomic"

	"github.com/personal-github/axon-engine/internal/engine"
)

// WDL (Win-Draw-Loss) results from Syzygy tablebases.
const (
	WDLWin      = 2
	WDLBlessed  = 1
	WDLDraw     = 0
	WDLCursed   = -1
	WDLLoss     = -2
	WDLNotFound = -3
)

// Tablebase represents a Syzygy tablebase prober.
type Tablebase struct {
	mu        sync.RWMutex
	path      string
	maxPieces int
	active    int32 // Atomic boolean
}

// GlobalTB is the global instance for tablebase probing.
var GlobalTB *Tablebase

// NewTablebase creates a new Tablebase instance and initializes probing.
func NewTablebase(path string) *Tablebase {
	if path == "" || path == "<none>" {
		return nil
	}

	tb := &Tablebase{
		path: path,
	}

	// In a real implementation, we would load the .rtbw and .rtbz files here.
	// For now, this is a placeholder for search integration.
	atomic.StoreInt32(&tb.active, 1)

	return tb
}

// ProbeWDL probes the Win-Draw-Loss status of the current position.
func (tb *Tablebase) ProbeWDL(b *engine.Board) (int, bool) {
	if tb == nil || atomic.LoadInt32(&tb.active) == 0 {
		return WDLNotFound, false
	}

	// Syzygy tablebases do not support positions where castling is possible.
	if b.Castling != 0 {
		return WDLNotFound, false
	}

	// We only probe if the piece count is within the tablebase range.
	pieceCount := b.Colors[engine.White].Count() + b.Colors[engine.Black].Count()

	if pieceCount > 6 { // Most common Syzygy sets are up to 6 pieces
		return WDLNotFound, false
	}

	// TODO: Integrate actual Syzygy probing logic using a Go library (like niklasf/syzygy)
	// or a C wrapper for Fathom.
	//
	// Probing logic would go here.

	return WDLNotFound, false
}

// SyzygyScore returns a search score based on the WDL value.
// It maps Syzygy wins/losses to scores just below the mate range to ensure
// the engine prefers a direct checkmate over a tablebase win where possible.
func SyzygyScore(wdl int, ply int) int {
	switch wdl {
	case WDLWin:
		return MateScore - 1000 - ply
	case WDLBlessed:
		return 0 // Treat blessed draw as draw
	case WDLCursed:
		return 0 // Treat cursed draw as draw
	case WDLLoss:
		return -MateScore + 1000 + ply
	default:
		return 0
	}
}

// Close releases any resources used by the tablebase prober.
func (tb *Tablebase) Close() {
	if tb == nil {
		return
	}
	tb.mu.Lock()
	defer tb.mu.Unlock()
	atomic.StoreInt32(&tb.active, 0)
}
