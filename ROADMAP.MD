# Axon Engine Roadmap

This document tracks the planned improvements and current progress of the Axon chess engine.

## 1. Search Enhancements
- [x] **Static Null Move Pruning (Reverse Futility Pruning)**
- [x] **Late Move Pruning (LMP)**
- [x] **Dynamic Time Management**
- [x] **Singular Extensions**
- [x] **Countermove Heuristic**
- [x] **Aspiration Windows**
- [x] **LMR Refinement**
- [x] **Adaptive Null Move Pruning**
- [x] **Futility Pruning**
- [x] **History Penalty (Negative History)**
- [x] **Piece-Specific History**
- [x] **SEE-based Pruning**
- [x] **ProbCut**
- [x] **Passed Pawn Extensions**
- [x] **Internal Iterative Deepening (IID)**
 
 ## 2. Evaluation Improvements (Hybrid Strategy)
- [x] **Texel Tuning**: Automated optimization of HCE parameters.
- [x] **Enhanced Pawn Evaluation**: Shield, passed, and connected logic.
- [x] **King Safety Refinement**: Attacking zone model with piece-specific weights.
- [x] **Non-Linear Mobility**: Use tables instead of linear multipliers.
- [x] **Virtual Mobility**: Penalties/bonuses for pressure on occupied squares.
- [x] **Pawn Storm Detection**: Calibration of advancing enemy pawns.
- [x] **NNUE Implementation**:
    - [x] **Architecture Design**: 768 -> 256 -> 1 topology.
    - [x] **Incremental Update Hooks**: Optimized `addPiece`/`removePiece`.
    - [x] **SIMD Optimizations**: AVX2 assembly kernels for accumulator updates.
    - [x] **Inference Engine**: SCReLU-based forward pass.
- [/] **NNUE Training**: Currently generating 1M+ positions for the first model.

## 3. Infrastructure & Features
- [x] **Benchmarking Tool**: Internal `bench` command.
- [x] **Opening Book Support**: Polyglot (.bin) support.
- [x] **Automated Parameter Integration**: `cmd/apply` tool.
- [x] **Modular UCI Protocol**: Dedicated package for protocol handling.
- [x] **High-Performance Datagen**: Multi-threaded self-play for NNUE data.
- [x] **SIMD CPU Detection**: Runtime detection for AVX2 support with Go fallbacks.

## 4. Optimization & Polish (Current Focus)
- [x] **Lockless Transposition Table**
- [x] **Pawn Hash Table**
- [ ] **Fast FEN Generation**: Buffer-based FEN formatting to boost datagen NPS.
- [ ] **Tablebase Support**: Integration of Syzygy WDL/DTZ probing.

---
*Last Updated: February 27, 2026*
