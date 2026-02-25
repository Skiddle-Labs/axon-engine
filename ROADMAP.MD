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
- [ ] **Specialized Endgame Scaling**: Adjust scores for known drawish endings (e.g., Opposite Colored Bishops).
- [ ] **Trapped Piece Detection**: Penalize pieces with zero mobility or those blocked by their own pawns.
- [ ] **Connected Passed Pawns**: Bonuses for passed pawns that support each other's advancement.
- [ ] **Bishop vs. Knight Scaling**: Dynamic adjustment of piece values based on pawn density.
- [ ] **Space Evaluation**: Positional bonuses for controlling the central four ranks.
- [ ] **King Ring Attacks**: Precise tactical model focusing on the immediate 8 squares around the king.

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
- [ ] **En Passant Refinement**: Robustify Zobrist hashing and evaluation (Passed Pawn awareness) for En Passant.

## 5. Advanced Tuning & Calibration
- [ ] **Log Loss (Cross-Entropy)**: Add support for Log Loss in the tuner to improve NNUE/HCE calibration.
- [ ] **ELO-Based SPSA**: Direct optimization for match performance.
- [ ] **Sigmoid Scaling (K-Factor) Analysis**: Tools to visualize how well the engine's scores match real-world probabilities.

## 6. Datagen & Dataset Pipeline
- [ ] **Lock-free Output Buffering**: Minimize mutex contention when writing results from many threads.
- [ ] **SIMD-Accelerated FEN/EPD**: Use SIMD for piece-to-char conversion and string formatting.
- [ ] **Batch Search**: Optimize search specifically for bulk data generation at low depths.
- [ ] **Binary Format Support**: Support for `.bin` or `.bullet` formats to eliminate string parsing overhead.

## 7. Testing & Quality Assurance
- [ ] **SPRT Framework**: Integrated Sequential Probability Ratio Testing for ELO validation.
- [ ] **Regression Suites**: Automated testing against previous releases.
- [ ] **UCI Option Support**: Expose search knobs via UCI for easier tuning.

## 8. Advanced Search & Optimization
- [ ] **Multi-Cut Pruning**: Prune nodes if multiple moves fail high at reduced depth.
- [ ] **Memory Prefetching**: Optimize TT lookups by prefetching memory before access.
- [ ] **AVX-512 NNUE**: Specialized kernels for ultra-high performance on modern hardware.
- [ ] **Double Extensions**: Detect and extend search for extremely critical tactical lines.
- [ ] **Internal Iterative Reductions (IIR)**: Reduce search depth when no TT move is available.
- [ ] **History-Based LMR**: Scale move reductions based on historical success/failure of moves.
- [ ] **Correction History**: Learn and apply systematic errors in static evaluation during search.
- [ ] **Capture History**: Use historical capture success to improve move ordering in Quiescence search.

## 9. Development & Support Tooling
- [ ] **EPD Processing Suite**: Specialized tools for shuffling, merging, and filtering blunders from datasets.
- [ ] **Automated SPRT Controller**: Scripts to manage Cutechess-cli matches and report statistical Elo gains.
- [ ] **Search Visualization**: Tools to export and visualize search trees for debugging pruning behavior.

## 10. Quality of Life & Developer Experience
- [ ] **Integrated Perft**: Built-in move generation speed and correctness tester.
- [ ] **Colored Board Display**: ANSI colors for the `display` command to make terminal testing more intuitive.
- [ ] **Logging System**: Dedicated logging to file for debugging long matches and unexpected crashes.
- [ ] **Tuner Auto-Pilot**: Scripts to automate the full pipeline: datagen -> FindBestK -> SPSA -> apply.

---
*Last Updated: February 27, 2026*
