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
- [x] **Specialized Endgame Scaling**: Adjust scores for known drawish endings (e.g., Opposite Colored Bishops).
- [x] **Trapped Piece Detection**: Penalize pieces with zero mobility or those blocked by their own pawns.
- [x] **Connected Passed Pawns**: Bonuses for passed pawns that support each other's advancement.
- [x] **Bishop vs. Knight Scaling**: Dynamic adjustment of piece values based on pawn density.
- [x] **Space Evaluation**: Positional bonuses for controlling the central four ranks.
- [x] **King Ring Attacks**: Precise tactical model focusing on the immediate 8 squares around the king.

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
- [x] **Fast FEN Generation**: Buffer-based FEN formatting to boost datagen NPS.
- [x] **En Passant Refinement**: Robustify Zobrist hashing and evaluation (Passed Pawn awareness) for En Passant.

## 5. Advanced Tuning & Calibration
- [x] **Log Loss (Cross-Entropy)**: Add support for Log Loss in the tuner to improve NNUE/HCE calibration.
- [ ] **ELO-Based SPSA**: Direct optimization for match performance.
- [x] **Sigmoid Scaling (K-Factor) Analysis**: Tools to visualize how well the engine's scores match real-world probabilities.

## 6. Datagen & Dataset Pipeline
- [x] **Lock-free Output Buffering**: Minimize mutex contention when writing results from many threads.
- [x] **SIMD-Accelerated FEN/EPD**: Use SIMD for piece-to-char conversion and string formatting.
- [x] **Batch Search**: Optimize search specifically for bulk data generation at low depths.
- [x] Binary Format Support: Support for `.bin` formats to eliminate string parsing overhead.

## 7. Testing & Quality Assurance
- [x] **SPRT Framework**: Integrated Sequential Probability Ratio Testing for ELO validation.
- [ ] **Regression Suites**: Automated testing against previous releases.
- [x] **UCI Option Support**: Expose search knobs via UCI for easier tuning.

## 8. Advanced Search & Optimization
- [x] **Multi-Cut Pruning**: Prune nodes if multiple moves fail high at reduced depth.
- [ ] **Memory Prefetching**: Optimize TT lookups by prefetching memory before access.
- [ ] **AVX-512 NNUE**: Specialized kernels for ultra-high performance on modern hardware.
- [x] **Double Extensions**: Detect and extend search for extremely critical tactical lines.
- [x] **Internal Iterative Reductions (IIR)**: Reduce search depth when no TT move is available.
- [x] **History-Based LMR**: Scale move reductions based on historical success/failure of moves.
- [x] **Correction History**: Learn and apply systematic errors in static evaluation during search.
- [x] **Capture History**: Use historical capture success to improve move ordering in Quiescence search.

## 9. Development & Support Tooling
- [x] EPD Processing Suite: Specialized tools for shuffling, merging, and filtering blunders from datasets (epdproc).
- [ ] **Automated SPRT Controller**: Scripts to manage Cutechess-cli matches and report statistical Elo gains.
- [ ] **Search Visualization**: Tools to export and visualize search trees for debugging pruning behavior.

## 10. Quality of Life & Developer Experience
- [x] **Integrated Perft**: Built-in move generation speed and correctness tester.
- [ ] **Colored Board Display**: ANSI colors for the `display` command to make terminal testing more intuitive.
- [ ] **Logging System**: Dedicated logging to file for debugging long matches and unexpected crashes.
- [ ] **Tuner Auto-Pilot**: Scripts to automate the full pipeline: datagen -> FindBestK -> SPSA -> apply.

## 11. Future Extensions
- [ ] **Chess960 (FRC)**: Support for Fischer Random Chess movegen and castling rules (Planned for after standard chess maturity).
- [x] **UCI_ShowWDL**: Display Win/Draw/Loss probabilities in UCI output for better user feedback.

## 12. Advanced UCI Protocol & Options
- [ ] **Strength Limiting**: Implementation of `UCI_LimitStrength` and `UCI_Elo` for casual play.
- [ ] **Pondering**: Full support for "search while pondering" (thinking on opponent's time).
- [ ] **UCI_AnalyseMode**: Specialized search/eval adjustments when the engine is used for analysis.
- [ ] **Heuristic Knobs**: Expose all internal parameters (LMR, Pruning margins, etc.) via `setoption`.

## 13. Release & Versioning Strategy
- [ ] **Combined Versioning**: Engine versioning will be based on `Engine Version + Network Version` (e.g., Axon 1.0 + Net v1) to track the tight coupling between search and evaluation.
- [ ] **Automated Build Pipeline**: CI/CD to bundle specific engine releases with their corresponding validated NNUE files.

---
*Last Updated: February 27, 2026*
