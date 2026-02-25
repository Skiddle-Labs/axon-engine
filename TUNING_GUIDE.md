# Axon Tuning Guide: Datagen & Texel Tuner

This guide explains how to use the Axon data generation and tuning pipeline to optimize the engine's evaluation parameters. Axon uses the **Texel Tuning Method**, which minimizes the error between the engine's static evaluation and the actual game results.

---

## The Tuning Pipeline

The process consists of three main steps:
1. **Data Generation**: Self-play games to create a dataset of FEN positions and game results.
2. **Tuning**: Running the high-performance tuner (using SPSA or Local Search) to find optimal weights.
3. **Integration**: Automatically applying the new values to the engine's source code.

---

## 1. Data Generation (`cmd/datagen`)

The `datagen` tool generates training data by playing games against itself.

### Usage
```bash
go build -o datagen.exe ./cmd/datagen
./datagen.exe -games 10000 -threads 8 -depth 8 -book opening_book.bin -out data.epd
```

### Parameters
- `-games`: Number of games to play.
- `-depth`: Search depth for each move (Recommended: 6-10).
- `-book`: Path to a Polyglot `.bin` book (Critical for variety).
- `-min-ply`: Start recording after this many plies (Default: 16).

---

## 2. Evaluation Tuning (`cmd/tuner`)

The Axon tuner is a high-performance, multi-threaded tool that optimizes over 1,700 parameters simultaneously.

### High-Performance Features
- **Feature Precomputation**: At startup, the tuner extracts all chess features (mobility, outposts, pawn structure) into a compact format. This allows the optimization loop to run thousands of times faster by avoiding redundant bitboard calculations.
- **Multi-threading**: The Mean Squared Error (MSE) calculation is parallelized across all available CPU cores.

### Build and Run
```bash
go build -o tuner.exe ./cmd/tuner
./tuner.exe -file data.epd -method spsa -iterations 5000 -save tuned_params.txt
```

### Optimization Methods
- **SPSA (`-method spsa`)**: *Recommended.* Uses Simultaneous Perturbation Stochastic Approximation. It adjusts all parameters at once in every iteration. This is the only practical way to tune the 1,500+ PST values effectively.
- **Local Search (`-method local`)**: Adjusts parameters one-by-one. Best for fine-tuning a small number of scalar values.

### Tuning Parameters
- `-iterations`: For SPSA, 5,000–50,000 is recommended for a large dataset.
- `-threads`: Defaults to 80% of logical cores.
- `-save`: The filename for the output results.

---

## 3. Automated Integration (`cmd/apply`)

Axon provides a utility to automatically inject tuned parameters back into the engine. The evaluation parameters are stored in `internal/eval/params.go` to keep them isolated from the core logic.

### How to Apply Results
Once your tuner finishes and produces `tuned_params.txt`, run the applier utility:

```bash
go run cmd/apply/main.go tuned_params.txt
```

This script will:
1. Parse the 1,700+ values from your results file.
2. Automatically update the arrays (`MgPST`, `EgPST`, `SafetyTable`) and scalars (`PawnMG`, `RookMobilityEG`, etc.) in `internal/eval/params.go`.
3. Preserve the structure and comments of the source file.

### Rebuild the Engine
After applying the parameters, recompile Axon to use the new weights:
```bash
go build -o axon.exe .
```

---

## Tips for Success

- **Dataset Size**: Aim for at least **500,000 positions** for a full PST tune.
- **SPSA Iterations**: SPSA is stochastic. If parameters aren't moving enough, increase the `-iterations`.
- **Validation**: After applying parameters, run the `bench` command to verify the engine still performs as expected: `./axon.exe bench`.
- **Iterative Improvement**: Tuning is a cycle. Use your new, stronger engine to generate a "higher quality" dataset for the next round of tuning.

---
*Happy Tuning!*