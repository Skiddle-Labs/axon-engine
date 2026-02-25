package search

import (
	"sync/atomic"
	"time"

	"github.com/Skiddle-Labs/axon-engine/internal/engine"
	"github.com/Skiddle-Labs/axon-engine/internal/eval"
	"github.com/Skiddle-Labs/axon-engine/internal/types"
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
	if found && excludedMove == engine.NoMove && (ply > 0 || len(e.RootExcludedMoves) == 0) {
		return ttScore
	}

	// 3. Check detection and extension
	inCheck := e.Board.IsSquareAttacked(e.Board.Pieces[e.Board.SideToMove][types.King].LSB(), e.Board.SideToMove^1)
	if inCheck {
		depth++
	}

	// 4. Base case: Depth reached or Quiescence Search
	if depth <= 0 {
		return e.quiescence(alpha, beta, ply)
	}

	staticEval := e.ApplyCorrection(eval.Evaluate(e.Board))

	// 5. Reverse Futility Pruning (RFP)
	// If static evaluation is significantly above beta, we can skip the search.
	if depth < 5 && !inCheck && excludedMove == engine.NoMove && ply > 0 && beta < MateScore-1000 {
		margin := depth * RFPMargin
		if staticEval-margin >= beta {
			return beta
		}
	}

	// 6. Internal Iterative Deepening (IID)
	// If we don't have a TT move at a high depth, perform a shallow search to find one.
	if depth >= 5 && ttMove == engine.NoMove && !inCheck && ply > 0 {
		e.negamax(depth-2, alpha, beta, ply, engine.NoMove)
		_, ttMove, _ = e.TT.Probe(e.Board.Hash, depth, alpha, beta, ply)
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

	// 8. ProbCut
	// If we are likely to fail high at a much higher beta, prune the branch.
	if depth >= 5 && !inCheck && ply > 0 && excludedMove == engine.NoMove && beta < MateScore-1000 {
		probBeta := beta + 200
		probDepth := depth - 4

		score := e.negamax(probDepth, probBeta-1, probBeta, ply, engine.NoMove)
		if score >= probBeta {
			return beta
		}
	}

	// 9. Null Move Pruning (NMP)
	// If we can afford to skip a move and still fail high, we can prune the branch.
	if depth >= 3 && !inCheck && ply > 0 && excludedMove == engine.NoMove && e.Board.HasMajorPieces(e.Board.SideToMove) {
		e.Board.MakeNullMove()
		// Adaptive Null Move Reduction
		r := NMPBase + depth/NMPDivisor
		score := -e.negamax(depth-1-r, -beta, -beta+1, ply+1, engine.NoMove)
		e.Board.UnmakeNullMove()
		if score >= beta {
			return beta
		}
	}

	// 10. Move Generation and Ordering
	ml := e.Board.GenerateMoves()
	e.orderMoves(&ml, ttMove, ply)

	alphaOrig := alpha
	bestMove := engine.NoMove
	bestScore := -Infinity
	legalMoves := 0
	var triedMoves [256]engine.Move

	// 11. Search Loop
	for i := 0; i < ml.Count; i++ {
		move := ml.Moves[i]
		if move == excludedMove {
			continue
		}

		// MultiPV check: skip moves already chosen for previous PVs
		if ply == 0 {
			isExcluded := false
			for _, m := range e.RootExcludedMoves {
				if move == m {
					isExcluded = true
					break
				}
			}
			if isExcluded {
				continue
			}
		}

		isCapture := move.Flags()&engine.CaptureFlag != 0
		isPromotion := move.Flags()&0x8000 != 0

		// Passed Pawn Extension
		pawnExtension := 0
		if !inCheck && depth > 2 {
			piece := e.Board.PieceAt(move.From()).Type()
			if piece == types.Pawn {
				to := move.To()
				rank := to.Rank()
				if (e.Board.SideToMove == types.White && rank >= 5) ||
					(e.Board.SideToMove == types.Black && rank <= 2) {
					// Check if passed pawn
					us := e.Board.SideToMove
					them := us ^ 1
					enemyPawns := e.Board.Pieces[them][types.Pawn]
					file := to.File()

					frontMask := engine.Bitboard(0)
					if us == types.White {
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
						pawnExtension = 1
					}
				}
			}
		}

		// Static Exchange Evaluation (SEE) pruning
		// Prune captures that are clearly losing material at low depths.
		// We avoid pruning moves at the root or moves that might be tactical (promotions/checks).
		if depth <= 4 && isCapture && legalMoves > 0 && !inCheck && ply > 0 && !isPromotion {
			if e.Board.SEE(move) < -50*depth {
				continue
			}
		}

		if !e.Board.MakeMove(move) {
			continue
		}

		// Check if the move resulted in a check
		givesCheck := e.Board.IsSquareAttacked(e.Board.Pieces[e.Board.SideToMove][types.King].LSB(), e.Board.SideToMove^1)

		// Late Move Pruning (LMP)
		// Prune quiet moves late in the list at low depths.
		if depth <= 4 && !inCheck && legalMoves > (5+depth*depth) && !isCapture && !isPromotion && !givesCheck {
			e.Board.UnmakeMove(move)
			continue
		}

		// Futility Pruning (FP)
		// Prune quiet moves if static eval is too far below alpha.
		if depth <= 3 && !inCheck && legalMoves > 0 && !isCapture && !isPromotion && !givesCheck {
			margin := depth * FPMargin
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
			score = -e.negamax(depth-1+extension+pawnExtension, -beta, -alpha, ply+1, engine.NoMove)
		} else {
			// 12. Late Move Reduction (LMR)
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
			score = -e.negamax(depth-1-reduction+pawnExtension, -(alpha + 1), -alpha, ply+1, engine.NoMove)
			if score > alpha && reduction > 0 {
				score = -e.negamax(depth-1+pawnExtension, -(alpha + 1), -alpha, ply+1, engine.NoMove)
			}
			if score > alpha && score < beta {
				score = -e.negamax(depth-1+pawnExtension, -beta, -alpha, ply+1, engine.NoMove)
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

	// 13. Terminal node handling
	if legalMoves == 0 {
		if inCheck {
			return -MateScore + ply // Mate
		}
		return 0 // Stalemate
	}

	// 14. TT Storage
	if excludedMove == engine.NoMove {
		// Update Correction History for non-mate scores
		if !inCheck && bestScore > -MateScore+1000 && bestScore < MateScore-1000 {
			e.UpdateCorrection(depth, bestScore, staticEval)
		}

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
