package eval

import (
	"github.com/Skiddle-Labs/axon-engine/internal/engine"
	"github.com/Skiddle-Labs/axon-engine/internal/types"
)

// evaluatePawns calculates the midgame and endgame scores for pawn structures.
func evaluatePawns(b *engine.Board, c types.Color) (int, int) {
	mg, eg := 0, 0
	them := c ^ 1
	pawns := b.Pieces[c][types.Pawn]
	enemyPawns := b.Pieces[them][types.Pawn]

	pawnCopy := pawns
	for pawnCopy != 0 {
		sq := pawnCopy.PopLSB()
		file := sq.File()
		rank := sq.Rank()

		// 1. Doubled pawns
		if (pawns & (engine.FileA << file)).Count() > 1 {
			mg += PawnDoubledMG
			eg += PawnDoubledEG
		}

		// 2. Isolated pawns
		isIsolated := true
		if file > 0 && (pawns&(engine.FileA<<(file-1))) != 0 {
			isIsolated = false
		}
		if file < 7 && (pawns&(engine.FileA<<(file+1))) != 0 {
			isIsolated = false
		}
		if isIsolated {
			mg += PawnIsolatedMG
			eg += PawnIsolatedEG

			// Penalty for isolated pawn on an open file
			if (enemyPawns & (engine.FileA << file)) == 0 {
				mg += IsolatedPawnOpenFileMG
				eg += IsolatedPawnOpenFileEG
			}
		}

		// 3. Connected and Phalanx pawns
		supported := false
		phalanx := false

		// Check for support (pawn behind diagonally)
		if c == types.White {
			if rank > 0 {
				if file > 0 && pawns.Test(types.NewSquare(file-1, rank-1)) {
					supported = true
				}
				if file < 7 && pawns.Test(types.NewSquare(file+1, rank-1)) {
					supported = true
				}
			}
		} else {
			if rank < 7 {
				if file > 0 && pawns.Test(types.NewSquare(file-1, rank+1)) {
					supported = true
				}
				if file < 7 && pawns.Test(types.NewSquare(file+1, rank+1)) {
					supported = true
				}
			}
		}

		// Check for phalanx (side-by-side)
		if file > 0 && pawns.Test(types.NewSquare(file-1, rank)) {
			phalanx = true
		}
		if file < 7 && pawns.Test(types.NewSquare(file+1, rank)) {
			phalanx = true
		}

		if supported {
			mg += PawnSupportedMG
			eg += PawnSupportedEG
		} else if phalanx {
			mg += PawnPhalanxMG
			eg += PawnPhalanxEG
		}

		// 4. Backward pawn detection
		if !supported && !phalanx {
			hasAdjacentBehind := false
			if c == types.White {
				for r := 0; r <= rank; r++ {
					if (file > 0 && pawns.Test(types.NewSquare(file-1, r))) ||
						(file < 7 && pawns.Test(types.NewSquare(file+1, r))) {
						hasAdjacentBehind = true
						break
					}
				}
				if !hasAdjacentBehind && rank < 7 {
					if (file > 0 && enemyPawns.Test(types.NewSquare(file-1, rank+1))) ||
						(file < 7 && enemyPawns.Test(types.NewSquare(file+1, rank+1))) {
						mg += PawnBackwardMG
						eg += PawnBackwardEG
					}
				}
			} else {
				for r := 7; r >= rank; r-- {
					if (file > 0 && pawns.Test(types.NewSquare(file-1, r))) ||
						(file < 7 && pawns.Test(types.NewSquare(file+1, r))) {
						hasAdjacentBehind = true
						break
					}
				}
				if !hasAdjacentBehind && rank > 0 {
					if (file > 0 && enemyPawns.Test(types.NewSquare(file-1, rank-1))) ||
						(file < 7 && enemyPawns.Test(types.NewSquare(file+1, rank-1))) {
						mg += PawnBackwardMG
						eg += PawnBackwardEG
					}
				}
			}
		}

		// 5. Passed pawns
		frontMask := engine.Bitboard(0)
		if c == types.White {
			for r := rank + 1; r <= 7; r++ {
				frontMask.Set(types.NewSquare(file, r))
				if file > 0 {
					frontMask.Set(types.NewSquare(file-1, r))
				}
				if file < 7 {
					frontMask.Set(types.NewSquare(file+1, r))
				}
			}
		} else {
			for r := rank - 1; r >= 0; r-- {
				frontMask.Set(types.NewSquare(file, r))
				if file > 0 {
					frontMask.Set(types.NewSquare(file-1, r))
				}
				if file < 7 {
					frontMask.Set(types.NewSquare(file+1, r))
				}
			}
		}

		isPassed := (frontMask & enemyPawns).IsEmpty()

		// En Passant Awareness: If this pawn just jumped over an enemy pawn's attack square,
		// it can be captured EP, so it's not "safely" passed yet.
		if isPassed && b.EnPassant != types.NoSquare {
			if c == types.White && rank == 3 && b.EnPassant == types.NewSquare(file, 2) {
				if (file > 0 && enemyPawns.Test(types.NewSquare(file-1, 3))) ||
					(file < 7 && enemyPawns.Test(types.NewSquare(file+1, 3))) {
					isPassed = false
				}
			} else if c == types.Black && rank == 4 && b.EnPassant == types.NewSquare(file, 5) {
				if (file > 0 && enemyPawns.Test(types.NewSquare(file-1, 4))) ||
					(file < 7 && enemyPawns.Test(types.NewSquare(file+1, 4))) {
					isPassed = false
				}
			}
		}

		if isPassed {
			bonus := 0
			if c == types.White {
				bonus = rank * rank
			} else {
				bonus = (7 - rank) * (7 - rank)
			}
			mg += bonus * PawnPassedMG
			eg += bonus * PawnPassedEG

			// 6. Blockade penalty
			// A passed pawn's strength is significantly reduced if an enemy piece is blockading its advance.
			stopSq := types.NoSquare
			if c == types.White && rank < 7 {
				stopSq = types.NewSquare(file, rank+1)
			} else if c == types.Black && rank > 0 {
				stopSq = types.NewSquare(file, rank-1)
			}

			if stopSq != types.NoSquare {
				blocker := b.PieceAt(stopSq)
				if blocker != types.NoPiece && blocker.Color() == them {
					mg -= 15
					eg -= 25
				}
			}

			// 7. Connected passed pawns
			if supported || phalanx {
				mg += ConnectedPassedMG
				eg += ConnectedPassedEG
			}
		}
	}

	return mg, eg
}
