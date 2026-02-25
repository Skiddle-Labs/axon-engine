package engine

import (
	"github.com/Skiddle-Labs/axon-engine/internal/nnue"
	"github.com/Skiddle-Labs/axon-engine/internal/types"
)

var castlingMask = [64]CastlingRights{
	13, 15, 15, 15, 12, 15, 15, 14,
	15, 15, 15, 15, 15, 15, 15, 15,
	15, 15, 15, 15, 15, 15, 15, 15,
	15, 15, 15, 15, 15, 15, 15, 15,
	15, 15, 15, 15, 15, 15, 15, 15,
	15, 15, 15, 15, 15, 15, 15, 15,
	15, 15, 15, 15, 15, 15, 15, 15,
	7, 15, 15, 15, 3, 15, 15, 11,
}

func (b *Board) addPiece(sq types.Square, p types.Piece) {
	c := p.Color()
	pt := p.Type()
	b.Pieces[c][pt].Set(sq)
	b.Colors[c].Set(sq)

	if nnue.CurrentNetwork == nil {
		return
	}

	// Perspectives: White (W) and Black (B)
	idxW := nnue.GetFeatureIndex(p, sq)
	idxB := nnue.GetFeatureIndex(p.FlippedColor(), sq.Flipped())

	for i := 0; i < types.L1Size; i++ {
		b.Accumulators[0][i] += nnue.CurrentNetwork.FeatureWeights[idxW][i]
		b.Accumulators[1][i] += nnue.CurrentNetwork.FeatureWeights[idxB][i]
	}
}

func (b *Board) removePiece(sq types.Square, p types.Piece) {
	c := p.Color()
	pt := p.Type()
	b.Pieces[c][pt].Clear(sq)
	b.Colors[c].Clear(sq)

	if nnue.CurrentNetwork == nil {
		return
	}

	// Perspectives: White (W) and Black (B)
	idxW := nnue.GetFeatureIndex(p, sq)
	idxB := nnue.GetFeatureIndex(p.FlippedColor(), sq.Flipped())

	for i := 0; i < types.L1Size; i++ {
		b.Accumulators[0][i] -= nnue.CurrentNetwork.FeatureWeights[idxW][i]
		b.Accumulators[1][i] -= nnue.CurrentNetwork.FeatureWeights[idxB][i]
	}
}

