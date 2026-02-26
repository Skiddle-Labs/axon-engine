package main

import (
	"sync"

	"github.com/Skiddle-Labs/axon-engine/internal/engine"
	"github.com/Skiddle-Labs/axon-engine/internal/eval"
	"github.com/Skiddle-Labs/axon-engine/internal/types"
)

// PrecomputedEntry stores all necessary data to evaluate a position without re-running bitboard operations.
type PrecomputedEntry struct {
	Result     float64
	SideToMove types.Color
	MgW, EgW   int
	Features   [2]PrecomputedFeatures
}

// PrecomputedFeatures stores the counts and indices for a single side's evaluation components.
type PrecomputedFeatures struct {
	PSTIndices [7][]int
	Material   [7]int

	PawnDoubled          int
	PawnIsolated         int
	IsolatedPawnOpenFile int
	PawnSupported        int
	PawnPhalanx          int
	PawnBackward         int
	PawnPassed           int // sum of rank*rank for passed pawns
	ConnectedPassed      int

	KnightMobility [9]int
	BishopMobility [14]int
	RookMobility   [15]int
	QueenMobility  [28]int

	HasBishopPair      bool
	BishopLongDiagonal int

	KingSq            types.Square
	KingShieldClose   int
	KingShieldFar     int
	KingShieldMissing int
	KingAttackerCount [7]int

	VirtualMobility int
	PawnStorm       int

	HangingPieceValueSum int
	WeakAttackerCount    int

	KnightOutpost    int
	BishopOutpost    int
	RookOpenFile     int
	RookHalfOpenFile int
	RookOn7th        int
	RookBattery      int

	Space                int
	TrappedPiece         int
	KingNearPassedPawnEG int
}

// PrecomputeEntries converts raw entries into precomputed entries.
func PrecomputeEntries(entries []Entry) []PrecomputedEntry {
	precomputed := make([]PrecomputedEntry, len(entries))
	numThreads := *threads
	if numThreads <= 0 {
		numThreads = 1
	}

	chunkSize := (len(entries) + numThreads - 1) / numThreads
	var wg sync.WaitGroup

	for i := 0; i < numThreads; i++ {
		start := i * chunkSize
		if start >= len(entries) {
			break
		}
		end := start + chunkSize
		if end > len(entries) {
			end = len(entries)
		}

		wg.Add(1)
		go func(s, e int) {
			defer wg.Done()
			for j := s; j < e; j++ {
				precomputed[j] = Precompute(entries[j])
			}
		}(start, end)
	}
	wg.Wait()

	return precomputed
}

// Precompute extracts all evaluation features from a single board state.
func Precompute(entry Entry) PrecomputedEntry {
	b := entry.board
	mgW, egW, _ := calculatePhase(b)

	pe := PrecomputedEntry{
		Result:     entry.result,
		SideToMove: b.SideToMove,
		MgW:        mgW,
		EgW:        egW,
	}

	pe.Features[types.White] = extractFeatures(b, types.White)
	pe.Features[types.Black] = extractFeatures(b, types.Black)

	return pe
}

func calculatePhase(b *engine.Board) (int, int, int) {
	phase := 0
	phase += b.Pieces[types.White][types.Knight].Count() * eval.KnightPhase
	phase += b.Pieces[types.Black][types.Knight].Count() * eval.KnightPhase
	phase += b.Pieces[types.White][types.Bishop].Count() * eval.BishopPhase
	phase += b.Pieces[types.Black][types.Bishop].Count() * eval.BishopPhase
	phase += b.Pieces[types.White][types.Rook].Count() * eval.RookPhase
	phase += b.Pieces[types.Black][types.Rook].Count() * eval.RookPhase
	phase += b.Pieces[types.White][types.Queen].Count() * eval.QueenPhase
	phase += b.Pieces[types.Black][types.Queen].Count() * eval.QueenPhase

	if phase > eval.TotalPhase {
		phase = eval.TotalPhase
	}

	mgW := phase
	egW := eval.TotalPhase - phase
	return mgW, egW, phase
}

