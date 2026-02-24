package engine

// GenerateMoves generates all pseudo-legal moves for the current board position.
// Pseudo-legal moves include moves that might leave the king in check.
func (b *Board) GenerateMoves() MoveList {
	ml := MoveList{}
	us := b.SideToMove
	them := us ^ 1
	occ := b.Occupancy()
	enemyOcc := b.Colors[them]

	// 1. King Moves (excluding castling for now)
	kingBB := b.Pieces[us][King]
	if !kingBB.IsEmpty() {
		from := kingBB.LSB()
		attacks := KingAttacks[from] & ^b.Colors[us]
		b.addMovesFromAttacks(&ml, from, attacks, enemyOcc)
	}

	// 2. Knight Moves
	knightBB := b.Pieces[us][Knight]
	for knightBB != 0 {
		from := knightBB.PopLSB()
		attacks := KnightAttacks[from] & ^b.Colors[us]
		b.addMovesFromAttacks(&ml, from, attacks, enemyOcc)
	}

	// 3. Bishop Moves
	bishopBB := b.Pieces[us][Bishop]
	for bishopBB != 0 {
		from := bishopBB.PopLSB()
		attacks := GetBishopAttacks(from, occ) & ^b.Colors[us]
		b.addMovesFromAttacks(&ml, from, attacks, enemyOcc)
	}

	// 4. Rook Moves
	rookBB := b.Pieces[us][Rook]
	for rookBB != 0 {
		from := rookBB.PopLSB()
		attacks := GetRookAttacks(from, occ) & ^b.Colors[us]
		b.addMovesFromAttacks(&ml, from, attacks, enemyOcc)
	}

	// 5. Queen Moves
	queenBB := b.Pieces[us][Queen]
	for queenBB != 0 {
		from := queenBB.PopLSB()
		attacks := GetQueenAttacks(from, occ) & ^b.Colors[us]
		b.addMovesFromAttacks(&ml, from, attacks, enemyOcc)
	}

	// 6. Pawn Moves
	pawns := b.Pieces[us][Pawn]
	if us == White {
		for pawns != 0 {
			from := pawns.PopLSB()

			// Single push
			to := from + 8
			if !occ.Test(to) {
				if to.Rank() == 7 {
					b.addPawnPromotionMoves(&ml, from, to, false)
				} else {
					ml.AddMove(NewMove(from, to, QuietFlag))
					// Double push
					if from.Rank() == 1 && !occ.Test(to+8) {
						ml.AddMove(NewMove(from, to+8, DoublePawnPush))
					}
				}
			}

			// Captures
			attacks := b.whitePawnAttacks(from)
			captureAttacks := attacks & enemyOcc
			for captureAttacks != 0 {
				to := captureAttacks.PopLSB()
				if to.Rank() == 7 {
					b.addPawnPromotionMoves(&ml, from, to, true)
				} else {
					ml.AddMove(NewMove(from, to, CaptureFlag))
				}
			}

			// En Passant
			if b.EnPassant != NoSquare && attacks.Test(b.EnPassant) {
				ml.AddMove(NewMove(from, b.EnPassant, EnPassantFlag))
			}
		}
	} else {
		for pawns != 0 {
			from := pawns.PopLSB()

			// Single push
			to := from - 8
			if !occ.Test(to) {
				if to.Rank() == 0 {
					b.addPawnPromotionMoves(&ml, from, to, false)
				} else {
					ml.AddMove(NewMove(from, to, QuietFlag))
					// Double push
					if from.Rank() == 6 && !occ.Test(to-8) {
						ml.AddMove(NewMove(from, to-8, DoublePawnPush))
					}
				}
			}

			// Captures
			attacks := b.blackPawnAttacks(from)
			captureAttacks := attacks & enemyOcc
			for captureAttacks != 0 {
				to := captureAttacks.PopLSB()
				if to.Rank() == 0 {
					b.addPawnPromotionMoves(&ml, from, to, true)
				} else {
					ml.AddMove(NewMove(from, to, CaptureFlag))
				}
			}

			// En Passant
			if b.EnPassant != NoSquare && attacks.Test(b.EnPassant) {
				ml.AddMove(NewMove(from, b.EnPassant, EnPassantFlag))
			}
		}
	}

	// 7. Castling
	if us == White {
		if b.Castling&WhiteKingside != 0 {
			if !occ.Test(F1) && !occ.Test(G1) {
				if !b.IsSquareAttacked(E1, Black) && !b.IsSquareAttacked(F1, Black) {
					ml.AddMove(NewMove(E1, G1, KingsideCast))
				}
			}
		}
		if b.Castling&WhiteQueenside != 0 {
			if !occ.Test(D1) && !occ.Test(C1) && !occ.Test(B1) {
				if !b.IsSquareAttacked(E1, Black) && !b.IsSquareAttacked(D1, Black) {
					ml.AddMove(NewMove(E1, C1, QueensideCast))
				}
			}
		}
	} else {
		if b.Castling&BlackKingside != 0 {
			if !occ.Test(F8) && !occ.Test(G8) {
				if !b.IsSquareAttacked(E8, White) && !b.IsSquareAttacked(F8, White) {
					ml.AddMove(NewMove(E8, G8, KingsideCast))
				}
			}
		}
		if b.Castling&BlackQueenside != 0 {
			if !occ.Test(D8) && !occ.Test(C8) && !occ.Test(B8) {
				if !b.IsSquareAttacked(E8, White) && !b.IsSquareAttacked(D8, White) {
					ml.AddMove(NewMove(E8, C8, QueensideCast))
				}
			}
		}
	}

	return ml
}

