# Axon Chess Engine (ACE)

Axon is a high-performance, tournament-grade chess engine written in Go (Golang). It utilizes a modern bitboard-based architecture and advanced search heuristics to deliver strong tactical and positional play. Axon communicates via the Universal Chess Interface (UCI) protocol, making it compatible with all standard chess GUIs.

## Key Features

### Board Representation
- **Bitboards**: High-speed board representation using 64-bit integers for all piece types and occupancy masks.
- **Zobrist Hashing**: Efficient, incremental position hashing for Transposition Table lookups and repetition detection.
- **Precomputed Attacks**: Fast lookup tables for leapers (Kings, Knights) and ray-casting for sliding pieces (Rooks, Bishops, Queens).
- **SEE (Static Exchange Evaluation)**: Accurately calculates the material balance of capture sequences to inform move ordering and pruning.

### Search Algorithms
- **Principal Variation Search (PVS)**: Optimized Alpha-Beta pruning that focuses on the most promising move branch.
- **Iterative Deepening**: Progressively deeper searches with real-time UCI feedback and time management.
- **Quiescence Search**: Stabilizes evaluation by searching captures and checks at leaf nodes. Features **Delta Pruning** to skip low-impact captures and **Check Handling** to avoid tactical blind spots.
- **Transposition Table (TT)**: Caches millions of search results with aging and replacement strategies to avoid redundant calculations.
- **Pruning Heuristics**: 
    - **Null Move Pruning (NMP)**: Detects overwhelmingly strong positions to skip branches early.
    - **Late Move Reductions (LMR)**: Reduces search depth for moves deemed unlikely to improve the score.
    - **Futility Pruning**: Skips quiet moves at low depths when the static evaluation is significantly below alpha.
    - **Internal Iterative Deepening (IID)**: Finds guide moves when TT data is missing.

### Move Ordering
- **TT Move**: Prioritizes the best move found in previous search iterations.
- **MVV-LVA**: Orders captures by Most Valuable Victim and Least Valuable Aggressor.
- **Killer Moves & History Heuristic**: Rewards moves that caused cutoffs in other branches of the search tree.
- **Promotion Prioritization**: Gives high priority to moves that create high-value pieces.

### Evaluation (Traditional)
- **Tapered Evaluation**: Dynamically interpolates between **Midgame** and **Endgame** scores based on non-pawn material.
- **Threat Evaluation**: Detects and penalizes **Hanging Pieces** (attacked and undefended) and **Bad Trades** (pieces attacked by lesser-value enemy units).
- **Pawn Structure**: Comprehensive evaluation of **Passed Pawns** (rank-based bonuses), **Connected Pawns**, **Isolated Pawns**, and **Doubled Pawns**.
- **Mobility & Activity**: Rewards pieces for controlling more squares and occupying central positions via Piece-Square Tables (PST).
- **King Safety**: Evaluates pawn shields and proximity of enemy pieces to the king's position.
- **Special Bonuses**: Includes bonuses for the **Bishop Pair** and other coordination motifs.

## Getting Started

### Prerequisites
- [Go](https://golang.org/doc/install) (version 1.18 or higher)

### Installation
```bash
git clone https://github.com/personal-github/axon-engine.git
cd axon-engine
go build -o axon main.go
```

### Usage
Axon is a command-line engine. You can run it directly and type UCI commands, or connect it to a GUI like Arena, Cute Chess, or Banksia.

```bash
./axon
```

**Common Commands:**
- `uci`: Identify the engine and list options.
- `isready`: Check engine readiness.
- `position startpos moves e2e4 e7e5`: Setup a specific position.
- `go depth 12`: Search to a specific depth.
- `go wtime 300000 btime 300000`: Search with a time limit (e.g., 5 minutes).
- `d`: Display the ASCII board, current evaluation, and hash status.

## Project Structure
- `/internal/engine`: Bitboards, Move Generation, SEE, and Zobrist Hashing.
- `/internal/search`: PVS logic, Quiescence Search, TT, and Move Ordering.
- `/internal/eval`: Tapered evaluation, Pawn Structure, and Positional Heuristics.
- `/internal/protocol`: UCI protocol communication layer.

## License
This project is licensed under the MIT License.