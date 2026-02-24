package engine

import (
	"encoding/binary"
	"math/rand"
	"os"
)

// Polyglot Zobrist keys as defined by the Polyglot specification.
var (
	polyglotPieceKeys  [12][64]uint64
	polyglotCastleKeys [16]uint64
	polyglotEPKeys     [8]uint64
	polyglotTurnKey    uint64
)

func init() {
	// Polyglot uses a specific LCG to generate its Zobrist keys.
	var seed uint64 = 0

	getNext := func() uint64 {
		seed = seed*6364136223846793005 + 1
		return seed
	}

	// Piece keys: 12 pieces * 64 squares
	// Order: White Pawn, Black Pawn, White Knight, Black Knight, ...
	for p := 0; p < 12; p++ {
		for s := 0; s < 64; s++ {
			polyglotPieceKeys[p][s] = getNext()
		}
	}

	// Castle keys: 16 keys for 4 bits of castling rights
	for i := 0; i < 16; i++ {
		polyglotCastleKeys[i] = getNext()
	}

	// En Passant keys: 8 files
	for i := 0; i < 8; i++ {
		polyglotEPKeys[i] = getNext()
	}

	// Turn key: Black to move
	polyglotTurnKey = getNext()
}

// PolyglotBook handles reading Polyglot .bin files.
type PolyglotBook struct {
	file *os.File
	size int64
}

// OpenBook opens a Polyglot .bin file.
func OpenBook(path string) (*PolyglotBook, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}

	return &PolyglotBook{
		file: f,
		size: info.Size(),
	}, nil
}

// Close closes the book file.
func (b *PolyglotBook) Close() error {
	if b.file != nil {
		return b.file.Close()
	}
	return nil
}

// GetMove probes the book for a move in the current position.
func (b *PolyglotBook) GetMove(board *Board) (Move, bool) {
	if b == nil || b.file == nil {
		return NoMove, false
	}

	hash := b.ComputePolyglotHash(board)
	entries := b.findEntries(hash)
	if len(entries) == 0 {
		return NoMove, false
	}

	// Simple weighted random selection
	totalWeight := uint32(0)
	for _, e := range entries {
		totalWeight += uint32(e.Weight)
	}

	if totalWeight == 0 {
		// Fallback to random move among available
		idx := rand.Intn(len(entries))
		return b.parsePolyglotMove(board, entries[idx].RawMove), true
	}

	r := uint32(rand.Intn(int(totalWeight)))
	currentWeight := uint32(0)
	for _, e := range entries {
		currentWeight += uint32(e.Weight)
		if r < currentWeight {
			return b.parsePolyglotMove(board, entries[e.Index].RawMove), true
		}
	}

	return b.parsePolyglotMove(board, entries[0].RawMove), true
}

type bookEntry struct {
	Hash    uint64
	RawMove uint16
	Weight  uint16
	Learn   uint32
	Index   int
}

func (b *PolyglotBook) findEntries(hash uint64) []bookEntry {
	numEntries := b.size / 16
	low := int64(0)
	high := numEntries - 1
	var entries []bookEntry

	for low <= high {
		mid := (low + high) / 2
		b.file.Seek(mid*16, 0)
		var entryHash uint64
		binary.Read(b.file, binary.BigEndian, &entryHash)

		if entryHash == hash {
			// Found one, look for others (they are sorted)
			curr := mid
			for curr >= 0 {
				b.file.Seek(curr*16, 0)
				var h uint64
				binary.Read(b.file, binary.BigEndian, &h)
				if h != hash {
					break
				}
				curr--
			}
			curr++
			for curr < numEntries {
				b.file.Seek(curr*16, 0)
				var h uint64
				var m, w uint16
				var l uint32
				binary.Read(b.file, binary.BigEndian, &h)
				if h != hash {
					break
				}
				binary.Read(b.file, binary.BigEndian, &m)
				binary.Read(b.file, binary.BigEndian, &w)
				binary.Read(b.file, binary.BigEndian, &l)
				entries = append(entries, bookEntry{Hash: h, RawMove: m, Weight: w, Learn: l, Index: len(entries)})
				curr++
			}
			return entries
		} else if entryHash < hash {
			low = mid + 1
		} else {
			high = mid - 1
		}
	}

	return nil
}