// MakeMove updates the board state with the given move.
// It returns true if the move is legal (doesn't leave the king in check).
func (b *Board) MakeMove(m Move) bool {
	us := b.SideToMove
	them := us ^ 1
	from := m.From()
	to := m.To()
	flags := m.Flags()

	// 1. Save current state to history
	b.History[b.Ply] = State{
		Move:          m,
		EnPassant:     b.EnPassant,
		Castling:      b.Castling,
		HalfMoveClock: b.HalfMoveClock,
		Hash:          b.Hash,
		PawnHash:      b.PawnHash,
		Accumulators:  b.Accumulators,
	}

	movingPiece := b.PieceAt(from)
	movingType := movingPiece.Type()

	// Update hash for moving piece departure
	b.Hash ^= PieceKeys[movingPiece][from]
	if movingType == types.Pawn {
		b.PawnHash ^= PieceKeys[movingPiece][from]
	}

	// Remove current En Passant and Castling from hash
	if b.EnPassant != types.NoSquare {
		b.Hash ^= EnPassantKeys[b.EnPassant.File()]
	}

	// Clear En Passant square for the new state
	b.EnPassant = types.NoSquare
	b.HalfMoveClock++

	// 2. Handle captures
	if flags&CaptureFlag != 0 {
		captureSq := to
		if flags == EnPassantFlag {
			if us == types.White {
				captureSq = to - 8
			} else {
				captureSq = to + 8
			}
		}

		capturedPiece := b.PieceAt(captureSq)
		b.History[b.Ply].CapturedPiece = capturedPiece

		// Remove the captured piece from board and hash
		b.removePiece(captureSq, capturedPiece)
		b.Hash ^= PieceKeys[capturedPiece][captureSq]
		if capturedPiece.Type() == types.Pawn {
			b.PawnHash ^= PieceKeys[capturedPiece][captureSq]
		}
		b.HalfMoveClock = 0
	} else {
		b.History[b.Ply].CapturedPiece = types.NoPiece
	}

	// 3. Move the piece
	b.removePiece(from, movingPiece)

	if flags&0x8000 != 0 { // Promotion
		var promoType types.PieceType
		switch flags & 0xB000 {
		case PromoQueen:
			promoType = types.Queen
		case PromoRook:
			promoType = types.Rook
		case PromoBishop:
			promoType = types.Bishop
		case PromoKnight:
			promoType = types.Knight
		}
		promoPiece := makePiece(us, promoType)
		b.addPiece(to, promoPiece)
		b.Hash ^= PieceKeys[promoPiece][to]
	} else {
		b.addPiece(to, movingPiece)
		b.Hash ^= PieceKeys[movingPiece][to]
		if movingType == types.Pawn {
			b.PawnHash ^= PieceKeys[movingPiece][to]
		}
	}

	// 4. Handle special move rules
	if movingType == types.Pawn {
		b.HalfMoveClock = 0
		if flags == DoublePawnPush {
			if us == types.White {
				b.EnPassant = from + 8
			} else {
				b.EnPassant = from - 8
			}
			b.Hash ^= EnPassantKeys[b.EnPassant.File()]
		}
	} else if movingType == types.King {
		if flags == KingsideCast {
			if us == types.White {
				b.removePiece(types.H1, types.WhiteRook)
				b.addPiece(types.F1, types.WhiteRook)
				b.Hash ^= PieceKeys[types.WhiteRook][types.H1] ^ PieceKeys[types.WhiteRook][types.F1]
			} else {
				b.removePiece(types.H8, types.BlackRook)
				b.addPiece(types.F8, types.BlackRook)
				b.Hash ^= PieceKeys[types.BlackRook][types.H8] ^ PieceKeys[types.BlackRook][types.F8]
			}
		} else if flags == QueensideCast {
			if us == types.White {
				b.removePiece(types.A1, types.WhiteRook)
				b.addPiece(types.D1, types.WhiteRook)
				b.Hash ^= PieceKeys[types.WhiteRook][types.A1] ^ PieceKeys[types.WhiteRook][types.D1]
			} else {
				b.removePiece(types.A8, types.BlackRook)
				b.addPiece(types.D8, types.BlackRook)
				b.Hash ^= PieceKeys[types.BlackRook][types.A8] ^ PieceKeys[types.BlackRook][types.D8]
			}
		}
	}

	// 5. Update castling rights
	b.Castling &= castlingMask[from]
	b.Castling &= castlingMask[to]
	b.Hash ^= CastlingKeys[b.Castling]

	// 6. Update side to move and final bits
	b.SideToMove = them
	b.Hash ^= SideKey
	if us == types.Black {
		b.FullMoveNumber++
	}
	b.Ply++

	// 7. Legality check: can't leave king in check
	kingSq := b.Pieces[us][types.King].LSB()
	if b.IsSquareAttacked(kingSq, them) {
		b.UnmakeMove(m)
		return false
	}

	return true
}

