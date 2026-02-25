package engine

import (
	"github.com/Skiddle-Labs/axon-engine/internal/types"
)

// GenerateMoves generates all pseudo-legal moves for the current board position.
// Pseudo-legal moves include moves that might leave the king in check.
func (b *Board) GenerateMoves() MoveList {
	return b.generateMoves(false)
}

// GenerateCaptures generates only pseudo-legal captures and promotions.
func (b *Board) GenerateCaptures() MoveList {
	return b.generateMoves(true)
}

func (b *Board) generateMoves(capturesOnly bool) MoveList {
	ml := MoveList{}
	us := b.SideToMove
	them := us ^ 1
	occ := b.Occupancy()
	enemyOcc := b.Colors[them]

	// 1. King Moves
	kingBB := b.Pieces[us][types.King]
	if !kingBB.IsEmpty() {
		from := kingBB.LSB()
		attacks := KingAttacks[from] & ^b.Colors[us]
		if capturesOnly {
			attacks &= enemyOcc
		}
		b.addMovesFromAttacks(&ml, from, attacks, enemyOcc)
	}

	// 2. Knight Moves
	knightBB := b.Pieces[us][types.Knight]
	for knightBB != 0 {
		from := knightBB.PopLSB()
		attacks := KnightAttacks[from] & ^b.Colors[us]
		if capturesOnly {
			attacks &= enemyOcc
		}
		b.addMovesFromAttacks(&ml, from, attacks, enemyOcc)
	}

	// 3. Bishop Moves
	bishopBB := b.Pieces[us][types.Bishop]
	for bishopBB != 0 {
		from := bishopBB.PopLSB()
		attacks := GetBishopAttacks(from, occ) & ^b.Colors[us]
		if capturesOnly {
			attacks &= enemyOcc
		}
		b.addMovesFromAttacks(&ml, from, attacks, enemyOcc)
	}

	// 4. Rook Moves
	rookBB := b.Pieces[us][types.Rook]
	for rookBB != 0 {
		from := rookBB.PopLSB()
		attacks := GetRookAttacks(from, occ) & ^b.Colors[us]
		if capturesOnly {
			attacks &= enemyOcc
		}
		b.addMovesFromAttacks(&ml, from, attacks, enemyOcc)
	}

	// 5. Queen Moves
	queenBB := b.Pieces[us][types.Queen]
	for queenBB != 0 {
		from := queenBB.PopLSB()
		attacks := GetQueenAttacks(from, occ) & ^b.Colors[us]
		if capturesOnly {
			attacks &= enemyOcc
		}
		b.addMovesFromAttacks(&ml, from, attacks, enemyOcc)
	}

	// 6. Pawn Moves
	pawns := b.Pieces[us][types.Pawn]
	if us == types.White {
		for pawns != 0 {
			from := pawns.PopLSB()

			// Single push
			to := from + 8
			if !occ.Test(to) {
				if to.Rank() == 7 {
					b.addPawnPromotionMoves(&ml, from, to, false)
				} else if !capturesOnly {
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
			if b.EnPassant != types.NoSquare && attacks.Test(b.EnPassant) {
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
				} else if !capturesOnly {
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
			if b.EnPassant != types.NoSquare && attacks.Test(b.EnPassant) {
				ml.AddMove(NewMove(from, b.EnPassant, EnPassantFlag))
			}
		}
	}

	// 7. Castling
	if !capturesOnly {
		if us == types.White {
			if b.Castling&WhiteKingside != 0 {
				if !occ.Test(types.F1) && !occ.Test(types.G1) {
					if !b.IsSquareAttacked(types.E1, types.Black) && !b.IsSquareAttacked(types.F1, types.Black) {
						ml.AddMove(NewMove(types.E1, types.G1, KingsideCast))
					}
				}
			}
			if b.Castling&WhiteQueenside != 0 {
				if !occ.Test(types.D1) && !occ.Test(types.C1) && !occ.Test(types.B1) {
					if !b.IsSquareAttacked(types.E1, types.Black) && !b.IsSquareAttacked(types.D1, types.Black) {
						ml.AddMove(NewMove(types.E1, types.C1, QueensideCast))
					}
				}
			}
		} else {
			if b.Castling&BlackKingside != 0 {
				if !occ.Test(types.F8) && !occ.Test(types.G8) {
					if !b.IsSquareAttacked(types.E8, types.White) && !b.IsSquareAttacked(types.F8, types.White) {
						ml.AddMove(NewMove(types.E8, types.G8, KingsideCast))
					}
				}
			}
			if b.Castling&BlackQueenside != 0 {
				if !occ.Test(types.D8) && !occ.Test(types.C8) && !occ.Test(types.B8) {
					if !b.IsSquareAttacked(types.E8, types.White) && !b.IsSquareAttacked(types.D8, types.White) {
						ml.AddMove(NewMove(types.E8, types.C8, QueensideCast))
					}
				}
			}
		}
	}

	return ml
}

func (b *Board) addPawnPromotionMoves(ml *MoveList, from, to types.Square, isCapture bool) {
	captureFlag := uint16(0)
	if isCapture {
		captureFlag = CaptureFlag
	}
	ml.AddMove(NewMove(from, to, PromoQueen|captureFlag))
	ml.AddMove(NewMove(from, to, PromoRook|captureFlag))
	ml.AddMove(NewMove(from, to, PromoBishop|captureFlag))
	ml.AddMove(NewMove(from, to, PromoKnight|captureFlag))
}

func (b *Board) whitePawnAttacks(sq types.Square) Bitboard {
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

func (b *Board) blackPawnAttacks(sq types.Square) Bitboard {
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
func (b *Board) addMovesFromAttacks(ml *MoveList, from types.Square, attacks Bitboard, enemyOcc Bitboard) {
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
func (b *Board) IsSquareAttacked(sq types.Square, attackerColor types.Color) bool {
	// Attacked by Pawns
	pawns := b.Pieces[attackerColor][types.Pawn]
	if attackerColor == types.White {
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
	if !(KnightAttacks[sq] & b.Pieces[attackerColor][types.Knight]).IsEmpty() {
		return true
	}

	// Attacked by Kings
	if !(KingAttacks[sq] & b.Pieces[attackerColor][types.King]).IsEmpty() {
		return true
	}

	// Attacked by Sliders
	occ := b.Occupancy()

	// Bishop/Queen diagonals
	bishopQueens := b.Pieces[attackerColor][types.Bishop] | b.Pieces[attackerColor][types.Queen]
	if !(GetBishopAttacks(sq, occ) & bishopQueens).IsEmpty() {
		return true
	}

	// Rook/Queen orthogonals
	rookQueens := b.Pieces[attackerColor][types.Rook] | b.Pieces[attackerColor][types.Queen]
	if !(GetRookAttacks(sq, occ) & rookQueens).IsEmpty() {
		return true
	}

	return false
}
