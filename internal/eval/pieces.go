package eval

import (
	"github.com/Skiddle-Labs/axon-engine/internal/engine"
	"github.com/Skiddle-Labs/axon-engine/internal/types"
)

// evaluatePieces calculates the material, mobility, and positional bonuses for all non-pawn pieces.
func evaluatePieces(b *engine.Board, c types.Color) (int, int) {
	mg, eg := 0, 0
	them := c ^ 1
	occ := b.Occupancy()
	pawns := b.Pieces[c][types.Pawn]
	enemyPawns := b.Pieces[them][types.Pawn]

	for pt := types.Knight; pt <= types.Queen; pt++ {
		pieces := b.Pieces[c][pt]
		for pieces != 0 {
			sq := pieces.PopLSB()
			rank := sq.Rank()
			file := sq.File()

			// 1. Material Values and Positional Features
			var attacks, mobility engine.Bitboard
			switch pt {
			case types.Knight:
				mg += KnightMG
				eg += KnightEG

				attacks = engine.KnightAttacks[sq]
				mobility = attacks & ^b.Colors[c]
				mobilityCount := mobility.Count()
				mg += KnightMobilityMG[mobilityCount]
				eg += KnightMobilityEG[mobilityCount]

				if mobilityCount == 0 {
					mg += TrappedPieceMG
					eg += TrappedPieceEG
				}

				// Virtual mobility (pressure on occupied squares)
				virtual := (attacks & b.Occupancy()).Count()
				mg += virtual * VirtualMobilityMG
				eg += virtual * VirtualMobilityEG

				if isOutpost(b, c, sq) {
					mg += KnightOutpostMG
					eg += KnightOutpostEG
				}

			case types.Bishop:
				mg += BishopMG
				eg += BishopEG

				attacks = engine.GetBishopAttacks(sq, occ)
				mobility = attacks & ^b.Colors[c]
				mobilityCount := mobility.Count()
				mg += BishopMobilityMG[mobilityCount]
				eg += BishopMobilityEG[mobilityCount]

				if mobilityCount == 0 {
					mg += TrappedPieceMG
					eg += TrappedPieceEG
				}

				virtual := (attacks & b.Occupancy()).Count()
				mg += virtual * VirtualMobilityMG
				eg += virtual * VirtualMobilityEG

				if isOutpost(b, c, sq) {
					mg += BishopOutpostMG
					eg += BishopOutpostEG
				}

				// Long Diagonal Bonus
				if engine.IsLongDiagonal(sq) {
					mg += BishopLongDiagonalMG
					eg += BishopLongDiagonalEG
				}

			case types.Rook:
				mg += RookMG
				eg += RookEG

				attacks = engine.GetRookAttacks(sq, occ)
				mobility = attacks & ^b.Colors[c]
				mobilityCount := mobility.Count()
				mg += RookMobilityMG[mobilityCount]
				eg += RookMobilityEG[mobilityCount]

				if mobilityCount == 0 {
					mg += TrappedPieceMG
					eg += TrappedPieceEG
				}

				virtual := (attacks & b.Occupancy()).Count()
				mg += virtual * VirtualMobilityMG
				eg += virtual * VirtualMobilityEG

				// File bonuses (Open/Half-Open)
				fileBB := engine.FileA << file
				usPawnsOnFile := (pawns & fileBB) != 0
				themPawnsOnFile := (enemyPawns & fileBB) != 0

				if !usPawnsOnFile {
					if !themPawnsOnFile {
						mg += RookOpenFileMG
						eg += RookOpenFileEG
					} else {
						mg += RookHalfOpenFileMG
						eg += RookHalfOpenFileEG
					}
				}

				// 7th Rank Bonus
				if (c == types.White && rank == 6) || (c == types.Black && rank == 1) {
					enemyKingSq := b.Pieces[them][types.King].LSB()
					enemyKingRank := enemyKingSq.Rank()
					// Bonus applies if enemy king is restricted to the last two ranks
					if (c == types.White && enemyKingRank >= 6) || (c == types.Black && enemyKingRank <= 1) {
						mg += RookOn7thMG
						eg += RookOn7thEG
					}
				}

				// Rook Battery Bonus
				// Check for other rooks on the same file or rank
				otherRooks := b.Pieces[c][types.Rook] ^ (engine.Bitboard(1) << sq)
				if otherRooks != 0 {
					rankBB := engine.Rank1 << (8 * rank)
					if (otherRooks&fileBB) != 0 || (otherRooks&rankBB) != 0 {
						mg += RookBatteryMG
						eg += RookBatteryEG
					}
				}

			case types.Queen:
				mg += QueenMG
				eg += QueenEG

				attacks = engine.GetQueenAttacks(sq, occ)
				mobility = attacks & ^b.Colors[c]
				mobilityCount := mobility.Count()
				mg += QueenMobilityMG[mobilityCount]
				eg += QueenMobilityEG[mobilityCount]

				if mobilityCount == 0 {
					mg += TrappedPieceMG
					eg += TrappedPieceEG
				}

				virtual := (attacks & b.Occupancy()).Count()
				mg += virtual * VirtualMobilityMG
				eg += virtual * VirtualMobilityEG
			}
		}
	}

	// 2. Special Bonuses
	if b.Pieces[c][types.Bishop].Count() >= 2 {
		mg += BishopPairMG
		eg += BishopPairEG
	}

	return mg, eg
}

