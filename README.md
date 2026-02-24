# Axon Chess Engine (ACE)

Axon is a high-performance, tournament-grade chess engine written in Go (Golang). It utilizes a modern bitboard-based architecture and advanced search heuristics to deliver strong tactical and positional play. Axon communicates via the Universal Chess Interface (UCI) protocol, making it compatible with all standard chess GUIs like Cute Chess, Arena, and Banksia.

## Key Features

### Board Representation
- **Bitboards**: High-speed board representation using 64-bit integers for all piece types and occupancy masks.
- **Zobrist Hashing**: Efficient, incremental position hashing for Transposition Table lookups, repetition detection, and 50-move rule tracking.
- **Precomputed Attacks**: Fast lookup tables for leapers (Kings, Knights) and ray-casting (Magic Bitboards) for sliding pieces.
- **SEE (Static Exchange Evaluation)**: Accurately calculates the material balance of capture sequences to inform move ordering and pruning.
- **Move History**: Full irreversible state tracking (En Passant, Castling, Half-move clock) for accurate game navigation.

### Search Algorithms
- **Parallel Search (Lazy SMP)**: Scalable search performance using multiple CPU cores. Helper threads populate a shared, sharded Transposition Table with depth-offset explorations to maximize tree coverage.
- **Principal Variation Search (PVS)**: Optimized Alpha-Beta pruning that focuses on the most promising move branch.
- **Iterative Deepening**: Progressively deeper searches with real-time UCI feedback.
- **Advanced Pruning & Reductions**:
    - **Static Null Move Pruning (RFP)**: Prunes nodes where static evaluation is significantly above beta at shallow depths.
    - **Null Move Pruning (NMP)**: Detects overwhelmingly strong positions to skip branches early.
    - **Late Move Reductions (LMR)**: Reduces search depth for moves deemed unlikely to improve the score based on history and count.
    - **Late Move Pruning (LMP)**: Prunes quiet moves at high move counts in late search nodes.
    - **Futility Pruning**: Skips quiet moves at low depths when the static evaluation is significantly below alpha.
- **Extensions**:
    - **Singular Extensions**: Extends the search for "forced" moves that are significantly better than any other option.
    - **Check Extensions**: Automatically extends depth when the king is in check to avoid tactical blind spots.
- **Dynamic Time Management**: 
    - Implements **Soft and Hard limits** for optimal time allocation.
    - **Move Stability**: Adjusts search time based on how often the best move changes across depths.
    - **Single Move Optimization**: Plays instantly if only one legal move exists (when not pondering).

### Move Ordering
- **TT Move**: Prioritizes the best move found in previous search iterations or deeper branches.
- **MVV-LVA**: Orders captures by Most Valuable Victim and Least Valuable Aggressor.
- **Killer Moves**: Prioritizes quiet moves that caused cutoffs at the same ply in other branches.
- **Countermove Heuristic**: Tracks and prioritizes successful responses to specific opponent moves.
- **History Heuristic**: Rewards moves that have historically caused beta cutoffs throughout the search.

### Evaluation (Traditional Tapered HCE)
- **Tapered Evaluation**: Dynamically interpolates between **Midgame** and **Endgame** scores based on non-pawn material.
- **Threat Evaluation**: Penalizes hanging pieces and bad trades (pieces attacked by lesser-value units).
- **Pawn Structure**: Evaluates Passed Pawns (rank-based), Connected Pawns, Isolated Pawns, and Doubled Pawns.
- **King Safety**: Evaluates pawn shields and the proximity of enemy pieces.

## Getting Started

### Installation
```bash
git clone https://github.com/personal-github/axon-engine.git
cd axon-engine
go build -o axon main.go
```

### Usage
Axon is a command-line engine. For the best experience, connect it to a UCI-compatible GUI.

```bash
./axon
```

**Common UCI Commands:**
- `uci`: Identify the engine and list available options.
- `isready`: Check engine readiness.
- `bench`: Runs a standardized performance test across multiple positions and reports NPS.
- `eval`: Displays the static evaluation of the current position.
- `position startpos moves e2e4`: Setup the board.
- `go ponder wtime 300000 btime 300000`: Start pondering (searching on opponent's time).
- `ponderhit`: Transition from pondering to active search after an opponent makes the expected move.

### UCI Options (`setoption name <X> value <Y>`)
- **Hash**: Transposition table size in MB (Default: 64, Min: 1, Max: 65536).
- **Threads**: Number of search threads (Default: 1, Min: 1, Max: 128).
- **MultiPV**: Number of principal variations to show (Default: 1, Min: 1, Max: 128).
- **Ponder**: Enable/Disable pondering support (Default: false).
- **Move Overhead**: Time buffer in ms to account for network/GUI lag (Default: 10, Max: 5000).
- **Slow Mover**: Percentage multiplier for time management (Default: 100, Range: 10-1000).
- **Clear Hash**: Button to manually wipe the Transposition Table.
- **UCI_AnalyseMode**: Optimizes search for GUI analysis.

## Project Structure
- `/internal/engine`: Bitboards, Move Generation, SEE, and Zobrist Hashing.
- `/internal/search`: PVS logic, Quiescence Search, TT, and Pruning Heuristics.
- `/internal/eval`: Tapered evaluation, Pawn Structure, and Positional Heuristics.
- `/internal/protocol`: UCI protocol communication and Time Management.

## License
This project is licensed under the MIT License.