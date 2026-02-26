package uci

import (
	"strconv"
	"strings"

	"github.com/Skiddle-Labs/axon-engine/internal/engine"
	"github.com/Skiddle-Labs/axon-engine/internal/nnue"
	"github.com/Skiddle-Labs/axon-engine/internal/search"
)

func (u *UCI) handleUCI() {
	u.send("id name Axon Engine (NNUE: " + nnue.NetworkName + ")")
	u.send("id author Skiddle Labs")

	// Options
	u.send("option name Threads type spin default 1 min 1 max 512")
	u.send("option name MultiPV type spin default 1 min 1 max 500")
	u.send("option name Hash type spin default 64 min 1 max 1048576")
	u.send("option name Move Overhead type spin default 10 min 0 max 5000")
	u.send("option name Slow Mover type spin default 100 min 10 max 1000")
	u.send("option name Clear Hash type button")
	u.send("option name OwnBook type check default true")
	u.send("option name BookPath type string default ")
	u.send("option name EvalFile type string default ")
	u.send("option name Use NNUE type check default true")
	u.send("option name UCI_ShowWDL type check default false")
	u.send("option name AspirationDelta type spin default 15 min 1 max 500")
	u.send("option name RFP_Margin type spin default 75 min 0 max 5000")
	u.send("option name FP_Margin type spin default 100 min 0 max 5000")
	u.send("option name NMP_Base type spin default 3 min 1 max 10")
	u.send("option name NMP_Divisor type spin default 6 min 1 max 20")
	u.send("option name LMR_Base type spin default 75 min 0 max 200")
	u.send("option name LMR_Multiplier type spin default 225 min 0 max 500")
	u.send("option name MC_R type spin default 3 min 1 max 10")
	u.send("option name MC_M type spin default 6 min 1 max 20")
	u.send("option name MC_C type spin default 3 min 1 max 10")

	u.send("uciok")
}

func (u *UCI) handleSetOption(fields []string) {
	// Format: setoption name <name> [value <value>]
	name := ""
	value := ""
	for i := 1; i < len(fields); i++ {
		if fields[i] == "name" && i+1 < len(fields) {
			name = strings.ToLower(fields[i+1])
			// Multi-word names
			curr := i + 2
			for curr < len(fields) && fields[curr] != "value" {
				name += " " + strings.ToLower(fields[curr])
				curr++
			}
			i = curr - 1
		} else if fields[i] == "value" {
			if i+1 < len(fields) {
				value = strings.Join(fields[i+1:], " ")
			}
			break
		}
	}

	switch name {
	case "threads":
		if v, err := strconv.Atoi(value); err == nil {
			u.threads = v
		}
	case "multipv":
		if v, err := strconv.Atoi(value); err == nil {
			u.multiPV = v
		}
	case "hash":
		if v, err := strconv.Atoi(value); err == nil {
			search.GlobalTT = search.NewTranspositionTable(v)
		}
	case "move overhead":
		if v, err := strconv.Atoi(value); err == nil {
			u.moveOverhead = v
		}
	case "slow mover":
		if v, err := strconv.Atoi(value); err == nil {
			u.slowMover = v
		}
	case "clear hash":
		search.GlobalTT.Clear()
	case "ownbook":
		u.bookBestMove = (value == "true")
	case "bookpath":
		if value != "" {
			book, err := engine.OpenBook(value)
			if err == nil {
				if u.book != nil {
					u.book.Close()
				}
				u.book = book
			}
		}
	case "evalfile":
		if value != "" {
			if err := nnue.LoadNetwork(value); err == nil {
				u.board.RefreshAccumulators()
				u.send("info string Loaded network: " + value)
			} else {
				u.send("info string Error loading network: " + err.Error())
			}
		}
	case "use nnue":
		nnue.UseNNUE = (value == "true")
	case "uci_showwdl":
		u.showWDL = (value == "true")
	case "aspirationdelta":
		if v, err := strconv.Atoi(value); err == nil {
			search.AspirationDelta = v
		}
	case "rfp_margin":
		if v, err := strconv.Atoi(value); err == nil {
			search.RFPMargin = v
		}
	case "fp_margin":
		if v, err := strconv.Atoi(value); err == nil {
			search.FPMargin = v
		}
	case "nmp_base":
		if v, err := strconv.Atoi(value); err == nil {
			search.NMPBase = v
		}
	case "nmp_divisor":
		if v, err := strconv.Atoi(value); err == nil {
			search.NMPDivisor = v
		}
	case "lmr_base":
		if v, err := strconv.Atoi(value); err == nil {
			search.LMR_Base = float64(v) / 100.0
			search.UpdateLMR(search.LMR_Base, search.LMR_Multiplier)
		}
	case "lmr_multiplier":
		if v, err := strconv.Atoi(value); err == nil {
			search.LMR_Multiplier = float64(v) / 100.0
			search.UpdateLMR(search.LMR_Base, search.LMR_Multiplier)
		}
	case "mc_r":
		if v, err := strconv.Atoi(value); err == nil {
			search.MC_R = v
		}
	case "mc_m":
		if v, err := strconv.Atoi(value); err == nil {
			search.MC_M = v
		}
	case "mc_c":
		if v, err := strconv.Atoi(value); err == nil {
			search.MC_C = v
		}
	}
}
