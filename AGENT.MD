# Agent Notes

## Overview
Axon is a Go (Golang) chess engine that speaks UCI. The primary entry point is `main.go`.

## Common Commands

### Build (macOS/Linux)
*Standard build:*
```bash
go build -o axon main.go
```

*Release build (Optimized, strips debug info):*
```bash
go build -ldflags="-s -w" -o axon main.go
```

### Build (Windows, cmd.exe)
*Standard build:*
```bat
go build -o axon.exe main.go
```

*Release build:*
```bat
go build -ldflags="-s -w" -o axon.exe main.go
```

### Run (macOS/Linux)
```bash
./axon
```

### Run (Windows, cmd.exe)
```bat
axon.exe
```

### Tests
*Run all tests:*
```bash
go test ./...
```

## Roadmap Phase
- [x] **Phase 1: UCI & Config**: document all options; add defaults for threads/hash/MultiPV; verify parsing coverage.
- [ ] **Phase 2: Opening Book**: add Polyglot book support; allow on/off and book file path option.
- [ ] **Phase 3: Search Strength**: deepen LMR/NMP/razoring tuning; improve move ordering; add aspiration windows.
- [ ] **Phase 4: Evaluation**: expand pawn structure, king safety, mobility; calibrate PSTs; add eval regression tests.
- [ ] **Phase 5: Endgame**: add basic tablebase probing (optional) and endgame-specific heuristics.
- [ ] **Phase 6: Testing & Tuning**: add bench suite, record baseline NPS/score; run SPRT matches for changes.

## Key Paths
- `internal/engine`: board representation, move generation, attacks, SEE, Zobrist.
- `internal/search`: search, TT, move ordering.
- `internal/eval`: evaluation.
- `internal/protocol`: UCI protocol handling.