func extractFeatures(b *engine.Board, c types.Color) PrecomputedFeatures {
	f := PrecomputedFeatures{}
	occ := b.Occupancy()
	them := c ^ 1

	// Pawns
	pawns := b.Pieces[c][types.Pawn]
	enemyPawns := b.Pieces[them][types.Pawn]
	f.Material[types.Pawn] = pawns.Count()

	pawnCopy := pawns
	for pawnCopy != 0 {
		sq := pawnCopy.PopLSB()
		f.PSTIndices[types.Pawn] = append(f.PSTIndices[types.Pawn], getPSTIndex(sq, c))

		file := sq.File()
		rank := sq.Rank()

		// Doubled pawns
		if (pawns & (engine.FileA << file)).Count() > 1 {
			f.PawnDoubled++
		}

		// Isolated pawns
		isIsolated := true
		if file > 0 && (pawns&(engine.FileA<<(file-1))) != 0 {
			isIsolated = false
		}
		if file < 7 && (pawns&(engine.FileA<<(file+1))) != 0 {
			isIsolated = false
		}
		if isIsolated {
			f.PawnIsolated++

			// Penalty for isolated pawn on an open file
			if (enemyPawns & (engine.FileA << file)) == 0 {
				f.IsolatedPawnOpenFile++
			}
		}

		// Connected pawns (protected by another pawn)
		supported := false
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

		// Phalanx pawns (side-by-side)
		phalanx := false
		if file > 0 && pawns.Test(types.NewSquare(file-1, rank)) {
			phalanx = true
		}
		if file < 7 && pawns.Test(types.NewSquare(file+1, rank)) {
			phalanx = true
		}

		if supported {
			f.PawnSupported++
		} else if phalanx {
			f.PawnPhalanx++
		}

		// Backward pawn detection
		isBackward := false
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
						isBackward = true
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
						isBackward = true
					}
				}
			}
		}
		if isBackward {
			f.PawnBackward++
		}

		// Passed pawns
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
		if (frontMask & enemyPawns).IsEmpty() {
			bonus := 0
			if c == types.White {
				bonus = rank * rank
			} else {
				bonus = (7 - rank) * (7 - rank)
			}
			f.PawnPassed += bonus

			// Connected passed pawns
			if supported || phalanx {
				f.ConnectedPassed++
			}
		}
	}

	// Non-pawn pieces
	for pt := types.Knight; pt <= types.Queen; pt++ {
		pieces := b.Pieces[c][pt]
		f.Material[pt] = pieces.Count()
		for pieces != 0 {
			sq := pieces.PopLSB()
			f.PSTIndices[pt] = append(f.PSTIndices[pt], getPSTIndex(sq, c))
			var attacks engine.Bitboard
			switch pt {
			case types.Knight:
				attacks = engine.KnightAttacks[sq]
			case types.Bishop:
				attacks = engine.GetBishopAttacks(sq, occ)
			case types.Rook:
				attacks = engine.GetRookAttacks(sq, occ)
			case types.Queen:
				attacks = engine.GetQueenAttacks(sq, occ)
			}
			mobility := attacks & ^b.Colors[c]
			count := mobility.Count()
			switch pt {
			case types.Knight:
				f.KnightMobility[count]++
			case types.Bishop:
				f.BishopMobility[count]++
			case types.Rook:
				f.RookMobility[count]++
			case types.Queen:
				f.QueenMobility[count]++
			}

			if count == 0 {
				f.TrappedPiece++
			}

			// Virtual mobility (pressure on occupied squares)
			f.VirtualMobility += (attacks & occ).Count()

			// Outpost and File features
			switch pt {
			case types.Knight:
				if isOutpostPre(b, c, sq) {
					f.KnightOutpost++
				}
			case types.Bishop:
				if isOutpostPre(b, c, sq) {
					f.BishopOutpost++
				}
				if engine.IsLongDiagonal(sq) {
					f.BishopLongDiagonal++
				}
			case types.Rook:
				rank := sq.Rank()
				file := sq.File()
				fileBB := engine.FileA << file
				usPawnsOnFile := (pawns & fileBB) != 0
				themPawnsOnFile := (enemyPawns & fileBB) != 0

				if !usPawnsOnFile {
					if !themPawnsOnFile {
						f.RookOpenFile++
					} else {
						f.RookHalfOpenFile++
					}
				}

				// 7th Rank Bonus
				if (c == types.White && rank == 6) || (c == types.Black && rank == 1) {
					enemyKingSq := b.Pieces[them][types.King].LSB()
					enemyKingRank := enemyKingSq.Rank()
					if (c == types.White && enemyKingRank >= 6) || (c == types.Black && enemyKingRank <= 1) {
						f.RookOn7th++
					}
				}

				// Rook Battery
				otherRooks := b.Pieces[c][types.Rook] ^ (engine.Bitboard(1) << sq)
				if otherRooks != 0 {
					rankBB := engine.Rank1 << (8 * rank)
					if (otherRooks&fileBB) != 0 || (otherRooks&rankBB) != 0 {
						f.RookBattery++
					}
				}
			}
		}
	}

	if b.Pieces[c][types.Bishop].Count() >= 2 {
		f.HasBishopPair = true
	}

	// Space
	enemyPawnAttacks := engine.Bitboard(0)
	if them == types.White {
		enemyPawnAttacks |= (enemyPawns & ^engine.FileA) << 7
		enemyPawnAttacks |= (enemyPawns & ^engine.FileH) << 9
	} else {
		enemyPawnAttacks |= (enemyPawns & ^engine.FileA) >> 9
		enemyPawnAttacks |= (enemyPawns & ^engine.FileH) >> 7
	}
	var spaceMask engine.Bitboard
	if c == types.White {
		spaceMask = engine.Rank2 | engine.Rank3 | engine.Rank4
	} else {
		spaceMask = engine.Rank7 | engine.Rank6 | engine.Rank5
	}
	spaceMask &= (engine.FileC | engine.FileD | engine.FileE | engine.FileF)
	f.Space = (spaceMask & ^enemyPawnAttacks).Count()

	// King
	kingBB := b.Pieces[c][types.King]
	f.Material[types.King] = kingBB.Count()
	if !kingBB.IsEmpty() {
		sq := kingBB.LSB()
		f.KingSq = sq
		f.PSTIndices[types.King] = append(f.PSTIndices[types.King], getPSTIndex(sq, c))

		// King Safety (Pawn Shield)
		rank := sq.Rank()
		file := sq.File()
		if c == types.White {
			if rank < 7 {
				for f_idx := file - 1; f_idx <= file+1; f_idx++ {
					if f_idx >= 0 && f_idx <= 7 {
						if pawns.Test(types.NewSquare(f_idx, rank+1)) {
							f.KingShieldClose++
						} else if rank < 6 && pawns.Test(types.NewSquare(f_idx, rank+2)) {
							f.KingShieldFar++
						} else {
							f.KingShieldMissing++
						}
					}
				}
			}
		} else {
			if rank > 0 {
				for f_idx := file - 1; f_idx <= file+1; f_idx++ {
					if f_idx >= 0 && f_idx <= 7 {
						if pawns.Test(types.NewSquare(f_idx, rank-1)) {
							f.KingShieldClose++
						} else if rank > 1 && pawns.Test(types.NewSquare(f_idx, rank-2)) {
							f.KingShieldFar++
						} else {
							f.KingShieldMissing++
						}
					}
				}
			}
		}

		// King Safety (Pawn Storm)
		enemyPawns := b.Pieces[them][types.Pawn]
		for fIdx := file - 1; fIdx <= file+1; fIdx++ {
			if fIdx < 0 || fIdx > 7 {
				continue
			}
			pawnsOnFile := enemyPawns & (engine.FileA << fIdx)
			if pawnsOnFile != 0 {
				var pSq types.Square
				var dist int
				if c == types.White {
					pSq = pawnsOnFile.MSB() // Highest rank pawn
					dist = 7 - pSq.Rank()
				} else {
					pSq = pawnsOnFile.LSB() // Lowest rank pawn
					dist = pSq.Rank()
				}

				if dist < 4 {
					f.PawnStorm += (4 - dist)
				}
			}
		}

		// King proximity to passed pawns
		// Support our own
		pCopy := b.Pieces[c][types.Pawn]
		for pCopy != 0 {
			psq := pCopy.PopLSB()
			if isPassedPre(b, c, psq) {
				dist := engine.ManhattanDistance(sq, psq)
				f.KingNearPassedPawnEG += (7 - dist)
			}
		}
		// Stop enemy
		eCopy := b.Pieces[them][types.Pawn]
		for eCopy != 0 {
			psq := eCopy.PopLSB()
			if isPassedPre(b, them, psq) {
				dist := engine.ManhattanDistance(sq, psq)
				f.KingNearPassedPawnEG += (7 - dist)
			}
		}

		// King Safety (Attacker Zone)
		zone := engine.KingAttacks[sq] | (engine.Bitboard(1) << sq)
		for pt := types.Knight; pt <= types.Queen; pt++ {
			p := b.Pieces[them][pt]
			for p != 0 {
				asq := p.PopLSB()
				var attacks engine.Bitboard
				switch pt {
				case types.Knight:
					attacks = engine.KnightAttacks[asq]
				case types.Bishop:
					attacks = engine.GetBishopAttacks(asq, occ)
				case types.Rook:
					attacks = engine.GetRookAttacks(asq, occ)
				case types.Queen:
					attacks = engine.GetQueenAttacks(asq, occ)
				}
				if !(attacks & zone).IsEmpty() {
					f.KingAttackerCount[pt]++
				}
			}
		}
	}

	// Threats
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
					f.HangingPieceValueSum += engine.PieceValues[pt]
				} else {
					for ept := types.Pawn; ept < pt; ept++ {
						if !(enemyAttackers & b.Pieces[them][ept]).IsEmpty() {
							f.WeakAttackerCount++
							break
						}
					}
				}
			}
		}
	}

	return f
}