// ComputePolyglotHash calculates the hash according to the Polyglot spec.
func (b *PolyglotBook) ComputePolyglotHash(board *Board) uint64 {
	var hash uint64

	// Pieces
	for s := 0; s < 64; s++ {
		sq := Square(s)
		piece := board.PieceAt(sq)
		if piece != NoPiece {
			// Polyglot piece mapping: WP, BP, WN, BN, WB, BB, WR, BR, WQ, BQ, WK, BK
			pIdx := 2 * int(piece.Type()-1)
			if piece.Color() == Black {
				pIdx++
			}
			// Polyglot uses rank*8 + file where A1=0, H8=63
			hash ^= polyglotPieceKeys[pIdx][s]
		}
	}

	// Castling
	cIdx := 0
	if board.Castling&WhiteKingside != 0 {
		cIdx |= 1
	}
	if board.Castling&WhiteQueenside != 0 {
		cIdx |= 2
	}
	if board.Castling&BlackKingside != 0 {
		cIdx |= 4
	}
	if board.Castling&BlackQueenside != 0 {
		cIdx |= 8
	}
	hash ^= polyglotCastleKeys[cIdx]

	// En Passant
	// Only XOR if there is a pawn that can capture it
	if board.EnPassant != NoSquare {
		file := board.EnPassant.File()
		canCapture := false
		if board.SideToMove == White {
			rank := board.EnPassant.Rank()
			if rank == 5 { // White EP square is rank 6 (index 5)
				if file > 0 && board.PieceAt(NewSquare(file-1, 4)) == WhitePawn {
					canCapture = true
				}
				if file < 7 && board.PieceAt(NewSquare(file+1, 4)) == WhitePawn {
					canCapture = true
				}
			}
		} else {
			rank := board.EnPassant.Rank()
			if rank == 2 { // Black EP square is rank 3 (index 2)
				if file > 0 && board.PieceAt(NewSquare(file-1, 3)) == BlackPawn {
					canCapture = true
				}
				if file < 7 && board.PieceAt(NewSquare(file+1, 3)) == BlackPawn {
					canCapture = true
				}
			}
		}
		if canCapture {
			hash ^= polyglotEPKeys[file]
		}
	}

	// Turn
	if board.SideToMove == Black {
		hash ^= polyglotTurnKey
	}

	return hash
}

func (b *PolyglotBook) parsePolyglotMove(board *Board, raw uint16) Move {
	toFile := int(raw & 0x07)
	toRank := int((raw >> 3) & 0x07)
	fromFile := int((raw >> 6) & 0x07)
	fromRank := int((raw >> 9) & 0x07)
	promo := int((raw >> 12) & 0x07)

	from := NewSquare(fromFile, fromRank)
	to := NewSquare(toFile, toRank)

	// Determine flags
	flags := QuietFlag
	piece := board.PieceAt(from)
	target := board.PieceAt(to)

	if target != NoPiece {
		flags = CaptureFlag
	}

	// Special moves
	if piece.Type() == Pawn {
		if to == board.EnPassant {
			flags = EnPassantFlag
		} else if int(toRank)-int(fromRank) == 2 || int(fromRank)-int(toRank) == 2 {
			flags = DoublePawnPush
		}
	} else if piece.Type() == King {
		if fromFile-toFile == 2 {
			flags = QueensideCast
		} else if toFile-fromFile == 2 {
			flags = KingsideCast
		}
	}

	// Promotions
	if promo > 0 {
		if flags&CaptureFlag != 0 {
			switch promo {
			case 1:
				flags = PromoKnight | CaptureFlag
			case 2:
				flags = PromoBishop | CaptureFlag
			case 3:
				flags = PromoRook | CaptureFlag
			case 4:
				flags = PromoQueen | CaptureFlag
			}
		} else {
			switch promo {
			case 1:
				flags = PromoKnight
			case 2:
				flags = PromoBishop
			case 3:
				flags = PromoRook
			case 4:
				flags = PromoQueen
			}
		}
	}

	return NewMove(from, to, flags)
}
