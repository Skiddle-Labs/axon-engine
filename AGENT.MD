# Agent Notes

## Overview
Axon is a Go (Golang) chess engine that speaks UCI. The primary entry point is `main.go`.

## Common Commands

### Build (Release)
```bash
go build -ldflags="-s -w" -o axon.exe main.go
```

### Tests
```bash
go test ./...
```

### Datagen (NNUE Training)
```bash
go build -o datagen.exe ./cmd/datagen
./datagen.exe -games 100000 -threads 8 -depth 6 -out training_data.epd
```

### Tuning (HCE)
```bash
go build -o tuner.exe ./cmd/tuner
./tuner.exe -file training_data.epd -method spsa -iterations 5000 -save tuned_params.txt
```

### Apply Tuned Params
```bash
go run cmd/apply/main.go tuned_params.txt
```

### UCI Utility Commands
- `bench`: Performance test.
- `eval`: Static evaluation breakdown.
- `count <file>`: Count positions in EPD.

## Project Structure
- `internal/types`: Core types and NNUE constants.
- `internal/engine`: Board, movegen, attacks, SEE, Zobrist.
- `internal/search`: Search logic, TT, ordering, SMP.
- `internal/eval`: HCE, PSTs, parameters.
- `internal/nnue`: NNUE architecture, accumulator logic, SIMD kernels.
- `internal/protocol/uci`: UCI protocol implementation.

## Guides
- [TUNING_GUIDE.md](TUNING_GUIDE.md): HCE parameter optimization.
- [TRAINING_GUIDE.md](TRAINING_GUIDE.md): NNUE training steps.