func getPSTIndex(sq types.Square, c types.Color) int {
	rank := int(sq) / 8
	file := int(sq) % 8
	if c == types.White {
		return (7-rank)*8 + file
	}
	return rank*8 + file
}

// Evaluate computes the score using precomputed features and current evaluation parameters.
func (pe *PrecomputedEntry) Evaluate() int {
	// Tempo bonus
	tempoMg, tempoEg := 0, 0
	if pe.SideToMove == types.White {
		tempoMg = eval.TempoMG
		tempoEg = eval.TempoEG
	} else {
		tempoMg = -eval.TempoMG
		tempoEg = -eval.TempoEG
	}

	mgWhite, egWhite := pe.evaluateColor(types.White)
	mgBlack, egBlack := pe.evaluateColor(types.Black)

	mgScore := mgWhite - mgBlack + tempoMg
	egScore := egWhite - egBlack + tempoEg

	score := (mgScore*pe.MgW + egScore*pe.EgW) / eval.TotalPhase

	// Mop-up
	if score > 300 || score < -300 {
		if pe.EgW > 14 { // Endgame weight > 14 corresponds to phase < 10
			bonus := 0
			us := types.White
			them := types.Black
			if score < 0 {
				us = types.Black
				them = types.White
			}

			enemyKingSq := pe.Features[them].KingSq
			ourKingSq := pe.Features[us].KingSq

			bonus += engine.CenterDistance(enemyKingSq) * eval.MopUpBonus
			dist := engine.ManhattanDistance(ourKingSq, enemyKingSq)
			bonus += (14 - dist) * (eval.MopUpBonus / 2)

			bonus = (bonus * pe.EgW) / eval.TotalPhase
			if score < 0 {
				score -= bonus
			} else {
				score += bonus
			}
		}
	}

	if pe.SideToMove == types.Black {
		return -score
	}
	return score
}

