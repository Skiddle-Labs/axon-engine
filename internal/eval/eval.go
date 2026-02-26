package eval

import (
	"github.com/Skiddle-Labs/axon-engine/internal/engine"
	"github.com/Skiddle-Labs/axon-engine/internal/nnue"
	"github.com/Skiddle-Labs/axon-engine/internal/types"
)

// Evaluate returns a score for the current board position using tapered evaluation.
func Evaluate(b *engine.Board) int {
	if isInsufficientMaterial(b) {
		return 0
	}

	mgW, egW, phase := calculatePhase(b)
	hasNNUE := nnue.UseNNUE && nnue.CurrentNetwork != nil

	if hasNNUE {
		nnueScore := nnue.EvaluateForward(&b.Accumulators[0], &b.Accumulators[1], b.SideToMove)

		// Optimization: Use pure NNUE in midgame for maximum search speed.
		// In the endgame (phase < 12), we blend with HCE for better specialized knowledge.
		if phase >= 12 {
			return nnueScore
		}

		// Calculate HCE only when needed for blending in the endgame.
		hceScore := calculateHCE(b, mgW, egW, phase)

		// Blend NNUE (80%) and HCE (20%) for robust endgame play.
		perspectiveHCE := hceScore
		if b.SideToMove == types.Black {
			perspectiveHCE = -hceScore
		}

		return (nnueScore*8 + perspectiveHCE*2) / 10
	}

	hceScore := calculateHCE(b, mgW, egW, phase)
	if b.SideToMove == types.Black {
		return -hceScore
	}
	return hceScore
}

// calculateHCE computes the hand-coded evaluation of the board from White's perspective.
func calculateHCE(b *engine.Board, mgW, egW, phase int) int {
	// Tempo bonus
	tempoMg, tempoEg := 0, 0
	if b.SideToMove == types.White {
		tempoMg = TempoMG
		tempoEg = TempoEG
	} else {
		tempoMg = -TempoMG
		tempoEg = -TempoEG
	}

	// Pawn Structure Cache
	var pMgW, pEgW, pMgB, pEgB int
	if entry, ok := GlobalPawnTable.Probe(b.PawnHash); ok {
		pMgW = entry.MgScore[0]
		pEgW = entry.EgScore[0]
		pMgB = entry.MgScore[1]
		pEgB = entry.EgScore[1]
	} else {
		pMgW, pEgW = evaluatePawnStructure(b, types.White)
		pMgB, pEgB = evaluatePawnStructure(b, types.Black)
		GlobalPawnTable.Store(b.PawnHash, pMgW, pEgW, pMgB, pEgB)
	}

	mgWhite, egWhite := evaluateColor(b, types.White, pMgW, pEgW)
	mgBlack, egBlack := evaluateColor(b, types.Black, pMgB, pEgB)

	mgScore := mgWhite - mgBlack + tempoMg
	egScore := egWhite - egBlack + tempoEg

	hceScore := (mgScore*mgW + egScore*egW) / TotalPhase

	// 8. Mop-up evaluation (driving enemy king to the edge)
	if hceScore > 300 || hceScore < -300 {
		hceScore += evaluateMopUp(b, hceScore, phase)
	}

	// Scale evaluation in drawish endgames
	return scaleEndgame(b, hceScore)
}

// calculatePhase determines the game phase for tapered evaluation.
// Returns mgWeight, egWeight, and phase (where TotalPhase is opening and 0 is endgame).
func calculatePhase(b *engine.Board) (int, int, int) {
	phase := 0
	phase += b.Pieces[types.White][types.Knight].Count() * KnightPhase
	phase += b.Pieces[types.Black][types.Knight].Count() * KnightPhase
	phase += b.Pieces[types.White][types.Bishop].Count() * BishopPhase
	phase += b.Pieces[types.Black][types.Bishop].Count() * BishopPhase
	phase += b.Pieces[types.White][types.Rook].Count() * RookPhase
	phase += b.Pieces[types.Black][types.Rook].Count() * RookPhase
	phase += b.Pieces[types.White][types.Queen].Count() * QueenPhase
	phase += b.Pieces[types.Black][types.Queen].Count() * QueenPhase

	if phase > TotalPhase {
		phase = TotalPhase
	}

	mgW := phase
	egW := TotalPhase - phase
	return mgW, egW, phase
}