func (b *Board) addPawnPromotionMoves(ml *MoveList, from, to Square, isCapture bool) {
	captureFlag := uint16(0)
	if isCapture {
		captureFlag = CaptureFlag
	}
	ml.AddMove(NewMove(from, to, PromoQueen|captureFlag))
	ml.AddMove(NewMove(from, to, PromoRook|captureFlag))
	ml.AddMove(NewMove(from, to, PromoBishop|captureFlag))
	ml.AddMove(NewMove(from, to, PromoKnight|captureFlag))
}

func (b *Board) whitePawnAttacks(sq Square) Bitboard {
	var attacks Bitboard
	bb := Bitboard(1 << sq)
	if bb&FileA == 0 {
		attacks |= bb << 7
	}
	if bb&FileH == 0 {
		attacks |= bb << 9
	}
	return attacks
}

func (b *Board) blackPawnAttacks(sq Square) Bitboard {
	var attacks Bitboard
	bb := Bitboard(1 << sq)
	if bb&FileA == 0 {
		attacks |= bb >> 9
	}
	if bb&FileH == 0 {
		attacks |= bb >> 7
	}
	return attacks
}

// addMovesFromAttacks is a helper to convert an attack bitboard into Move objects.
func (b *Board) addMovesFromAttacks(ml *MoveList, from Square, attacks Bitboard, enemyOcc Bitboard) {
	for attacks != 0 {
		to := attacks.PopLSB()
		flag := QuietFlag
		if enemyOcc.Test(to) {
			flag = CaptureFlag
		}
		ml.AddMove(NewMove(from, to, flag))
	}
}

// IsSquareAttacked returns true if the square is attacked by the given color.
func (b *Board) IsSquareAttacked(sq Square, attackerColor Color) bool {
	// Attacked by Pawns
	pawns := b.Pieces[attackerColor][Pawn]
	if attackerColor == White {
		if sq.Rank() > 0 {
			if sq.File() > 0 && pawns.Test(sq-9) {
				return true
			}
			if sq.File() < 7 && pawns.Test(sq-7) {
				return true
			}
		}
	} else {
		if sq.Rank() < 7 {
			if sq.File() < 7 && pawns.Test(sq+9) {
				return true
			}
			if sq.File() > 0 && pawns.Test(sq+7) {
				return true
			}
		}
	}

	// Attacked by Knights
	if !(KnightAttacks[sq] & b.Pieces[attackerColor][Knight]).IsEmpty() {
		return true
	}

	// Attacked by Kings
	if !(KingAttacks[sq] & b.Pieces[attackerColor][King]).IsEmpty() {
		return true
	}

	// Attacked by Sliders
	occ := b.Occupancy()

	// Bishop/Queen diagonals
	bishopQueens := b.Pieces[attackerColor][Bishop] | b.Pieces[attackerColor][Queen]
	if !(GetBishopAttacks(sq, occ) & bishopQueens).IsEmpty() {
		return true
	}

	// Rook/Queen orthogonals
	rookQueens := b.Pieces[attackerColor][Rook] | b.Pieces[attackerColor][Queen]
	if !(GetRookAttacks(sq, occ) & rookQueens).IsEmpty() {
		return true
	}

	return false
}
