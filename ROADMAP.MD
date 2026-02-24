# Axon Engine Roadmap

This document tracks the planned improvements and current progress of the Axon chess engine.

## 1. Search Enhancements (Current Focus)
- [x] **Static Null Move Pruning (Reverse Futility Pruning)**: Prune nodes where static evaluation is significantly above beta at low depths.
- [x] **Late Move Pruning (LMP)**: Prune quiet moves at high move counts in late search nodes.
- [x] **Dynamic Time Management**: Implement evaluation-based and move-stability-based time allocation.
- [x] **Singular Extensions**: Extend the search for moves that are significantly better than the second-best move.
- [x] **Countermove Heuristic**: Improve move ordering by tracking successful responses to specific moves.

## 2. Evaluation Improvements (Hybrid Strategy)
- [x] **Texel Tuning**: Automated optimization of current HCE parameters (PST, material, mobility) using game results.
- [x] **Data Generation**: Implement a self-play tool (`cmd/datagen`) to generate tuning data from engine-vs-engine games.
- [x] **Enhanced Pawn Evaluation**: Add specific logic for pawn storms, space control, and sophisticated outposts.
- [x] **King Safety Refinement**: Implement a more granular pawn shield and attacking zone evaluation.
- [ ] **NNUE Implementation**: Implement a neural network evaluation as a final step while retaining HCE for speed and data generation.

## 3. Infrastructure & Features
- [ ] **Syzygy Tablebase Support**: Perfect endgame play using 3-4-5-6 piece tablebases (Scaffolding implemented).
- [x] **Benchmarking Tool**: Implement a `bench` command for consistent performance measurement across different builds.
- [x] **Opening Book Support**: Add Polyglot (.bin) support with weighted random and best-move selection strategies.
- [ ] **Logging System**: Implement structured logging for better debugging in engine-vs-engine matches.

## 4. Optimization
- [ ] **SIMD Optimizations**: Use SIMD instructions for critical paths like NNUE inference or bitboard operations.
- [ ] **Lockless Transposition Table**: Refine the TT implementation to reduce contention in high-thread scenarios.

---
*Last Updated: February 2026*
