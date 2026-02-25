# Axon Chess Engine (ACE)

Axon is a high-performance, tournament-grade chess engine written in Go (Golang). It utilizes a modern bitboard-based architecture and advanced search heuristics to deliver strong tactical and positional play. Axon communicates via the Universal Chess Interface (UCI) protocol, making it compatible with all standard chess GUIs like Cute Chess, Arena, and Banksia.

## Key Features

### Board Representation
- **Bitboards**: High-speed board representation using 64-bit integers for all piece types and occupancy masks.
- **Zobrist Hashing**: Efficient, incremental position hashing for Transposition Table lookups, repetition detection, and Polyglot-compatible book probing.
- **Precomputed Attacks**: Fast lookup tables for leapers (Kings, Knights) and Ray-casting (Magic Bitboards) for sliding pieces.
- **SEE (Static Exchange Evaluation)**: Accurately calculates the material balance of capture sequences.

### Search Algorithms
- **Parallel Search (Lazy SMP)**: Scalable search performance using multiple CPU cores with a lockless Transposition Table.
- **Principal Variation Search (PVS)**: Optimized Alpha-Beta pruning focusing on the most promising moves.
- **Advanced Pruning & Reductions**:
    - **Static Null Move Pruning (RFP)**: Prunes nodes where the static evaluation is significantly above beta.
    - **Null Move Pruning (NMP)**: Detects overwhelmingly strong positions to skip branches.
    - **Late Move Reductions (LMR)** and **Late Move Pruning (LMP)**: Reduces or skips moves deemed unlikely to improve the score.
- **Extensions**:
    - **Singular Extensions**: Extends the search for "forced" moves that are significantly better than alternatives.
    - **Check Extensions**: Automatically extends depth when the king is in check.
### Move Ordering
- **TT Move**: Prioritizes the best move found in previous search iterations.
- **MVV-LVA**: Orders captures by Most Valuable Victim and Least Valuable Aggressor.
- **Killer Moves & Countermove Heuristic**: Prioritizes quiet moves that have proven effective in similar branches.
- **History Heuristic**: Rewards moves that have historically caused beta cutoffs.

### Evaluation (Hybrid Tapered HCE)
- **Tapered Evaluation**: Dynamically interpolates between Midgame and Endgame scores.
- **Refined King Safety**: Uses an attacking zone model and non-linear safety tables to detect threats.
- **Advanced Pawn Evaluation**: Includes logic for Phalanx pawns, Backward pawns, and supported structures.
- **Texel Tuning**: The evaluation is prepared for automated tuning using the included `tuner` tool.

## Getting Started

### Installation
```bash
git clone https://github.com/Skiddle-Labs/axon-engine.git
cd axon-engine
go build -o axon main.go
```

### Usage
Axon is a command-line engine. Connect it to a UCI-compatible GUI for the best experience.

```bash
./axon
```

**Common UCI Commands:**
- `uci`: Identify the engine and list available options.
- `bench`: Runs a standardized performance test and reports NPS.
- `eval`: Displays a detailed breakdown of the static evaluation.
- `position startpos moves e2e4`: Setup the board.

## Configuration (UCI Options)

Axon can be configured using the `setoption name <Name> value <Value>` command.

- **Hash**: Transposition table size in MB (Default: 64, Max: 65536).
- **Threads**: Number of search threads (Default: 1, Max: 128).
- **Book File**: Path to a **Polyglot (.bin)** opening book.
    - `setoption name Book File value C:\books\pro.bin`
- **Book Best Move**: If true, the engine picks the move with highest weight from the book (Default: false).
- **Book Depth**: Maximum ply count for book usage (Default: 255).
- **Move Overhead**: Time buffer in ms to account for network/GUI lag (Default: 10).
- **Slow Mover**: Percentage multiplier for time management (Default: 100).
- **Clear Hash**: Manually wipe the Transposition Table.

## Project Structure
- `/internal/engine`: Bitboards, Move Generation, Book Probing, and Zobrist Hashing.
- `/internal/search`: PVS logic and Pruning Heuristics.
- `/internal/eval`: Tapered evaluation, PSTs, and Positional Heuristics.
- `/cmd/datagen`: Self-play tool for generating tuning data with opening book support.
- `/cmd/tuner`: Automated evaluation tuning using the Texel Method.

## License
This project is licensed under the MIT License.