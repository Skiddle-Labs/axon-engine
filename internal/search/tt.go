package search

import (
	"sync/atomic"
	"unsafe"

	"github.com/personal-github/axon-engine/internal/engine"
)

const (
	ExactFlag uint8 = iota
	AlphaFlag
	BetaFlag
)

// ttEntry represents a single record in the transposition table.
// We use a 128-bit structure split into two 64-bit atomic values to be lockless.
type ttEntry struct {
	hash uint64
	data uint64
}

// TranspositionTable is a fixed-size hash table for storing search results.
type TranspositionTable struct {
	entries []ttEntry
	mask    uint64
}

// NewTranspositionTable allocates a new TT with the specified size in Megabytes.
func NewTranspositionTable(sizeMB int) *TranspositionTable {
	sizePerEntry := uint64(unsafe.Sizeof(ttEntry{}))
	totalEntries := (uint64(sizeMB) * 1024 * 1024) / sizePerEntry

	// Ensure totalEntries is a power of 2 for fast masking
	count := uint64(1)
	for count*2 <= totalEntries {
		count *= 2
	}

	return &TranspositionTable{
		entries: make([]ttEntry, count),
		mask:    count - 1,
	}
}

// Store saves a search result into the transposition table.
func (tt *TranspositionTable) Store(hash uint64, depth int, score int, flag uint8, move engine.Move, ply int) {
	if tt == nil || len(tt.entries) == 0 {
		return
	}

	index := hash & tt.mask
	entry := &tt.entries[index]

	// Adjust mate scores to be independent of the current search depth (ply)
	storedScore := score
	if storedScore > MateScore-1000 {
		storedScore += ply
	} else if storedScore < -MateScore+1000 {
		storedScore -= ply
	}

	// Pack data into 64 bits:
	// Move (16 bits) | Score (16 bits) | Depth (8 bits) | Flag (8 bits) | Age (8 bits) | Reserved (8 bits)
	packedData := uint64(move) | (uint64(uint16(storedScore)) << 16) | (uint64(uint8(depth)) << 32) | (uint64(flag) << 40)

	// Replacement Strategy: Depth-Preferred
	// Note: In a lockless TT, we load the old values first.
	oldHash := atomic.LoadUint64(&entry.hash)
	oldData := atomic.LoadUint64(&entry.data)
	oldDepth := int8(oldData >> 32)

	if oldHash == 0 || oldHash != hash || int(oldDepth) <= depth {
		// To maintain consistency in a lockless environment:
		// 1. Store the data first
		// 2. Store the hash second (this acts as the commitment)
		atomic.StoreUint64(&entry.data, packedData)
		atomic.StoreUint64(&entry.hash, hash)
	}
}

// Probe retrieves a search result from the transposition table if it exists.
// Returns the score, best move, and a boolean indicating if a valid cut-off score was found.
func (tt *TranspositionTable) Probe(hash uint64, depth int, alpha, beta int, ply int) (int, engine.Move, bool) {
	if tt == nil || len(tt.entries) == 0 {
		return 0, engine.NoMove, false
	}

	index := hash & tt.mask
	entry := &tt.entries[index]

	// Lockless Probe Protocol:
	// 1. Load the hash
	// 2. If hash matches, load the data
	// 3. Load the hash again to verify it hasn't been overwritten during the data load
	eHash := atomic.LoadUint64(&entry.hash)
	if eHash != hash {
		return 0, engine.NoMove, false
	}

	eData := atomic.LoadUint64(&entry.data)
	if atomic.LoadUint64(&entry.hash) != hash {
		return 0, engine.NoMove, false
	}

	// Unpack data
	m := engine.Move(eData & 0xFFFF)
	score := int(int16(eData >> 16))
	eDepth := int(int8(eData >> 32))
	flag := uint8(eData >> 40)

	// Adjust mate scores back to be relative to the current search depth
	if score > MateScore-1000 {
		score -= ply
	} else if score < -MateScore+1000 {
		score += ply
	}

	// If the stored search was at least as deep as the current one,
	// we might be able to return a cut-off score.
	if eDepth >= depth {
		if flag == ExactFlag {
			return score, m, true
		}
		if flag == AlphaFlag && score <= alpha {
			return score, m, true
		}
		if flag == BetaFlag && score >= beta {
			return score, m, true
		}
	}

	// Even if depth is insufficient for a cut-off, return the move for ordering.
	return score, m, false
}

// HashFull returns the percentage of the transposition table that is occupied, in permille (0 to 1000).
func (tt *TranspositionTable) HashFull() int {
	if tt == nil || len(tt.entries) == 0 {
		return 0
	}

	used := 0
	sampleSize := 1000
	if len(tt.entries) < sampleSize {
		sampleSize = len(tt.entries)
	}

	for i := 0; i < sampleSize; i++ {
		if atomic.LoadUint64(&tt.entries[i].hash) != 0 {
			used++
		}
	}

	return (used * 1000) / sampleSize
}

// Clear wipes all entries from the transposition table.
func (tt *TranspositionTable) Clear() {
	if tt == nil {
		return
	}
	for i := range tt.entries {
		atomic.StoreUint64(&tt.entries[i].hash, 0)
		atomic.StoreUint64(&tt.entries[i].data, 0)
	}
}
