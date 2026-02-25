package search

import (
	"sync/atomic"
	"time"

	"github.com/Skiddle-Labs/axon-engine/internal/engine"
	"github.com/Skiddle-Labs/axon-engine/internal/eval"
)

// negamax is the core recursive search function of the engine.
// It implements Alpha-Beta pruning with several modern enhancements:
// RFP, NMP, IIR, Singular Extensions, LMP, LMR, and Futility Pruning.
func (e *Engine) negamax(depth, alpha, beta, ply int, excludedMove engine.Move) int {
	e.localNodes++
	if e.localNodes >= 2048 {
		e.syncNodes()
		nodes := atomic.LoadUint64(e.Nodes)
		if (e.TimeLimit > 0 && time.Since(e.StartTime) >= e.TimeLimit) ||
			(e.NodesLimit > 0 && nodes >= e.NodesLimit) {
			atomic.StoreInt32(e.Stopped, 1)
		}
	}

	if atomic.LoadInt32(e.Stopped) != 0 {
		return 0
	}

	// 1. Repetition and Draw detection
	if ply > 0 {
		for i := e.Board.Ply - 2; i >= e.Board.Ply-int(e.Board.HalfMoveClock); i -= 2 {
			if i >= 0 && e.Board.History[i].Hash == e.Board.Hash {
				return 0
			}
		}
	}

	// 2. Transposition Table Probe
	ttScore, ttMove, found := e.TT.Probe(e.Board.Hash, depth, alpha, beta, ply)
	if found && excludedMove == engine.NoMove {
		return ttScore
	}

	// 3. Check detection and extension
	inCheck := e.Board.IsSquareAttacked(e.Board.Pieces[e.Board.SideToMove][engine.King].LSB(), e.Board.SideToMove^1)
	if inCheck {
		depth++
	}

	// 4. Base case: Depth reached or Quiescence Search
	if depth <= 0 {
		return e.quiescence(alpha, beta, ply)
	}

	staticEval := eval.Evaluate(e.Board)

	// 5. Reverse Futility Pruning (RFP)
	// If static evaluation is significantly above beta, we can skip the search.
	if depth < 5 && !inCheck && excludedMove == engine.NoMove && ply > 0 && beta < MateScore-1000 {
		margin := depth * 75
		if staticEval-margin >= beta {
			return beta
		}
	}

	// 6. Internal Iterative Reductions (IIR)
	// If we don't have a TT move at a high depth, reduce slightly to avoid expensive blind search.
	if depth >= 3 && ttMove == engine.NoMove && !inCheck && ply > 0 {
		depth--
	}

	// 7. Singular Extensions
	// If the TT move is significantly better than other moves, extend the search.
	extension := 0
	if depth >= 8 && ttMove != engine.NoMove && excludedMove == engine.NoMove && found {
		singularBeta := ttScore - 2*depth
		score := e.negamax((depth-1)/2, singularBeta-1, singularBeta, ply, ttMove)
		if score < singularBeta {
			extension = 1
		}
	}

	// 8. Null Move Pruning (NMP)
	// If we can afford to skip a move and still fail high, we can prune the branch.
	if depth >= 3 && !inCheck && ply > 0 && excludedMove == engine.NoMove && e.Board.HasMajorPieces(e.Board.SideToMove) {
		e.Board.MakeNullMove()
		// Adaptive Null Move Reduction: R = 3 + depth / 6
		r := 3 + depth/6
		score := -e.negamax(depth-1-r, -beta, -beta+1, ply+1, engine.NoMove)
		e.Board.UnmakeNullMove()
		if score >= beta {
			return beta
		}
	}

	// 9. Move Generation and Ordering
	ml := e.Board.GenerateMoves()
	e.orderMoves(&ml, ttMove, ply)

	alphaOrig := alpha
	bestMove := engine.NoMove
	bestScore := -Infinity
	legalMoves := 0
	var triedMoves [256]engine.Move

	// 10. Search Loop
	for i := 0; i < ml.Count; i++ {
		move := ml.Moves[i]
		if move == excludedMove {
			continue
		}

		isCapture := move.Flags()&engine.CaptureFlag != 0
		isPromotion := move.Flags()&0x8000 != 0

		if !e.Board.MakeMove(move) {
			continue
		}

		// Check if the move resulted in a check
		givesCheck := e.Board.IsSquareAttacked(e.Board.Pieces[e.Board.SideToMove][engine.King].LSB(), e.Board.SideToMove^1)

		// Late Move Pruning (LMP)
		// Prune quiet moves late in the list at low depths.
		if depth <= 4 && !inCheck && legalMoves > (5+depth*depth) && !isCapture && !isPromotion && !givesCheck {
			e.Board.UnmakeMove(move)
			continue
		}

		// Futility Pruning (FP)
		// Prune quiet moves if static eval is too far below alpha.
		if depth <= 3 && !inCheck && legalMoves > 0 && !isCapture && !isPromotion && !givesCheck {
			margin := depth * 100
			if staticEval+margin < alpha {
				e.Board.UnmakeMove(move)
				continue
			}
		}

		triedMoves[legalMoves] = move
		legalMoves++

		var score int
		if legalMoves == 1 {
			// Search Principal Variation fully
			score = -e.negamax(depth-1+extension, -beta, -alpha, ply+1, engine.NoMove)
		} else {
			// 11. Late Move Reduction (LMR)
			// Reduce the search depth for moves that are unlikely to be the best.
			reduction := 0
			if depth >= 3 && legalMoves > 4 && !inCheck && !isCapture && !isPromotion {
				reduction = getReduction(depth, legalMoves)

				// Refine reduction based on move characteristics
				if givesCheck {
					reduction--
				}
				if ply < 128 && (move == e.KillerMoves[ply][0] || move == e.KillerMoves[ply][1]) {
					reduction--
				}
				if reduction < 0 {
					reduction = 0
				}
			}

			// PVS (Principal Variation Search)
			score = -e.negamax(depth-1-reduction, -(alpha + 1), -alpha, ply+1, engine.NoMove)
			if score > alpha && reduction > 0 {
				score = -e.negamax(depth-1, -(alpha + 1), -alpha, ply+1, engine.NoMove)
			}
			if score > alpha && score < beta {
				score = -e.negamax(depth-1, -beta, -alpha, ply+1, engine.NoMove)
			}
		}

		e.Board.UnmakeMove(move)

		if atomic.LoadInt32(e.Stopped) != 0 {
			return 0
		}

		if score > bestScore {
			bestScore = score
			bestMove = move
		}

		if score > alpha {
			alpha = score
		}

		// Beta Cutoff (Fail-High)
		if score >= beta {
			// Update search heuristics
			if !isCapture && ply < 128 {
				// Killer moves
				if move != e.KillerMoves[ply][0] {
					e.KillerMoves[ply][1] = e.KillerMoves[ply][0]
					e.KillerMoves[ply][0] = move
				}

				// History bonus
				piece := e.Board.PieceAt(move.From()).Type()
				if e.HistoryTable != nil {
					e.HistoryTable[e.Board.SideToMove][piece][move.To()] += depth * depth
				}

				// Countermoves
				if e.Board.Ply > 0 && e.CounterMoves != nil {
					prevMove := e.Board.History[e.Board.Ply-1].Move
					e.CounterMoves[prevMove.From()][prevMove.To()] = move
				}

				// History penalty for failed quiet moves in this node
				if e.HistoryTable != nil {
					for j := 0; j < legalMoves-1; j++ {
						m := triedMoves[j]
						if m.Flags()&engine.CaptureFlag == 0 {
							p := e.Board.PieceAt(m.From()).Type()
							e.HistoryTable[e.Board.SideToMove][p][m.To()] -= depth * depth
						}
					}
				}
			}
			break
		}
	}

	// 12. Terminal node handling
	if legalMoves == 0 {
		if inCheck {
			return -MateScore + ply // Mate
		}
		return 0 // Stalemate
	}

	// 13. TT Storage
	if excludedMove == engine.NoMove {
		flag := ExactFlag
		if bestScore <= alphaOrig {
			flag = AlphaFlag
		} else if bestScore >= beta {
			flag = BetaFlag
		}
		e.TT.Store(e.Board.Hash, depth, bestScore, flag, bestMove, ply)
	}

	return bestScore
}

// quiescence search handles capture resolutions at the leaves of the search tree
// to avoid the "horizon effect".
func (e *Engine) quiescence(alpha, beta, ply int) int {
	e.localNodes++
	if e.localNodes >= 2048 {
		e.syncNodes()
	}

	if atomic.LoadInt32(e.Stopped) != 0 {
		return 0
	}

	// Standing pat: if the current position is good enough to fail high, stop.
	standingPat := eval.Evaluate(e.Board)
	if standingPat >= beta {
		return beta
	}
	if standingPat > alpha {
		alpha = standingPat
	}

	// Only search captures in quiescence
	ml := e.Board.GenerateCaptures()
	e.orderMoves(&ml, engine.NoMove, ply)

	for i := 0; i < ml.Count; i++ {
		move := ml.Moves[i]

		// Static Exchange Evaluation (SEE) pruning
		// Skip captures that are clearly losing material.
		if e.Board.SEE(move) < 0 {
			continue
		}

		if !e.Board.MakeMove(move) {
			continue
		}
		score := -e.quiescence(-beta, -alpha, ply+1)
		e.Board.UnmakeMove(move)

		if score >= beta {
			return beta
		}
		if score > alpha {
			alpha = score
		}
	}

	return alpha
}