// UnmakeMove reverts the board state to before the given move was made.
func (b *Board) UnmakeMove(m Move) {
	b.Ply--
	us := b.SideToMove ^ 1 // Side that made the move
	from := m.From()
	to := m.To()
	flags := m.Flags()

	state := b.History[b.Ply]

	// Restore board from history
	b.EnPassant = state.EnPassant
	b.Castling = state.Castling
	b.HalfMoveClock = state.HalfMoveClock
	b.Hash = state.Hash
	b.PawnHash = state.PawnHash
	b.Accumulators = state.Accumulators

	// 1. Move piece back
	movingPiece := b.PieceAt(to)
	b.removePiece(to, movingPiece)

	originalType := movingPiece.Type()
	if flags&0x8000 != 0 {
		originalType = types.Pawn
	}
	originalPiece := makePiece(us, originalType)
	b.addPiece(from, originalPiece)

	// 2. Restore captured piece
	if flags&CaptureFlag != 0 {
		captureSq := to
		if flags == EnPassantFlag {
			if us == types.White {
				captureSq = to - 8
			} else {
				captureSq = to + 8
			}
		}
		capturedPiece := state.CapturedPiece
		b.addPiece(captureSq, capturedPiece)
	}

	// 3. Restore castling rooks
	if originalType == types.King {
		if flags == KingsideCast {
			if us == types.White {
				b.removePiece(types.F1, types.WhiteRook)
				b.addPiece(types.H1, types.WhiteRook)
			} else {
				b.removePiece(types.F8, types.BlackRook)
				b.addPiece(types.H8, types.BlackRook)
			}
		} else if flags == QueensideCast {
			if us == types.White {
				b.removePiece(types.D1, types.WhiteRook)
				b.addPiece(types.A1, types.WhiteRook)
			} else {
				b.removePiece(types.D8, types.BlackRook)
				b.addPiece(types.A8, types.BlackRook)
			}
		}
	}

	// 4. Reset counters
	if us == types.Black {
		b.FullMoveNumber--
	}
	b.SideToMove = us
}

// MakeNullMove makes a null move (passing the turn).
func (b *Board) MakeNullMove() {
	b.History[b.Ply] = State{
		Move:          NoMove,
		EnPassant:     b.EnPassant,
		Castling:      b.Castling,
		HalfMoveClock: b.HalfMoveClock,
		Hash:          b.Hash,
		PawnHash:      b.PawnHash,
		CapturedPiece: types.NoPiece,
		Accumulators:  b.Accumulators,
	}

	if b.EnPassant != types.NoSquare {
		b.Hash ^= EnPassantKeys[b.EnPassant.File()]
	}

	b.EnPassant = types.NoSquare
	b.HalfMoveClock++
	b.SideToMove ^= 1
	b.Hash ^= SideKey

	if b.SideToMove == types.White { // Side that moved was Black
		b.FullMoveNumber++
	}

	b.Ply++
}

// RefreshAccumulators fully recalculates the NNUE accumulators from the current board state.
func (b *Board) RefreshAccumulators() {
	b.Accumulators = [2]types.Accumulator{}
	if nnue.CurrentNetwork == nil {
		return
	}

	for i := 0; i < types.L1Size; i++ {
		b.Accumulators[0][i] = nnue.CurrentNetwork.FeatureBiases[i]
		b.Accumulators[1][i] = nnue.CurrentNetwork.FeatureBiases[i]
	}

	for c := types.White; c <= types.Black; c++ {
		for pt := types.Pawn; pt <= types.King; pt++ {
			pieces := b.Pieces[c][pt]
			for pieces != 0 {
				sq := pieces.PopLSB()

				p := makePiece(c, pt)

				// Perspectives: White (W) and Black (B)
				idxW := nnue.GetFeatureIndex(p, sq)
				idxB := nnue.GetFeatureIndex(p.FlippedColor(), sq.Flipped())

				for i := 0; i < types.L1Size; i++ {
					b.Accumulators[0][i] += nnue.CurrentNetwork.FeatureWeights[idxW][i]
					b.Accumulators[1][i] += nnue.CurrentNetwork.FeatureWeights[idxB][i]
				}
			}
		}
	}
}

// UnmakeNullMove reverts a null move.
func (b *Board) UnmakeNullMove() {
	b.Ply--
	state := b.History[b.Ply]
	b.EnPassant = state.EnPassant
	b.Castling = state.Castling
	b.HalfMoveClock = state.HalfMoveClock
	b.Hash = state.Hash
	b.PawnHash = state.PawnHash

	if b.SideToMove == types.White { // Side that moved was Black
		b.FullMoveNumber--
	}
	b.SideToMove ^= 1
}
