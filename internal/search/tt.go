package search

import (
	"unsafe"

	"github.com/personal-github/axon-engine/internal/board"
)

const (
	ExactFlag uint8 = iota
	AlphaFlag
	BetaFlag
)

// TTEntry represents a single record in the transposition table.
type TTEntry struct {
	Hash  uint64     // Zobrist hash of the position
	Move  board.Move // Best move found in this position
	Score int16      // Evaluation score
	Depth int8       // Depth of the search that produced this score
	Flag  uint8      // Type of score (Exact, Alpha, or Beta)
}

// TranspositionTable is a fixed-size hash table for storing search results.
type TranspositionTable struct {
	Entries []TTEntry
	Count   uint64
}

// NewTranspositionTable allocates a new TT with the specified size in Megabytes.
func NewTranspositionTable(sizeMB int) *TranspositionTable {
	sizePerEntry := uint64(unsafe.Sizeof(TTEntry{}))
	numEntries := (uint64(sizeMB) * 1024 * 1024) / sizePerEntry

	return &TranspositionTable{
		Entries: make([]TTEntry, numEntries),
		Count:   numEntries,
	}
}

// Store saves a search result into the transposition table.
func (tt *TranspositionTable) Store(hash uint64, depth int, score int, flag uint8, move board.Move, ply int) {
	if tt.Count == 0 {
		return
	}

	index := hash % tt.Count
	entry := &tt.Entries[index]

	// Adjust mate scores to be independent of the current search depth (ply)
	// This allows the engine to recognize a mate found at different search branches.
	storedScore := score
	if storedScore > MateScore-1000 {
		storedScore += ply
	} else if storedScore < -MateScore+1000 {
		storedScore -= ply
	}

	// Replacement strategy: depth-preferred.
	// We replace the entry if the new search was deeper or if the entry is empty.
	if entry.Hash == 0 || int(entry.Depth) <= depth {
		entry.Hash = hash
		entry.Move = move
		entry.Score = int16(storedScore)
		entry.Depth = int8(depth)
		entry.Flag = flag
	}
}

// Probe retrieves a search result from the transposition table if it exists.
// Returns the score, best move, and a boolean indicating if a valid cut-off score was found.
func (tt *TranspositionTable) Probe(hash uint64, depth int, alpha, beta int, ply int) (int, board.Move, bool) {
	if tt.Count == 0 {
		return 0, board.NoMove, false
	}

	index := hash % tt.Count
	entry := tt.Entries[index]

	if entry.Hash == hash {
		score := int(entry.Score)

		// Adjust mate scores back to be relative to the current search depth
		if score > MateScore-1000 {
			score -= ply
		} else if score < -MateScore+1000 {
			score += ply
		}

		// If the stored search was at least as deep as the current one,
		// we might be able to return a cut-off score.
		if int(entry.Depth) >= depth {
			if entry.Flag == ExactFlag {
				return score, entry.Move, true
			}
			if entry.Flag == AlphaFlag && score <= alpha {
				return score, entry.Move, true
			}
			if entry.Flag == BetaFlag && score >= beta {
				return score, entry.Move, true
			}
		}

		// Even if the depth is insufficient for a cut-off, we return the move
		// to use it for move ordering (TT move should be tried first).
		return score, entry.Move, false
	}

	return 0, board.NoMove, false
}

// Clear wipes all entries from the transposition table.
func (tt *TranspositionTable) Clear() {
	for i := range tt.Entries {
		tt.Entries[i] = TTEntry{}
	}
}
