package board

// GenerateMoves generates all pseudo-legal moves for the current board position.
// Pseudo-legal moves include moves that might leave the king in check.
func (b *Board) GenerateMoves() MoveList {
	ml := MoveList{}
	us := b.SideToMove
	them := us ^ 1
	occ := b.Occupancy()
	enemyOcc := b.Colors[them]
	empty := ^occ

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

	// TODO: Implement Pawn Moves
	// TODO: Implement Castling

	return ml
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
	// (Implementation depends on pawn capture logic)

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
