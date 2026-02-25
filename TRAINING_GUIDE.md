# Axon NNUE Training Guide

This guide describes how to train a neural network for the Axon chess engine using the generated self-play data.

## 1. Data Collection
Use the `datagen` tool to collect at least 1,000,000 positions. For a high-strength network, 10M+ positions are recommended.

### Build and Run (Linux/MacOS)
```bash
# Build the datagen tool
go build -o datagen ./cmd/datagen

# Run datagen
./datagen -games 100000 -threads 8 -depth 6 -out training_data.epd
```

### Run (Windows)
```bash
./datagen.exe -games 100000 -threads 8 -depth 6 -out training_data.epd
```

## 2. Recommended Trainer: Bullet
[Bullet](https://github.com/official-stockfish/bullet) is a fast, GPU-accelerated trainer written in Rust. It is the standard for modern chess engine development.

### Installation
1. Install [Rust](https://rustup.rs/).
2. Clone and build Bullet:
   ```bash
   git clone https://github.com/official-stockfish/bullet
   cd bullet
   cargo build --release
   ```

## 3. Preparing the Data
Bullet can read EPD files directly, but for faster training, it is recommended to convert them to a binary format.

### Shuffling
Shuffle your EPD file to ensure the network doesn't see highly correlated positions (e.g., from the same game) in sequence.
```bash
# Example using a simple shuffler or 'shuf' on Linux
shuf training_data.epd -o training_data_shuffled.epd
```

## 4. Training Configuration
Create a `axon.toml` file for Bullet. Axon uses a simple `768 -> 256 -> 1` architecture.

### Network Architecture
- **Inputs**: 768 (12 pieces $\times$ 64 squares).
- **L1 (Hidden)**: 256 neurons.
- **Activation**: SCReLU (clamped square).
- **Output**: Single value, quantized.

### Example `axon.toml`
```toml
[dataset]
path = "path/to/training_data_shuffled.epd"

[input]
type = "Simple"
width = 768

[network]
layers = [
    { type = "Linear", inputs = 768, outputs = 256 },
    { type = "SCReLU" },
    { type = "Linear", inputs = 512, outputs = 1 } # 512 = 256 * 2 (White + Black perspectives)
]

[training]
batch_size = 16384
epochs = 100
lr = 0.001
loss = "Sigmoid"
```
*(Note: Adjust paths and parameters based on your hardware and dataset size.)*

## 5. Exporting to Axon
Once training is complete, Bullet will output a weight file (often `.nnue` or `.bin`).

### Quantization Scaling
Axon expects weights to be quantized as follows:
- **QA**: 255 (Activation scaling)
- **QB**: 64 (Final score scaling)

The `internal/nnue/nnue.go` loader expects a flat binary file with:
1. `FeatureWeights`: `[768][256] int16`
2. `FeatureBiases`: `[256] int16`
3. `OutputWeights`: `[512] int16`
4. `OutputBias`: `int32`

## 6. Loading the Network in Axon
Place your `.nnue` file in the engine directory and enable it via UCI:

```bash
setoption name EvalFile value axon_v1.nnue
setoption name Use NNUE value true
```

Run `bench` to verify the NPS and ensure the network is loading correctly.
