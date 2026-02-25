# Axon Chess Engine (ACE)

Axon is a high-performance, tournament-grade chess engine written in Go (Golang). It utilizes a modern bitboard-based architecture and advanced search heuristics to deliver strong tactical and positional play. Axon communicates via the Universal Chess Interface (UCI) protocol, making it compatible with all standard chess GUIs like Cute Chess, Arena, and Banksia.

## Key Features

### Board Representation
- **Bitboards**: High-speed board representation using 64-bit integers for all piece types and occupancy masks.
- **Zobrist Hashing**: Efficient, incremental position hashing for Transposition Table lookups, repetition detection, and Polyglot-compatible book probing.
- **Precomputed Attacks**: Fast lookup tables for leapers (Kings, Knights) and Ray-casting (Magic Bitboards) for sliding pieces.
- **SEE (Static Exchange Evaluation)**: Accurately calculates the material balance of capture sequences to prune losing captures.
- **Pawn Hash Table**: High-speed specialized cache for pawn structure evaluations, significantly boosting search speed (NPS).
- **NNUE Accumulators**: Incremental first-layer hidden layer updates (HalfKP features) integrated into the move-handling core for future neural network inference.

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
- **Other Search Heuristics**:
    - **ProbCut**: Aggressive pruning at high depths by searching with a narrow window and reduced depth.
    - **Internal Iterative Deepening (IID)**: Performs a shallow search to find a candidate move when no Transposition Table move is available.

### Move Ordering
- **TT Move**: Prioritizes the best move found in previous search iterations.
- **MVV-LVA**: Orders captures by Most Valuable Victim and Least Valuable Aggressor.
- **Killer Moves & Countermove Heuristic**: Prioritizes quiet moves that have proven effective in similar branches.
- **History Heuristic & Penalty**: Rewards moves that cause beta cutoffs and penalizes quiet moves that fail to improve alpha (Negative History).

### Evaluation (Hybrid HCE + NNUE)
- **Tapered HCE**: Dynamically interpolates between Midgame and Endgame scores using hand-crafted features.
- **NNUE (In Progress)**: Currently supports incremental feature updates and high-performance data generation for neural network training.
- **Refined King Safety**: Uses an attacking zone model and non-linear safety tables with piece-specific weighting.
- **Advanced Pawn Evaluation**: Sophisticated logic for passed pawns, connected structures, and shields.
- **SPSA & Texel Tuning**: Support for both Local Search and SPSA (Simultaneous Perturbation Stochastic Approximation) for parameter optimization.
- **Endgame Scaling**: Specialized logic for detecting insufficient material and scaling evaluations in drawish endgames.

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
- `position startpos moves e2e4`: Setup the board and play a move.

## Configuration (UCI Options)

Axon can be configured using the `setoption name <Name> value <Value>` command.

- **Hash**: Transposition table size in MB (Default: 64, Max: 65536).
- **Threads**: Number of search threads (Default: 1, Max: 128).
- **MultiPV**: Number of best move lines to analyze simultaneously (Default: 1). Optimized implementation using root-level TT exclusion for accurate analysis.
- **Book File**: Path to a **Polyglot (.bin)** opening book.
- **Book Best Move**: If true, the engine picks the move with the highest weight from the book.
- **Move Overhead**: Time buffer in ms to account for network/GUI lag (Default: 10).
- **Slow Mover**: Percentage multiplier for time management (Default: 100).
- **Clear Hash**: Manually wipe the Transposition Table.

## Development Tools

Axon includes a robust toolchain for engine development and automated tuning.

### Datagen (`cmd/datagen`)
Generates high-quality EPD training data through multi-threaded self-play with opening book support.

### Tuner (`cmd/tuner`)
A multi-threaded optimizer supporting both **Texel (Local Search)** and **SPSA** methods. It minimizes Mean Squared Error (MSE) between evaluation and game results.

### Applier (`cmd/apply`)
An automated tool to parse tuner results (`.txt`) and inject the optimized parameters directly into the Go source code (`internal/eval/params.go`). Now supports piece-specific weights and table-based parameters with cross-package type awareness.

For detailed instructions, see the [Tuning Guide](TUNING_GUIDE.md).

## Project Structure
- `/internal/engine`: Core logic (Bitboards, MoveGen, Zobrist, Book Probing).
- `/internal/search`: Modular search components (Negamax, Ordering, LMR, TT).
- `/internal/eval`: Tapered evaluation, PSTs, and positional heuristics.
- `/cmd`: Tuning, data generation, and parameter application utilities.

## License
This project is licensed under the MIT License.