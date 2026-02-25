# Axon Engine Roadmap

This document tracks the planned improvements and current progress of the Axon chess engine.

## 1. Search Enhancements
- [x] **Static Null Move Pruning (Reverse Futility Pruning)**: Prune nodes where static evaluation is significantly above beta at low depths.
- [x] **Late Move Pruning (LMP)**: Prune quiet moves at high move counts in late search nodes.
- [x] **Dynamic Time Management**: Implement evaluation-based and move-stability-based time allocation.
- [x] **Singular Extensions**: Extend the search for moves that are significantly better than the second-best move.
- [x] **Countermove Heuristic**: Improve move ordering by tracking successful responses to specific moves.
- [x] **Aspiration Windows**: Reduce search nodes by using a narrow score window around the previous depth's evaluation.
- [x] **LMR Refinement**: Precomputed logarithmic reduction table scaling with both depth and move index.
- [x] **Adaptive Null Move Pruning**: Dynamic reduction formula (`3 + depth/6`) for more aggressive pruning at high depths.
- [x] **Futility Pruning**: Skip quiet moves at low depths when static evaluation plus a margin fails to reach alpha.
- [x] **History Penalty (Negative History)**: Sharpen move ordering by penalizing quiet moves that failed to cause a cutoff.
- [x] **Piece-Specific History**: Multi-dimensional history table tracking successes by side and piece type.

## 2. Evaluation Improvements (Hybrid Strategy)
- [x] **Texel Tuning**: Automated optimization of current HCE parameters (PST, material, mobility) using game results.
- [x] **Data Generation**: Implement a self-play tool (`cmd/datagen`) to generate tuning data from engine-vs-engine games.
- [x] **Enhanced Pawn Evaluation**: Logic for pawn shields, passed pawns, and connected structures.
- [x] **King Safety Refinement**: Granular pawn shield and attacking zone evaluation with piece-specific weights.
- [x] **SPSA Tuning**: Simultaneous Perturbation Stochastic Approximation for robust high-dimensional parameter optimization.
- [x] **Enhanced Positional Evaluation**: Logic for outposts, open/half-open files, and bishop-pair scaling.
- [x] **Pawn Storm Detection**: Specifically penalize/reward the advancement of pawns toward the enemy king.
    - *Note: New parameters added. Must run a short SPSA (5k-10k iterations) to calibrate these weights after the current 50k run finishes.*
- [ ] **NNUE Implementation**: Implement a neural network evaluation as a final step while retaining HCE for speed and data generation.

## 3. Infrastructure & Features (Current Focus)
- [x] **Benchmarking Tool**: Internal `bench` command for consistent performance and node-count measurement.
- [x] **Opening Book Support**: Polyglot (.bin) support with weighted random and best-move selection.
- [x] **Automated Parameter Integration**: Tooling to automatically apply tuner results (`.txt`) back into the Go source code.
- [x] **Modular Search Architecture**: Decoupled search logic into specialized components (ordering, LMR, negamax) for better maintainability.
- [x] **Logging System**: Implement structured logging for better debugging in engine-vs-engine matches.

## 4. Optimization
- [x] **Lockless Transposition Table**: High-performance, atomic-based TT to reduce contention in multi-threaded searches.
- [ ] **SIMD Optimizations**: Use SIMD instructions for critical paths like bitboard operations or future NNUE inference.

---
*Last Updated: February 2026*