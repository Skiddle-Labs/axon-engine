package eval

import (
	"github.com/Skiddle-Labs/axon-engine/internal/engine"
)

// evaluatePawns calculates the midgame and endgame scores for pawn structures.
func evaluatePawns(b *engine.Board, c engine.Color) (int, int) {
	mg, eg := 0, 0
	them := c ^ 1
	pawns := b.Pieces[c][engine.Pawn]
	enemyPawns := b.Pieces[them][engine.Pawn]

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
		}

		// 3. Connected and Phalanx pawns
		supported := false
		phalanx := false

		// Check for support (pawn behind diagonally)
		if c == engine.White {
			if rank > 0 {
				if file > 0 && pawns.Test(engine.NewSquare(file-1, rank-1)) {
					supported = true
				}
				if file < 7 && pawns.Test(engine.NewSquare(file+1, rank-1)) {
					supported = true
				}
			}
		} else {
			if rank < 7 {
				if file > 0 && pawns.Test(engine.NewSquare(file-1, rank+1)) {
					supported = true
				}
				if file < 7 && pawns.Test(engine.NewSquare(file+1, rank+1)) {
					supported = true
				}
			}
		}

		// Check for phalanx (side-by-side)
		if file > 0 && pawns.Test(engine.NewSquare(file-1, rank)) {
			phalanx = true
		}
		if file < 7 && pawns.Test(engine.NewSquare(file+1, rank)) {
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
			if c == engine.White {
				for r := 0; r <= rank; r++ {
					if (file > 0 && pawns.Test(engine.NewSquare(file-1, r))) ||
						(file < 7 && pawns.Test(engine.NewSquare(file+1, r))) {
						hasAdjacentBehind = true
						break
					}
				}
				if !hasAdjacentBehind && rank < 7 {
					if (file > 0 && enemyPawns.Test(engine.NewSquare(file-1, rank+1))) ||
						(file < 7 && enemyPawns.Test(engine.NewSquare(file+1, rank+1))) {
						mg += PawnBackwardMG
						eg += PawnBackwardEG
					}
				}
			} else {
				for r := 7; r >= rank; r-- {
					if (file > 0 && pawns.Test(engine.NewSquare(file-1, r))) ||
						(file < 7 && pawns.Test(engine.NewSquare(file+1, r))) {
						hasAdjacentBehind = true
						break
					}
				}
				if !hasAdjacentBehind && rank > 0 {
					if (file > 0 && enemyPawns.Test(engine.NewSquare(file-1, rank-1))) ||
						(file < 7 && enemyPawns.Test(engine.NewSquare(file+1, rank-1))) {
						mg += PawnBackwardMG
						eg += PawnBackwardEG
					}
				}
			}
		}

		// 5. Passed pawns
		frontMask := engine.Bitboard(0)
		if c == engine.White {
			for r := rank + 1; r <= 7; r++ {
				frontMask.Set(engine.NewSquare(file, r))
				if file > 0 {
					frontMask.Set(engine.NewSquare(file-1, r))
				}
				if file < 7 {
					frontMask.Set(engine.NewSquare(file+1, r))
				}
			}
		} else {
			for r := rank - 1; r >= 0; r-- {
				frontMask.Set(engine.NewSquare(file, r))
				if file > 0 {
					frontMask.Set(engine.NewSquare(file-1, r))
				}
				if file < 7 {
					frontMask.Set(engine.NewSquare(file+1, r))
				}
			}
		}

		if (frontMask & enemyPawns).IsEmpty() {
			bonus := 0
			if c == engine.White {
				bonus = rank * rank
			} else {
				bonus = (7 - rank) * (7 - rank)
			}
			mg += bonus * PawnPassedMG
			eg += bonus * PawnPassedEG
		}
	}

	return mg, eg
}
