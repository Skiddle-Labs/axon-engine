package eval

import (
	"sync/atomic"
)

// PawnEntry represents a cached evaluation of a specific pawn structure.
type PawnEntry struct {
	Hash    uint64
	MgScore [2]int // [White, Black]
	EgScore [2]int // [White, Black]
}

// PawnTable is a specialized hash table for caching pawn structure evaluations.
// Since pawn structures change infrequently compared to piece movements,
// caching these results significantly boosts NPS.
type PawnTable struct {
	entries []PawnEntry
	mask    uint64
}

// GlobalPawnTable is the shared cache used by the evaluation engine.
// A default size of 16384 entries provides a good balance between hit rate and memory.
var GlobalPawnTable = NewPawnTable(16384)

// NewPawnTable creates a new PawnTable with the specified number of entries.
// The size is rounded down to the nearest power of two.
func NewPawnTable(size int) *PawnTable {
	if size < 1 {
		size = 1
	}

	// Ensure size is a power of 2
	count := 1
	for count*2 <= size {
		count *= 2
	}

	return &PawnTable{
		entries: make([]PawnEntry, count),
		mask:    uint64(count - 1),
	}
}

// Probe retrieves a cached pawn evaluation if it exists.
func (pt *PawnTable) Probe(hash uint64) (*PawnEntry, bool) {
	index := hash & pt.mask
	entry := &pt.entries[index]

	// Use atomic load for thread safety in multi-threaded search
	if atomic.LoadUint64(&entry.Hash) == hash {
		return entry, true
	}
	return nil, false
}

// Store saves a pawn evaluation into the table.
func (pt *PawnTable) Store(hash uint64, mgW, egW, mgB, egB int) {
	index := hash & pt.mask
	entry := &pt.entries[index]

	// Overwrite strategy: always replace
	entry.MgScore[0] = mgW
	entry.EgScore[0] = egW
	entry.MgScore[1] = mgB
	entry.EgScore[1] = egB

	// Set the hash last to act as an atomic commitment
	atomic.StoreUint64(&entry.Hash, hash)
}

// Clear wipes all entries from the pawn table.
func (pt *PawnTable) Clear() {
	for i := range pt.entries {
		atomic.StoreUint64(&pt.entries[i].Hash, 0)
	}
}

// Size returns the number of entries in the table.
func (pt *PawnTable) Size() int {
	return len(pt.entries)
}
