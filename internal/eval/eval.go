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

	if nnue.UseNNUE && nnue.CurrentNetwork != nil {
		return nnue.EvaluateForward(&b.Accumulators[0], &b.Accumulators[1], b.SideToMove)
	}

	mgW, egW, _ := calculatePhase(b)

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

	mgScore := mgWhite - mgBlack
	egScore := egWhite - egBlack

	score := (mgScore*mgW + egScore*egW) / TotalPhase

	// Scale evaluation in drawish endgames
	score = scaleEndgame(b, score)

	if b.SideToMove == types.Black {
		return -score
	}
	return score
}

// calculatePhase determines the game phase for tapered evaluation.
func calculatePhase(b *engine.Board) (int, int, int) {
	phase := TotalPhase
	phase -= (b.Pieces[types.White][types.Knight].Count() + b.Pieces[types.Black][types.Knight].Count()) * KnightPhase
	phase -= (b.Pieces[types.White][types.Bishop].Count() + b.Pieces[types.Black][types.Bishop].Count()) * BishopPhase
	phase -= (b.Pieces[types.White][types.Rook].Count() + b.Pieces[types.Black][types.Rook].Count()) * RookPhase
	phase -= (b.Pieces[types.White][types.Queen].Count() + b.Pieces[types.Black][types.Queen].Count()) * QueenPhase

	if phase < 0 {
		phase = 0
	}

	egW := phase
	mgW := TotalPhase - phase
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
