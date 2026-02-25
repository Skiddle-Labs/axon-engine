package eval

import (
	"github.com/Skiddle-Labs/axon-engine/internal/engine"
)

// Evaluate returns a score for the current board position using tapered evaluation.
func Evaluate(b *engine.Board) int {
	if isInsufficientMaterial(b) {
		return 0
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
		pMgW, pEgW = evaluatePawnStructure(b, engine.White)
		pMgB, pEgB = evaluatePawnStructure(b, engine.Black)
		GlobalPawnTable.Store(b.PawnHash, pMgW, pEgW, pMgB, pEgB)
	}

	mgWhite, egWhite := evaluateColor(b, engine.White, pMgW, pEgW)
	mgBlack, egBlack := evaluateColor(b, engine.Black, pMgB, pEgB)

	mgScore := mgWhite - mgBlack
	egScore := egWhite - egBlack

	score := (mgScore*mgW + egScore*egW) / TotalPhase

	// Scale evaluation in drawish endgames
	if score > 0 && b.Pieces[engine.White][engine.Pawn] == 0 {
		score = score * 3 / 4
	} else if score < 0 && b.Pieces[engine.Black][engine.Pawn] == 0 {
		score = score * 3 / 4
	}

	if b.SideToMove == engine.Black {
		return -score
	}
	return score
}

// calculatePhase determines the game phase for tapered evaluation.
func calculatePhase(b *engine.Board) (int, int, int) {
	phase := TotalPhase
	phase -= (b.Pieces[engine.White][engine.Knight].Count() + b.Pieces[engine.Black][engine.Knight].Count()) * KnightPhase
	phase -= (b.Pieces[engine.White][engine.Bishop].Count() + b.Pieces[engine.Black][engine.Bishop].Count()) * BishopPhase
	phase -= (b.Pieces[engine.White][engine.Rook].Count() + b.Pieces[engine.Black][engine.Rook].Count()) * RookPhase
	phase -= (b.Pieces[engine.White][engine.Queen].Count() + b.Pieces[engine.Black][engine.Queen].Count()) * QueenPhase

	if phase < 0 {
		phase = 0
	}

	egW := phase
	mgW := TotalPhase - phase
	return mgW, egW, phase
}

// evaluateColor computes the evaluation for a single side.
func evaluateColor(b *engine.Board, c engine.Color, pMg, pEg int) (int, int) {
	mg, eg := pMg, pEg

	// 1. PST and Piece Material
	// We handle piece material in evaluatePieces and non-pawn PST here.
	for pt := engine.Knight; pt <= engine.King; pt++ {
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

	return mg, eg
}

// evaluatePawnStructure calculates pawn-only scores (material, PST, structure).
func evaluatePawnStructure(b *engine.Board, c engine.Color) (int, int) {
	mg, eg := 0, 0

	pawns := b.Pieces[c][engine.Pawn]
	for pawns != 0 {
		sq := pawns.PopLSB()
		idx := getPST(sq, c)
		mg += MgPST[engine.Pawn][idx] + PawnMG
		eg += EgPST[engine.Pawn][idx] + PawnEG
	}

	pmg, peg := evaluatePawns(b, c)
	return mg + pmg, eg + peg
}

// getPST maps a square to its value in the Piece-Square Table.
func getPST(sq engine.Square, c engine.Color) int {
	rank := int(sq) / 8
	file := int(sq) % 8
	if c == engine.White {
		return (7-rank)*8 + file
	}
	return rank*8 + file
}

// isInsufficientMaterial returns true if the position is a forced draw by rule.
func isInsufficientMaterial(b *engine.Board) bool {
	if b.Pieces[engine.White][engine.Pawn] != 0 || b.Pieces[engine.Black][engine.Pawn] != 0 {
		return false
	}
	if b.Pieces[engine.White][engine.Rook] != 0 || b.Pieces[engine.Black][engine.Rook] != 0 ||
		b.Pieces[engine.White][engine.Queen] != 0 || b.Pieces[engine.Black][engine.Queen] != 0 {
		return false
	}

	wKnights := b.Pieces[engine.White][engine.Knight].Count()
	wBishops := b.Pieces[engine.White][engine.Bishop].Count()
	bKnights := b.Pieces[engine.Black][engine.Knight].Count()
	bBishops := b.Pieces[engine.Black][engine.Bishop].Count()

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