// evaluateThreats detects hanging pieces or pieces attacked by weaker ones.
func evaluateThreats(b *engine.Board, c types.Color) (int, int) {
	mg, eg := 0, 0
	them := c ^ 1
	occ := b.Occupancy()
	enemyOcc := b.Colors[them]
	usOcc := b.Colors[c]

	for pt := types.Pawn; pt <= types.Queen; pt++ {
		subset := b.Pieces[c][pt]
		for subset != 0 {
			sq := subset.PopLSB()
			attackers := b.AllAttackers(sq, occ)
			enemyAttackers := attackers & enemyOcc

			if !enemyAttackers.IsEmpty() {
				defenders := attackers & usOcc
				if defenders.IsEmpty() {
					// Hanging piece penalty
					mg -= engine.PieceValues[pt] / HangingDivisorMG
					eg -= engine.PieceValues[pt] / HangingDivisorEG
				} else {
					// Attacked by weaker piece
					for ept := types.Pawn; ept < pt; ept++ {
						if !(enemyAttackers & b.Pieces[them][ept]).IsEmpty() {
							mg += WeakAttackerMG
							eg += WeakAttackerEG
							break
						}
					}
				}
			}
		}
	}

	return mg, eg
}

// isOutpost determines if a piece is on a square supported by a pawn and cannot be chased by enemy pawns.
func isOutpost(b *engine.Board, c types.Color, sq types.Square) bool {
	rank := sq.Rank()
	file := sq.File()

	// Only ranks 3-6 (indices 2-5) for outposts
	if rank < 2 || rank > 5 {
		return false
	}

	pawns := b.Pieces[c][types.Pawn]
	enemyPawns := b.Pieces[c^1][types.Pawn]

	// 1. Supported by a pawn
	supported := false
	if c == types.White {
		if file > 0 && pawns.Test(types.NewSquare(file-1, rank-1)) {
			supported = true
		}
		if file < 7 && pawns.Test(types.NewSquare(file+1, rank-1)) {
			supported = true
		}
	} else {
		if file > 0 && pawns.Test(types.NewSquare(file-1, rank+1)) {
			supported = true
		}
		if file < 7 && pawns.Test(types.NewSquare(file+1, rank+1)) {
			supported = true
		}
	}

	if !supported {
		return false
	}

	// 2. Cannot be attacked by an enemy pawn
	if c == types.White {
		for r := rank + 1; r <= 7; r++ {
			if file > 0 && enemyPawns.Test(types.NewSquare(file-1, r)) {
				return false
			}
			if file < 7 && enemyPawns.Test(types.NewSquare(file+1, r)) {
				return false
			}
		}
	} else {
		for r := rank - 1; r >= 0; r-- {
			if file > 0 && enemyPawns.Test(types.NewSquare(file-1, r)) {
				return false
			}
			if file < 7 && enemyPawns.Test(types.NewSquare(file+1, r)) {
				return false
			}
		}
	}

	return true
}
