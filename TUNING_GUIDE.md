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
# Windows
go build -o datagen.exe ./cmd/datagen
./datagen.exe -games 100000 -threads 8 -depth 6 -out training_data.epd
```

### Tips for Datagen
- **Volume**: For NNUE training, aim for 1M+ positions. For HCE tuning, 100k-500k is usually sufficient.
- **Monitoring**: Use the `count` command in the main engine to check progress: `./axon.exe count training_data.epd`.
- **Variety**: Use a depth of 6-8 and multiple threads. Lower depths generate more positions per second, which is often better for overall data variety.

---

## 2. Evaluation Tuning (`cmd/tuner`)

The Axon tuner is a high-performance, multi-threaded tool that optimizes hundreds of parameters simultaneously, including Piece-Square Tables (PST) and non-linear mobility curves.

### High-Performance Features
- **Feature Precomputation**: Extracts mobility, outposts, and king safety features into a compact format before optimization.
- **Non-Linear Mobility**: Automatically tunes every bin of the mobility tables for Knights, Bishops, Rooks, and Queens.
- **SPSA Support**: Handles high-dimensional parameter spaces (like PSTs) much better than standard local search.

### Build and Run
```bash
# Windows
go build -o tuner.exe ./cmd/tuner
./tuner.exe -file training_data.epd -method spsa -iterations 5000 -save tuned_params.txt
```

### Optimization Methods
- **SPSA (`-method spsa`)**: *Recommended.* Adjusts all parameters simultaneously. Use this for full PST and mobility tuning.
- **Local Search (`-method local`)**: Adjusts parameters one-by-one. Best for small-scale calibration.

---

## 3. Automated Integration (`cmd/apply`)

Axon provides a utility to automatically inject tuned parameters back into the engine's source code (`internal/eval/params.go`).

### How to Apply Results
```bash
go run cmd/apply/main.go tuned_params.txt
```

This script handles:
1. **Scalars**: Material values, bonuses, and penalties.
2. **Arrays**: Piece-Square Tables, Safety tables, and non-linear Mobility tables.
3. **Type Mapping**: Automatically maps internal piece types (e.g., `Pawn`) to the correct Go syntax (`types.Pawn`).

### Rebuild the Engine
```bash
go build -o axon.exe .
```

---

## Tips for Success

- **Dataset Size**: More is better. 500k positions is a good baseline for HCE tuning.
- **Validation**: Always run `bench` after applying parameters to ensure no performance regression.
- **Iterative Loop**: Use the tuned engine to generate even higher-quality data for the next tuning run.
