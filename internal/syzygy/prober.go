//go:build cgo
// +build cgo

package syzygy

/*
#include "fathom.h"
#include <stdlib.h>
#include <stdbool.h>
*/
import "C"

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/personal-github/axon-engine/internal/engine"
)

var (
	initOnce sync.Once
	isInit   bool
	mu       sync.Mutex
)

// Init initializes the Fathom prober with the given path to Syzygy files (.rtbw and .rtbz).
func Init(path string) error {
	var err error
	initOnce.Do(func() {
		if path == "" || path == "<none>" {
			return
		}

		cPath := C.CString(path)
		defer C.free(unsafe.Pointer(cPath))

		if bool(C.fathom_init(cPath)) {
			isInit = true
		} else {
			err = fmt.Errorf("failed to initialize Syzygy tablebases at: %s", path)
		}
	})
	return err
}

// IsInitialized returns true if the tablebases are successfully loaded.
func IsInitialized() bool {
	return isInit
}

// ProbeWDL probes the Win-Draw-Loss value for a board position.
// Returns the WDL value and true if the position was found in the tablebases.
func ProbeWDL(b *engine.Board) (int, bool) {
	if !isInit {
		return WDLNotFound, false
	}

	// Syzygy tablebases do not support positions where castling is possible.
	if b.Castling != 0 {
		return WDLNotFound, false
	}

	// Only probe if piece count is within Syzygy range (typically up to 6 or 7).
	pieceCount := b.Colors[engine.White].Count() + b.Colors[engine.Black].Count()
	if pieceCount > 7 {
		return WDLNotFound, false
	}

	side := C.uint(0)
	if b.SideToMove == engine.Black {
		side = 1
	}

	// We use a mutex to ensure thread safety when calling the underlying C library.
	mu.Lock()
	res := C.fathom_probe_wdl(
		C.ulonglong(b.Colors[engine.White]),
		C.ulonglong(b.Colors[engine.Black]),
		C.ulonglong(b.Pieces[engine.White][engine.King]|b.Pieces[engine.Black][engine.King]),
		C.ulonglong(b.Pieces[engine.White][engine.Queen]|b.Pieces[engine.Black][engine.Queen]),
		C.ulonglong(b.Pieces[engine.White][engine.Rook]|b.Pieces[engine.Black][engine.Rook]),
		C.ulonglong(b.Pieces[engine.White][engine.Bishop]|b.Pieces[engine.Black][engine.Bishop]),
		C.ulonglong(b.Pieces[engine.White][engine.Knight]|b.Pieces[engine.Black][engine.Knight]),
		C.ulonglong(b.Pieces[engine.White][engine.Pawn]|b.Pieces[engine.Black][engine.Pawn]),
		C.uint(b.EnPassant),
		C.uint(b.Castling),
		side,
	)
	mu.Unlock()

	// Fathom WDL results: 0=Loss, 1=Blessed Loss, 2=Draw, 3=Cursed Win, 4=Win
	// Mapping to internal Axon WDL constants:
	switch res {
	case 4:
		return WDLWin, true
	case 3:
		return WDLCursed, true // Win truncated to draw by 50-move rule
	case 2:
		return WDLDraw, true
	case 1:
		return WDLBlessed, true // Loss saved to draw by 50-move rule
	case 0:
		return WDLLoss, true
	default:
		return WDLNotFound, false
	}
}

// ProbeDTZ probes the Distance-To-Zero value for a board position.
// Useful for root moves and generating perfect endgame play.
func ProbeDTZ(b *engine.Board) (int, bool) {
	if !isInit {
		return 0, false
	}

	if b.Castling != 0 {
		return 0, false
	}

	side := C.uint(0)
	if b.SideToMove == engine.Black {
		side = 1
	}

	mu.Lock()
	res := C.fathom_probe_dtz(
		C.ulonglong(b.Colors[engine.White]),
		C.ulonglong(b.Colors[engine.Black]),
		C.ulonglong(b.Pieces[engine.White][engine.King]|b.Pieces[engine.Black][engine.King]),
		C.ulonglong(b.Pieces[engine.White][engine.Queen]|b.Pieces[engine.Black][engine.Queen]),
		C.ulonglong(b.Pieces[engine.White][engine.Rook]|b.Pieces[engine.Black][engine.Rook]),
		C.ulonglong(b.Pieces[engine.White][engine.Bishop]|b.Pieces[engine.Black][engine.Bishop]),
		C.ulonglong(b.Pieces[engine.White][engine.Knight]|b.Pieces[engine.Black][engine.Knight]),
		C.ulonglong(b.Pieces[engine.White][engine.Pawn]|b.Pieces[engine.Black][engine.Pawn]),
		C.uint(b.EnPassant),
		C.uint(b.Castling),
		side,
	)
	mu.Unlock()

	if res < 0 {
		return 0, false
	}

	return int(res), true
}
