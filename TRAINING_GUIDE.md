# Axon NNUE Training Guide

This guide describes how to train a neural network for the Axon chess engine using the generated self-play data and the **nnue-pytorch** trainer.

## 1. Data Collection
Use the `datagen` tool to collect at least 1,000,000 positions. For a high-strength network, 10M+ positions are recommended.

### Build and Run (Linux/MacOS)
```bash
# Build the datagen tool
go build -o datagen ./cmd/datagen

# Run datagen (EPD format)
./datagen -games 100000 -threads 8 -depth 6 -out training_data.epd

# Run datagen (High-performance Binary format)
./datagen -games 100000 -threads 8 -depth 6 -bin -out training_data.bin
```

### Run (Windows)
```bash
./datagen.exe -games 100000 -threads 8 -depth 6 -bin -out training_data.bin
```

## 2. Recommended Trainer: nnue-pytorch
[nnue-pytorch](https://github.com/official-stockfish/nnue-pytorch) is the official Stockfish NNUE trainer and the current standard for high-performance chess engine training.

### Installation
1.  **Install Python 3.9+**: Ensure you have a modern Python environment.
2.  **Clone the Repository**:
    ```bash
    git clone https://github.com/official-stockfish/nnue-pytorch
    cd nnue-pytorch
    ```
3.  **Install Dependencies**:
    ```bash
    pip install -r requirements.txt
    ```

## 3. Preparing the Data
Axon provides the `epdproc` tool for dataset management. It converts formats and shuffles data, which is critical for preventing the model from over-learning specific game sequences.

### Build the tool
```bash
go build -o epdproc ./cmd/epdproc
```

### Conversion and Shuffling
**Convert EPD to Shuffled Binary (Recommended):**
```bash
./epdproc -in training_data.epd -out shuffled.bin -if epd -of bin -shuffle
```

**Shuffle existing Binary data:**
```bash
./epdproc -in training_data.bin -out shuffled.bin -if bin -of bin -shuffle
```
*Note: The binary output format is a standard 32-byte packed format compatible with nnue-pytorch.*

## 4. Training Configuration
Axon utilizes a standard `768 -> 256 -> 1` architecture. This consists of a perspective-aware input layer (HalfKP), a 256-neuron hidden layer with SCReLU activation, and a single output neuron.

### Network Architecture
-   **Inputs**: 768 (12 piece types $\times$ 64 squares).
-   **L1 (Hidden)**: 256 neurons.
-   **Activation**: SCReLU (clamped square).
-   **Output**: Single value, quantized for centipawn evaluation.

### Running the Trainer
Execute the training script with parameters tuned for Axon's architecture:

```bash
python train.py \
    --data "path/to/shuffled.bin" \
    --architecture "HalfKP(768)->256->1" \
    --batch-size 16384 \
    --epochs 100 \
    --lr 0.001 \
    --gamma 0.99 \
    --wd 0.0 \
    --num-workers 4
```

*Adjust `--num-workers` and `--batch-size` according to your GPU's VRAM and CPU core count.*

## 5. Exporting to Axon
After training, `nnue-pytorch` produces checkpoint files (`.ckpt`). You must convert these to the raw binary format Axon expects using the provided serialization scripts in the `nnue-pytorch` repository.

### Quantization Scaling
Axon's engine implementation expects weights to be quantized using the following constants:
-   **QA**: 255 (Activation scaling)
-   **QB**: 64 (Final score scaling)

The `internal/nnue/nnue.go` loader expects a flat binary file containing:
1.  `FeatureWeights`: `[768][256] int16`
2.  `FeatureBiases`: `[256] int16`
3.  `OutputWeights`: `[512] int16`
4.  `OutputBias`: `int32`

## 6. Loading the Network in Axon
Place your final `.nnue` file in the engine's root directory and enable it via the UCI interface:

```bash
setoption name EvalFile value axon_v1.nnue
setoption name Use NNUE value true
```

Run the `bench` command to verify performance and ensure the weights have loaded correctly.
```bash
bench
```
