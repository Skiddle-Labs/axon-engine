# Axon Chess Engine (ACE)

Axon is a high-performance, tournament-grade chess engine written in Go (Golang). It utilizes a modern bitboard-based architecture and advanced search heuristics to deliver strong tactical and positional play. Axon communicates via the Universal Chess Interface (UCI) protocol, making it compatible with all standard chess GUIs like Cute Chess, Arena, and Banksia.

## Key Features

### Board Representation
- **Bitboards**: High-speed board representation using 64-bit integers for all piece types and occupancy masks.
- **Zobrist Hashing**: Efficient, incremental position hashing for Transposition Table lookups, repetition detection, and Polyglot-compatible book probing.
- **Precomputed Attacks**: Fast lookup tables for leapers (Kings, Knights) and Ray-casting (Magic Bitboards) for sliding pieces.
- **SEE (Static Exchange Evaluation)**: Accurately calculates the material balance of capture sequences to prune losing captures.
- **Pawn Hash Table**: High-speed specialized cache for pawn structure evaluations, significantly boosting search speed (NPS).
- **NNUE Accumulators**: Incremental first-layer hidden layer updates (HalfKP features) integrated into the move-handling core. Optimized with **AVX2 SIMD** kernels for maximum throughput.

### Search Algorithms
- **Modular Search Architecture**: Decoupled search logic (Negamax, Ordering, LMR, TT) for high maintainability and performance.
- **Parallel Search (Lazy SMP)**: Scalable search performance using multiple CPU cores with a high-speed lockless Transposition Table.
- **Principal Variation Search (PVS)**: Optimized Alpha-Beta pruning focusing on the most promising moves.
- **Advanced Pruning & Reductions**:
    - **Adaptive Null Move Pruning (NMP)**: Dynamic reduction (`3 + depth/6`) for aggressive pruning at high depths.
    - **Static Null Move Pruning (RFP)**: Prunes nodes where the static evaluation is significantly above beta.
    - **Refined LMR**: Precomputed logarithmic reduction table scaling with both depth and move index.
    - **Futility Pruning**: Skips quiet moves at low depths when the position is unlikely to improve alpha.
- **Extensions**:
    - **Singular Extensions**: Extends the search for "forced" moves that are significantly better than alternatives.
    - **Check Extensions**: Automatically extends depth when the king is in check.
    - **Passed Pawn Extensions**: Automatically extends the search when a passed pawn reaches the 6th or 7th rank.

### Evaluation (Hybrid HCE + NNUE)
- **NNUE**: High-performance neural network evaluation (768 -> 256 -> 1 architecture) with SIMD-accelerated inference.
- **Tapered HCE**: Dynamically interpolates between Midgame and Endgame scores using hand-crafted features.
- **Refined King Safety**: Uses an attacking zone model and non-linear safety tables with piece-specific weighting.
- **Advanced Positional Evaluation**: Logic for non-linear mobility, outposts, and virtual mobility pressure.
- **SPSA & Texel Tuning**: Support for both Local Search and SPSA (Simultaneous Perturbation Stochastic Approximation) for parameter optimization.

## Getting Started

### Installation
```bash
git clone https://github.com/Skiddle-Labs/axon-engine.git
cd axon-engine

# On Windows:
go build -o axon.exe .

# On Linux or macOS:
go build -o axon .
```

### Usage
Axon is a command-line engine. Connect it to a UCI-compatible GUI for the best experience.

```bash
# Windows
./axon.exe

# Linux or macOS
./axon
```

**Common UCI Commands:**
- `uci`: Identify the engine and list available options.
- `bench`: Runs a standardized performance test and reports NPS and node counts.
- `eval`: Displays a detailed breakdown of the static evaluation.
- `count <file.epd>`: Counts the number of positions in an EPD file.

## Development Tools

### Datagen (`cmd/datagen`)
Generates high-quality EPD training data through multi-threaded self-play. Optimized for high volume (millions of positions) for NNUE training.

### Tuner (`cmd/tuner`)
A multi-threaded optimizer supporting both **Texel (Local Search)** and **SPSA** methods. Now supports non-linear mobility tables and automated feature precomputation.

### Applier (`cmd/apply`)
Automated tool to inject tuned parameters directly into the Go source code (`internal/eval/params.go`).

For detailed instructions, see the [Tuning Guide](TUNING_GUIDE.md) and [Training Guide](TRAINING_GUIDE.md).

## License
This project is licensed under the MIT License.
