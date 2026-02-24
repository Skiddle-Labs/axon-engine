//go:build !cgo
// +build !cgo

package syzygy

import "github.com/personal-github/axon-engine/internal/engine"

// Init is a no-op for non-CGO builds.
func Init(path string) error {
	return nil
}

// IsInitialized returns false for non-CGO builds.
func IsInitialized() bool {
	return false
}

// ProbeWDL returns WDLNotFound for non-CGO builds.
func ProbeWDL(b *engine.Board) (int, bool) {
	return WDLNotFound, false
}

// ProbeDTZ returns 0 for non-CGO builds.
func ProbeDTZ(b *engine.Board) (int, bool) {
	return 0, false
}
