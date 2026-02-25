# Axon Tuning Guide: Datagen & Texel Tuner

This guide explains how to use the Axon data generation and tuning pipeline to optimize the engine's evaluation parameters. Axon uses the **Texel Tuning Method**, which minimizes the error between the engine's static evaluation and the actual game results.

---

## The Tuning Pipeline

The process consists of three main steps:
1. **Data Generation**: Self-play games to create a dataset of FEN positions and game results.
2. **Tuning**: Running the Texel Tuner on the dataset to find optimal weights.
3. **Integration**: Updating the evaluation constants in the source code.

---

## 1. Data Generation (`cmd/datagen`)

The `datagen` tool generates high-quality training data by playing games against itself using short time controls and slight randomness (to ensure variety).

### Build the Tool
```bash
go build -o datagen.exe ./cmd/datagen
```

### Parameters
- `-out`: Path to the output EPD file (Default: `my_training_data.epd`).
- `-games`: Number of games to play.
- `-threads`: Number of concurrent games (Default: Number of CPUs).
- `-depth`: Search depth for each move (Default: 8).
- `-book`: (Optional) Path to a Polyglot `.bin` book to vary openings.
- `-min-ply`: Start recording positions after this many plies (Default: 16).

### Usage Example
To generate 10,000 games using 8 threads and an opening book:
```bash
./datagen.exe -games 10000 -threads 8 -depth 6 -book opening_book.bin -out data_10k.epd
```

### Best Practices for Datagen
- **Use a Book**: Always use an opening book. Without it, the engine will play the same few lines repeatedly, leading to "overfitted" data.
- **Recording Phase**: Data is only recorded after the engine starts searching (post-book).
- **Quality vs Quantity**: Depth 6-8 is usually sufficient. Higher depth provides better quality but takes significantly longer.

---

## 2. Evaluation Tuning (`cmd/tuner`)

The `tuner` reads the generated EPD file and uses a local search algorithm (like Gradient Descent or SPSA) to find the evaluation constants that best predict the game outcomes.

### Build the Tool
```bash
go build -o tuner.exe ./cmd/tuner
```

### How it Works (Texel Method)
The tuner calculates a "Sigmoid" value of the engine's static evaluation for every position in your dataset. It then compares this to the actual game result (1.0 for Win, 0.5 for Draw, 0.0 for Loss) and adjusts the parameters to minimize the **Mean Squared Error (MSE)**.

### Usage Example
```bash
./tuner.exe -file data_10k.epd -iterations 100
```

### Tuning Output
The tuner will output a set of optimized values. Look for a format similar to:
```text
Optimized PST Values:
[0, 5, 10, ...]
Optimized Mobility Values:
...
```

---

## 3. Integration

Once the tuner finishes, you must manually apply the new values to your engine:

1. Open `internal/eval/eval.go`.
2. Locate the constants or arrays (like `PST` or `PieceValues`).
3. Replace the old values with the "Optimized" values from the tuner output.
4. Re-run `bench` or play matches to verify the Elo gain.

---

## Tips for Success

- **Dataset Size**: For meaningful results, aim for at least **500,000 to 1,000,000 positions**. 10,000 games typically yield ~300,000 usable positions.
- **Variety**: Ensure your dataset contains various phases of the game (middlegame, endgame).
- **Iteration**: Tuning is an iterative process. Once you update the constants, you can run `datagen` again with the "new" stronger engine to generate even better data for the next round of tuning.
- **MSE Check**: If the Mean Squared Error is increasing, your dataset might be too noisy or your learning rate (if applicable) is too high.

---

## Dataset Format
The data is stored in EPD format with the game result attached:
`fen [result]`
Example:
`rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - [0.5]`

---
*Happy Tuning!*