// evaluateColor computes the evaluation for a single side.
func evaluateColor(b *engine.Board, c types.Color, pMg, pEg int) (int, int) {
	mg, eg := pMg, pEg

	// 1. PST and Piece Material
	// We handle piece material in evaluatePieces and non-pawn PST here.
	for pt := types.Knight; pt <= types.King; pt++ {
		pieces := b.Pieces[c][pt]
		for pieces != 0 {
			sq := pieces.PopLSB()
			idx := getPST(sq, c)
			mg += MgPST[pt][idx]
			eg += EgPST[pt][idx]
		}
	}

	// 2. Pieces: Material, Mobility, Outposts, Bishop Pair
	pcmg, pceg := evaluatePieces(b, c)
	mg += pcmg
	eg += pceg

	// 4. King Safety: Shields, Attackers, Storms
	kmg, keg := evaluateKingSafety(b, c)
	mg += kmg
	eg += keg

	// 5. Threats: Hanging pieces, Weak attackers
	tmg, teg := evaluateThreats(b, c)
	mg += tmg
	eg += teg

	// 6. Space Evaluation
	mg += evaluateSpace(b, c)

	// 8. King proximity to passed pawns in endgames
	eg += evaluateKingPassedPawnProximity(b, c)

	// 7. Bishop vs Knight Scaling
	numPawns := b.Pieces[c][types.Pawn].Count()
	if numPawns > 8 {
		numPawns = 8
	}
	numBishops := b.Pieces[c][types.Bishop].Count()
	numKnights := b.Pieces[c][types.Knight].Count()
	mg += numBishops * BishopPawnScaling[numPawns]
	eg += numBishops * BishopPawnScaling[numPawns]
	mg += numKnights * KnightPawnScaling[numPawns]
	eg += numKnights * KnightPawnScaling[numPawns]

	return mg, eg
}

// evaluatePawnStructure calculates pawn-only scores (material, PST, structure).
func evaluatePawnStructure(b *engine.Board, c types.Color) (int, int) {
	mg, eg := 0, 0

	pawns := b.Pieces[c][types.Pawn]
	for pawns != 0 {
		sq := pawns.PopLSB()
		idx := getPST(sq, c)
		mg += MgPST[types.Pawn][idx] + PawnMG
		eg += EgPST[types.Pawn][idx] + PawnEG
	}

	pmg, peg := evaluatePawns(b, c)
	return mg + pmg, eg + peg
}

// evaluateSpace rewards controlling the central ranks.
func evaluateSpace(b *engine.Board, c types.Color) int {
	score := 0
	them := c ^ 1
	enemyPawnAttacks := engine.Bitboard(0)

	enemyPawns := b.Pieces[them][types.Pawn]
	if them == types.White {
		// White pawn attacks
		enemyPawnAttacks |= (enemyPawns & ^engine.FileA) << 7
		enemyPawnAttacks |= (enemyPawns & ^engine.FileH) << 9
	} else {
		// Black pawn attacks
		enemyPawnAttacks |= (enemyPawns & ^engine.FileA) >> 9
		enemyPawnAttacks |= (enemyPawns & ^engine.FileH) >> 7
	}

	// Space region: ranks 2, 3, 4 for White; 7, 6, 5 for Black.
	// We count squares that are not attacked by enemy pawns.
	var spaceMask engine.Bitboard
	if c == types.White {
		spaceMask = engine.Rank2 | engine.Rank3 | engine.Rank4
	} else {
		spaceMask = engine.Rank7 | engine.Rank6 | engine.Rank5
	}

	// Filter by central files
	spaceMask &= (engine.FileC | engine.FileD | engine.FileE | engine.FileF)

	// Count squares in spaceMask not attacked by enemy pawns
	safeSpace := spaceMask & ^enemyPawnAttacks
	score = safeSpace.Count() * SpaceMG

	return score
}

// evaluateMopUp encourages pushing the enemy king to the corners when ahead.
func evaluateMopUp(b *engine.Board, score, phase int) int {
	// Only applicable if one side has a significant material advantage
	// and we are approaching the endgame.
	if phase > 10 {
		return 0
	}

	bonus := 0
	us := types.White
	them := types.Black
	if score < 0 {
		us = types.Black
		them = types.White
	}

	enemyKingSq := b.Pieces[them][types.King].LSB()
	ourKingSq := b.Pieces[us][types.King].LSB()

	// 1. Centralization bonus (push enemy king to edges)
	bonus += engine.CenterDistance(enemyKingSq) * MopUpBonus

	// 2. Proximity bonus (keep our king near enemy king)
	dist := engine.ManhattanDistance(ourKingSq, enemyKingSq)
	bonus += (14 - dist) * (MopUpBonus / 2)

	// Scale by phase (stronger in deep endgame)
	bonus = (bonus * (TotalPhase - phase)) / TotalPhase

	if score < 0 {
		return -bonus
	}
	return bonus
}

