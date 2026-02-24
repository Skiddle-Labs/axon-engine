# Axon Chess Engine (ACE)

Axon is a high-performance, tournament-grade chess engine written in Go (Golang). It utilizes a modern bitboard-based architecture and advanced search heuristics to deliver strong tactical and positional play. Axon communicates via the Universal Chess Interface (UCI) protocol, making it compatible with all standard chess GUIs.

## Key Features

### Board Representation
- **Bitboards**: High-speed board representation using 64-bit integers for all piece types and occupancy masks.
- **Zobrist Hashing**: Efficient, incremental position hashing for Transposition Table lookups and repetition detection.
- **Precomputed Attacks**: Fast lookup tables for leapers (Kings, Knights) and ray-casting for sliding pieces (Rooks, Bishops, Queens).

### Search Algorithms
- **Principal Variation Search (PVS)**: Optimized Alpha-Beta pruning that focuses on the most promising move.
- **Iterative Deepening**: Progressively deeper searches with real-time feedback.
- **Transposition Table (TT)**: Caches millions of search results to avoid redundant calculations across different branches.
- **Null Move Pruning (NMP)**: Drastically reduces search space by "passing" the turn to detect overwhelmingly strong positions.
- **Late Move Reductions (LMR)**: Safely reduces search depth for moves that are unlikely to be the best based on ordering.
- **Aspiration Windows**: Narrows the Alpha-Beta window around the previous score to speed up pruning.
- **Internal Iterative Deepening (IID)**: Performs shallow searches to find guide moves when no Transposition Table data is available.
- **Futility Pruning**: Skips quiet moves at low depths that cannot reasonably improve the score.
- **Quiescence Search**: Extends search during tactical exchanges to avoid the "horizon effect."

### Move Ordering
- **Static Exchange Evaluation (SEE)**: Calculates the outcome of capture sequences to avoid losing material blindly.
- **Killer Moves**: Remembers tactical "refutation" moves that caused cutoffs at specific depths.
- **History Heuristic**: Learns which moves have been historically strong throughout the search.
- **MVV-LVA**: Prioritizes captures based on the Most Valuable Victim and Least Valuable Aggressor.

### Evaluation (Traditional)
- **Tapered Evaluation**: Dynamically interpolates between **Midgame** and **Endgame** scores based on remaining material.
- **Mobility**: Rewards pieces with higher freedom of movement and activity.
- **King Safety**: Evaluates the pawn shield and the King's exposure to enemy pieces.
- **Pawn Structure**: Understands and penalizes doubled and isolated pawns.
- **Repetition Detection**: Detects and avoids (or seeks) three-fold repetitions.

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
Axon is a command-line engine. You can run it directly and type UCI commands, or connect it to a GUI like Arena or Cute Chess.

```bash
./axon
```

**Common Commands:**
- `uci`: Identify the engine and list options.
- `isready`: Check engine readiness.
- `position startpos moves e2e4 e7e5`: Setup a specific position.
- `go depth 12`: Search to a specific depth.
- `go wtime 300000 btime 300000`: Search with a time limit.
- `d`: Display the ASCII board and current state info.

## Project Structure
- `/internal/board`: Bitboards, Move Generation, SEE, and Zobrist Hashing.
- `/internal/search`: PVS logic, Transposition Table, and Move Ordering.
- `/internal/eval`: Tapered evaluation and positional heuristics.
- `/internal/uci`: UCI protocol communication layer.

## License
This project is licensed under the MIT License.