func (pe *PrecomputedEntry) evaluateColor(c types.Color) (int, int) {
	f := pe.Features[c]
	mg, eg := 0, 0

	// Material
	mg += f.Material[types.Pawn] * eval.PawnMG
	eg += f.Material[types.Pawn] * eval.PawnEG
	mg += f.Material[types.Knight] * eval.KnightMG
	eg += f.Material[types.Knight] * eval.KnightEG
	mg += f.Material[types.Bishop] * eval.BishopMG
	eg += f.Material[types.Bishop] * eval.BishopEG
	mg += f.Material[types.Rook] * eval.RookMG
	eg += f.Material[types.Rook] * eval.RookEG
	mg += f.Material[types.Queen] * eval.QueenMG
	eg += f.Material[types.Queen] * eval.QueenEG

	// PST
	for pt := types.Pawn; pt <= types.King; pt++ {
		for _, idx := range f.PSTIndices[pt] {
			mg += eval.MgPST[pt][idx]
			eg += eval.EgPST[pt][idx]
		}
	}

	// Pawn Structure
	mg += f.PawnDoubled * eval.PawnDoubledMG
	eg += f.PawnDoubled * eval.PawnDoubledEG
	mg += f.PawnIsolated * eval.PawnIsolatedMG
	eg += f.PawnIsolated * eval.PawnIsolatedEG
	mg += f.PawnSupported * eval.PawnSupportedMG
	eg += f.PawnSupported * eval.PawnSupportedEG
	mg += f.PawnPhalanx * eval.PawnPhalanxMG
	eg += f.PawnPhalanx * eval.PawnPhalanxEG
	mg += f.PawnBackward * eval.PawnBackwardMG
	eg += f.PawnBackward * eval.PawnBackwardEG
	mg += f.PawnPassed * eval.PawnPassedMG
	eg += f.PawnPassed * eval.PawnPassedEG
	mg += f.IsolatedPawnOpenFile * eval.IsolatedPawnOpenFileMG
	eg += f.IsolatedPawnOpenFile * eval.IsolatedPawnOpenFileEG
	mg += f.ConnectedPassed * eval.ConnectedPassedMG
	eg += f.ConnectedPassed * eval.ConnectedPassedEG

	// Mobility
	for i := 0; i < 9; i++ {
		mg += f.KnightMobility[i] * eval.KnightMobilityMG[i]
		eg += f.KnightMobility[i] * eval.KnightMobilityEG[i]
	}
	for i := 0; i < 14; i++ {
		mg += f.BishopMobility[i] * eval.BishopMobilityMG[i]
		eg += f.BishopMobility[i] * eval.BishopMobilityEG[i]
	}
	for i := 0; i < 15; i++ {
		mg += f.RookMobility[i] * eval.RookMobilityMG[i]
		eg += f.RookMobility[i] * eval.RookMobilityEG[i]
	}
	for i := 0; i < 28; i++ {
		mg += f.QueenMobility[i] * eval.QueenMobilityMG[i]
		eg += f.QueenMobility[i] * eval.QueenMobilityEG[i]
	}

	mg += f.VirtualMobility * eval.VirtualMobilityMG
	eg += f.VirtualMobility * eval.VirtualMobilityEG

	mg += f.TrappedPiece * eval.TrappedPieceMG
	eg += f.TrappedPiece * eval.TrappedPieceEG

	// Other
	if f.HasBishopPair {
		mg += eval.BishopPairMG
		eg += eval.BishopPairEG
	}

	mg += f.BishopLongDiagonal * eval.BishopLongDiagonalMG
	eg += f.BishopLongDiagonal * eval.BishopLongDiagonalEG

	mg += f.KnightOutpost * eval.KnightOutpostMG
	eg += f.KnightOutpost * eval.KnightOutpostEG
	mg += f.BishopOutpost * eval.BishopOutpostMG
	eg += f.BishopOutpost * eval.BishopOutpostEG
	mg += f.RookOpenFile * eval.RookOpenFileMG
	eg += f.RookOpenFile * eval.RookOpenFileEG
	mg += f.RookHalfOpenFile * eval.RookHalfOpenFileMG
	eg += f.RookHalfOpenFile * eval.RookHalfOpenFileEG
	mg += f.RookOn7th * eval.RookOn7thMG
	eg += f.RookOn7th * eval.RookOn7thEG
	mg += f.RookBattery * eval.RookBatteryMG
	eg += f.RookBattery * eval.RookBatteryEG

	mg += f.Space * eval.SpaceMG

	// King Safety
	mg += f.KingShieldClose * eval.KingShieldClose
	mg += f.KingShieldFar * eval.KingShieldFar
	mg += f.KingShieldMissing * eval.KingShieldMissing

	mg += f.PawnStorm * eval.PawnStormMG
	eg += f.PawnStorm * eval.PawnStormEG

	eg += f.KingNearPassedPawnEG * eval.KingNearPassedPawnEG

	attackerCount, attackerWeight := 0, 0
	for pt := types.Knight; pt <= types.Queen; pt++ {
		attackerCount += f.KingAttackerCount[pt]
		attackerWeight += f.KingAttackerCount[pt] * eval.KingAttackerWeight[pt]
	}

	if attackerCount > 0 {
		penaltyIndex := attackerWeight
		if penaltyIndex >= 100 {
			penaltyIndex = 99
		}
		mg -= eval.SafetyTable[penaltyIndex]
	}

	// Threats
	if eval.HangingDivisorMG != 0 {
		mg -= f.HangingPieceValueSum / eval.HangingDivisorMG
	}
	if eval.HangingDivisorEG != 0 {
		eg -= f.HangingPieceValueSum / eval.HangingDivisorEG
	}
	mg += f.WeakAttackerCount * eval.WeakAttackerMG
	eg += f.WeakAttackerCount * eval.WeakAttackerEG

	return mg, eg
}

func isOutpostPre(b *engine.Board, c types.Color, sq types.Square) bool {
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

// isPassedPre is a helper for precomputation.
func isPassedPre(b *engine.Board, c types.Color, sq types.Square) bool {
	file := sq.File()
	rank := sq.Rank()
	enemyPawns := b.Pieces[c^1][types.Pawn]

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

	return (frontMask & enemyPawns).IsEmpty()
}