// evaluateKingPassedPawnProximity rewards the king for being near passed pawns in the endgame.
func evaluateKingPassedPawnProximity(b *engine.Board, c types.Color) int {
	bonus := 0
	kingSq := b.Pieces[c][types.King].LSB()
	them := c ^ 1

	// Support our own passed pawns
	pawns := b.Pieces[c][types.Pawn]
	for pawns != 0 {
		sq := pawns.PopLSB()
		if isPassed(b, c, sq) {
			dist := engine.ManhattanDistance(kingSq, sq)
			bonus += (7 - dist) * KingNearPassedPawnEG
		}
	}

	// Stay near enemy passed pawns to stop them
	enemyPawns := b.Pieces[them][types.Pawn]
	for enemyPawns != 0 {
		sq := enemyPawns.PopLSB()
		if isPassed(b, them, sq) {
			dist := engine.ManhattanDistance(kingSq, sq)
			bonus += (7 - dist) * KingNearPassedPawnEG
		}
	}

	return bonus
}

// isPassed is a helper to determine if a pawn is passed.
func isPassed(b *engine.Board, c types.Color, sq types.Square) bool {
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

// getPST maps a square to its value in the Piece-Square Table.
func getPST(sq types.Square, c types.Color) int {
	rank := int(sq) / 8
	file := int(sq) % 8
	if c == types.White {
		return (7-rank)*8 + file
	}
	return rank*8 + file
}

// isInsufficientMaterial returns true if the position is a forced draw by rule.
func isInsufficientMaterial(b *engine.Board) bool {
	if b.Pieces[types.White][types.Pawn] != 0 || b.Pieces[types.Black][types.Pawn] != 0 {
		return false
	}
	if b.Pieces[types.White][types.Rook] != 0 || b.Pieces[types.Black][types.Rook] != 0 ||
		b.Pieces[types.White][types.Queen] != 0 || b.Pieces[types.Black][types.Queen] != 0 {
		return false
	}

	wKnights := b.Pieces[types.White][types.Knight].Count()
	wBishops := b.Pieces[types.White][types.Bishop].Count()
	bKnights := b.Pieces[types.Black][types.Knight].Count()
	bBishops := b.Pieces[types.Black][types.Bishop].Count()

	if wKnights == 0 && wBishops == 0 && bKnights == 0 && bBishops == 0 {
		return true // K vs K
	}
	if wKnights == 1 && wBishops == 0 && bKnights == 0 && bBishops == 0 {
		return true // KN vs K
	}
	if wKnights == 0 && wBishops == 1 && bKnights == 0 && bBishops == 0 {
		return true // KB vs K
	}
	if wKnights == 0 && wBishops == 0 && bKnights == 1 && bBishops == 0 {
		return true // K vs KN
	}
	if wKnights == 0 && wBishops == 0 && bKnights == 0 && bBishops == 1 {
		return true // K vs KB
	}

	return false
}

// scaleEndgame adjusts the evaluation for known drawish or favorable endgame patterns.
func scaleEndgame(b *engine.Board, score int) int {
	// 1. Basic scaling for pawnless endgames
	if score > 0 && b.Pieces[types.White][types.Pawn] == 0 {
		score = score * 3 / 4
	} else if score < 0 && b.Pieces[types.Black][types.Pawn] == 0 {
		score = score * 3 / 4
	}

	// 2. Opposite Colored Bishops
	if isOppositeBishops(b) {
		// In pure OCB endgames with few pawns, drawishness is very high.
		numPawns := b.Pieces[types.White][types.Pawn].Count() + b.Pieces[types.Black][types.Pawn].Count()
		if numPawns <= 2 {
			score /= 2
		} else if numPawns <= 4 {
			score = score * 3 / 4
		}
	}

	return score
}

// isOppositeBishops returns true if both sides have exactly one bishop and they are on different colors.
func isOppositeBishops(b *engine.Board) bool {
	if b.Pieces[types.White][types.Bishop].Count() != 1 || b.Pieces[types.Black][types.Bishop].Count() != 1 {
		return false
	}

	if b.Pieces[types.White][types.Knight] != 0 || b.Pieces[types.Black][types.Knight] != 0 ||
		b.Pieces[types.White][types.Rook] != 0 || b.Pieces[types.Black][types.Rook] != 0 ||
		b.Pieces[types.White][types.Queen] != 0 || b.Pieces[types.Black][types.Queen] != 0 {
		return false
	}

	wSq := b.Pieces[types.White][types.Bishop].LSB()
	bSq := b.Pieces[types.Black][types.Bishop].LSB()

	wColor := (int(wSq/8) + int(wSq%8)) & 1
	bColor := (int(bSq/8) + int(bSq%8)) & 1

	return wColor != bColor
}
