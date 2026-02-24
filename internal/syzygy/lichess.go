package syzygy

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/personal-github/axon-engine/internal/engine"
)

// LichessResult represents the response from the Lichess Tablebase API.
type LichessResult struct {
	Category string `json:"category"`
	DTZ      int    `json:"dtz"`
	DTM      int    `json:"dtm"`
	Moves    []struct {
		UCI      string `json:"uci"`
		Category string `json:"category"`
		DTZ      int    `json:"dtz"`
	} `json:"moves"`
}

const lichessBaseURL = "https://tablebase.lichess.ovh/standard"

var httpClient = &http.Client{
	Timeout: 5 * time.Second,
}

// ProbeLichessWDL probes the Lichess Tablebase API for the Win-Draw-Loss status.
// This is slow (network request) and should only be used at the root or for analysis.
func ProbeLichessWDL(b *engine.Board) (int, bool) {
	// Syzygy tablebases do not support castling positions.
	if b.Castling != 0 {
		return WDLNotFound, false
	}

	// Only 7 pieces or fewer are supported by Lichess API.
	pieceCount := b.Colors[engine.White].Count() + b.Colors[engine.Black].Count()
	if pieceCount > 7 {
		return WDLNotFound, false
	}

	fen := b.FEN()
	query := url.Values{}
	query.Set("fen", fen)

	resp, err := httpClient.Get(lichessBaseURL + "?" + query.Encode())
	if err != nil {
		return WDLNotFound, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return WDLNotFound, false
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return WDLNotFound, false
	}

	var result LichessResult
	if err := json.Unmarshal(body, &result); err != nil {
		return WDLNotFound, false
	}

	switch result.Category {
	case "win":
		return WDLWin, true
	case "maybe-win":
		return WDLCursed, true
	case "draw":
		return WDLDraw, true
	case "maybe-loss":
		return WDLBlessed, true
	case "loss":
		return WDLLoss, true
	default:
		return WDLNotFound, false
	}
}

// GetLichessBestMove probes the Lichess Tablebase API and returns the best move in UCI format.
func GetLichessBestMove(b *engine.Board) (string, bool) {
	if b.Castling != 0 {
		return "", false
	}

	fen := b.FEN()
	query := url.Values{}
	query.Set("fen", fen)

	resp, err := httpClient.Get(lichessBaseURL + "?" + query.Encode())
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", false
	}

	var result LichessResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", false
	}

	if len(result.Moves) == 0 {
		return "", false
	}

	// The API usually returns moves sorted by quality.
	// For wins, we want the move with the most negative DTZ (fastest win).
	// For draws, any draw move.
	// For losses, the move with the highest DTZ (slowest loss).

	bestIdx := 0
	return result.Moves[bestIdx].UCI, true
}

// MapLichessCategory maps the API category string to internal WDL constants.
func MapLichessCategory(cat string) int {
	switch cat {
	case "win":
		return WDLWin
	case "maybe-win":
		return WDLCursed
	case "draw":
		return WDLDraw
	case "maybe-loss":
		return WDLBlessed
	case "loss":
		return WDLLoss
	default:
		return WDLNotFound
	}
}
