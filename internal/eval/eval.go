package eval

import (
	"github.com/Skiddle-Labs/axon-engine/internal/engine"
)

// Evaluate returns a score for the current board position using tapered evaluation.
func Evaluate(b *engine.Board) int {
	mgW, egW, _ := calculatePhase(b)

	mgWhite, egWhite := evaluateColor(b, engine.White)
	mgBlack, egBlack := evaluateColor(b, engine.Black)

	mgScore := mgWhite - mgBlack
	egScore := egWhite - egBlack

	score := (mgScore*mgW + egScore*egW) / TotalPhase

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
func evaluateColor(b *engine.Board, c engine.Color) (int, int) {
	mg, eg := 0, 0

	// 1. PST and Pawn Material
	// We handle piece material in evaluatePieces and pawn material here.
	for pt := engine.Pawn; pt <= engine.King; pt++ {
		pieces := b.Pieces[c][pt]
		for pieces != 0 {
			sq := pieces.PopLSB()
			idx := getPST(sq, c)
			mg += MgPST[pt][idx]
			eg += EgPST[pt][idx]

			if pt == engine.Pawn {
				mg += PawnMG
				eg += PawnEG
			}
		}
	}

	// 2. Pawns: Structure, Passed, Storm
	pmg, peg := evaluatePawns(b, c)
	mg += pmg
	eg += peg

	// 3. Pieces: Material, Mobility, Outposts, Bishop Pair
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

// getPST maps a square to its value in the Piece-Square Table.
func getPST(sq engine.Square, c engine.Color) int {
	rank := int(sq) / 8
	file := int(sq) % 8
	if c == engine.White {
		return (7-rank)*8 + file
	}
	return rank*8 + file
}
