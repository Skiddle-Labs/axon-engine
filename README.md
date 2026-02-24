# Axon Chess Engine (ACE)

Axon is a high-performance chess engine written in Go (Golang). It implements a modern bitboard-based architecture and communicates via the Universal Chess Interface (UCI) protocol, making it compatible with various chess GUIs like Arena, Cute Chess, and Fritz.

## Features

### Board Representation
- **Bitboards**: Uses 64-bit unsigned integers to represent piece positions, enabling extremely fast move generation and position analysis.
- **Zobrist Hashing**: Implements incremental position hashing for efficient Transposition Table lookups and repetition detection.
- **Magic-ready Attacks**: Includes precomputed attack tables for leaper pieces (Kings, Knights) and ray-casting for sliding pieces (Rooks, Bishops, Queens).

### Search & Strategy
- **Negamax Search**: The core search algorithm using Alpha-Beta pruning.
- **Iterative Deepening**: Progressively searches deeper to provide the best move within time constraints.
- **Quiescence Search**: Extends the search in volatile positions (captures) to avoid the "horizon effect."
- **Move Ordering**: Optimizes Alpha-Beta pruning using MVV-LVA (Most Valuable Victim - Least Valuable Aggressor) to explore tactical lines first.
- **Transposition Table (TT)**: Caches search results to avoid redundant calculations across different branches of the move tree.

### Evaluation
- **Material Weighting**: Standard piece values.
- **Piece-Square Tables (PST)**: Positional heuristics that encourage central development and king safety.

## Getting Started

### Prerequisites
- [Go](https://golang.org/doc/install) (version 1.18 or higher recommended)

### Installation
Clone the repository and build the binary:
```bash
git clone https://github.com/personal-github/axon-engine.git
cd axon-engine
go build -o axon main.go
```

### Usage
Run the engine directly from the terminal to interact via UCI commands:
```bash
./axon
```

Common UCI commands:
- `uci`: Identify the engine.
- `isready`: Check if the engine is ready.
- `position startpos moves e2e4`: Set up the board.
- `go depth 6`: Start searching for the best move.
- `d`: (Custom) Display the ASCII representation of the current board.

## Project Structure
- `/internal/board`: Core logic for bitboards, move generation, and board state.
- `/internal/search`: Minimax search, Alpha-Beta pruning, and Transposition Table.
- `/internal/eval`: Position evaluation and heuristics.
- `/internal/uci`: UCI protocol communication handler.

## License
This project is licensed under the MIT License.