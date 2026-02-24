package board

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
		EnPassant:     b.EnPassant,
		Castling:      b.Castling,
		HalfMoveClock: b.HalfMoveClock,
		Hash:          b.Hash,
	}

	movingPiece := b.PieceAt(from)
	movingType := movingPiece.Type()

	// Update hash for moving piece departure
	b.Hash ^= PieceKeys[movingPiece][from]

	// Remove current En Passant and Castling from hash
	if b.EnPassant != NoSquare {
		b.Hash ^= EnPassantKeys[b.EnPassant.File()]
	}
	b.Hash ^= CastlingKeys[b.Castling]

	// Clear En Passant square for the new state
	b.EnPassant = NoSquare
	b.HalfMoveClock++

	// 2. Handle captures
	if flags&CaptureFlag != 0 {
		captureSq := to
		if flags == EnPassantFlag {
			if us == White {
				captureSq = to - 8
			} else {
				captureSq = to + 8
			}
		}

		capturedPiece := b.PieceAt(captureSq)
		b.History[b.Ply].CapturedPiece = capturedPiece

		// Remove the captured piece from board and hash
		b.Pieces[them][capturedPiece.Type()].Clear(captureSq)
		b.Colors[them].Clear(captureSq)
		b.Hash ^= PieceKeys[capturedPiece][captureSq]
		b.HalfMoveClock = 0
	} else {
		b.History[b.Ply].CapturedPiece = NoPiece
	}

	// 3. Move the piece
	b.Pieces[us][movingType].Clear(from)
	b.Colors[us].Clear(from)

	if flags&0x8000 != 0 { // Promotion
		var promoType PieceType
		switch flags & 0xB000 {
		case PromoQueen:
			promoType = Queen
		case PromoRook:
			promoType = Rook
		case PromoBishop:
			promoType = Bishop
		case PromoKnight:
			promoType = Knight
		}
		promoPiece := makePiece(us, promoType)
		b.Pieces[us][promoType].Set(to)
		b.Colors[us].Set(to)
		b.Hash ^= PieceKeys[promoPiece][to]
	} else {
		b.Pieces[us][movingType].Set(to)
		b.Colors[us].Set(to)
		b.Hash ^= PieceKeys[movingPiece][to]
	}

	// 4. Handle special move rules
	if movingType == Pawn {
		b.HalfMoveClock = 0
		if flags == DoublePawnPush {
			if us == White {
				b.EnPassant = from + 8
			} else {
				b.EnPassant = from - 8
			}
			b.Hash ^= EnPassantKeys[b.EnPassant.File()]
		}
	} else if movingType == King {
		if flags == KingsideCast {
			if us == White {
				b.Pieces[White][Rook].Clear(H1)
				b.Colors[White].Clear(H1)
				b.Pieces[White][Rook].Set(F1)
				b.Colors[White].Set(F1)
				b.Hash ^= PieceKeys[WhiteRook][H1] ^ PieceKeys[WhiteRook][F1]
			} else {
				b.Pieces[Black][Rook].Clear(H8)
				b.Colors[Black].Clear(H8)
				b.Pieces[Black][Rook].Set(F8)
				b.Colors[Black].Set(F8)
				b.Hash ^= PieceKeys[BlackRook][H8] ^ PieceKeys[BlackRook][F8]
			}
		} else if flags == QueensideCast {
			if us == White {
				b.Pieces[White][Rook].Clear(A1)
				b.Colors[White].Clear(A1)
				b.Pieces[White][Rook].Set(D1)
				b.Colors[White].Set(D1)
				b.Hash ^= PieceKeys[WhiteRook][A1] ^ PieceKeys[WhiteRook][D1]
			} else {
				b.Pieces[Black][Rook].Clear(A8)
				b.Colors[Black].Clear(A8)
				b.Pieces[Black][Rook].Set(D8)
				b.Colors[Black].Set(D8)
				b.Hash ^= PieceKeys[BlackRook][A8] ^ PieceKeys[BlackRook][D8]
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
	if us == Black {
		b.FullMoveNumber++
	}
	b.Ply++

	// 7. Legality check: can't leave king in check
	kingSq := b.Pieces[us][King].LSB()
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
	them := b.SideToMove
	from := m.From()
	to := m.To()
	flags := m.Flags()

	state := b.History[b.Ply]

	// Restore board from history
	b.EnPassant = state.EnPassant
	b.Castling = state.Castling
	b.HalfMoveClock = state.HalfMoveClock
	b.Hash = state.Hash

	// 1. Move piece back
	movingPiece := b.PieceAt(to)
	movingType := movingPiece.Type()
	b.Pieces[us][movingType].Clear(to)
	b.Colors[us].Clear(to)

	originalType := movingType
	if flags&0x8000 != 0 {
		originalType = Pawn
	}
	b.Pieces[us][originalType].Set(from)
	b.Colors[us].Set(from)

	// 2. Restore captured piece
	if flags&CaptureFlag != 0 {
		captureSq := to
		if flags == EnPassantFlag {
			if us == White {
				captureSq = to - 8
			} else {
				captureSq = to + 8
			}
		}
		capturedPiece := state.CapturedPiece
		b.Pieces[them][capturedPiece.Type()].Set(captureSq)
		b.Colors[them].Set(captureSq)
	}

	// 3. Restore castling rooks
	if originalType == King {
		if flags == KingsideCast {
			if us == White {
				b.Pieces[White][Rook].Clear(F1)
				b.Colors[White].Clear(F1)
				b.Pieces[White][Rook].Set(H1)
				b.Colors[White].Set(H1)
			} else {
				b.Pieces[Black][Rook].Clear(F8)
				b.Colors[Black].Clear(F8)
				b.Pieces[Black][Rook].Set(H8)
				b.Colors[Black].Set(H8)
			}
		} else if flags == QueensideCast {
			if us == White {
				b.Pieces[White][Rook].Clear(D1)
				b.Colors[White].Clear(D1)
				b.Pieces[White][Rook].Set(A1)
				b.Colors[White].Set(A1)
			} else {
				b.Pieces[Black][Rook].Clear(D8)
				b.Colors[Black].Clear(D8)
				b.Pieces[Black][Rook].Set(A8)
				b.Colors[Black].Set(A8)
			}
		}
	}

	// 4. Reset counters
	if us == Black {
		b.FullMoveNumber--
	}
	b.SideToMove = us
}

// MakeNullMove makes a null move (passing the turn).
func (b *Board) MakeNullMove() {
	b.History[b.Ply] = State{
		EnPassant:     b.EnPassant,
		Castling:      b.Castling,
		HalfMoveClock: b.HalfMoveClock,
		Hash:          b.Hash,
		CapturedPiece: NoPiece,
	}

	if b.EnPassant != NoSquare {
		b.Hash ^= EnPassantKeys[b.EnPassant.File()]
	}

	b.EnPassant = NoSquare
	b.HalfMoveClock++
	b.SideToMove ^= 1
	b.Hash ^= SideKey

	if b.SideToMove == White { // Side that moved was Black
		b.FullMoveNumber++
	}

	b.Ply++
}

// UnmakeNullMove reverts a null move.
func (b *Board) UnmakeNullMove() {
	b.Ply--
	state := b.History[b.Ply]
	b.EnPassant = state.EnPassant
	b.Castling = state.Castling
	b.HalfMoveClock = state.HalfMoveClock
	b.Hash = state.Hash

	if b.SideToMove == White { // Side that moved was Black
		b.FullMoveNumber--
	}
	b.SideToMove ^= 1
}
