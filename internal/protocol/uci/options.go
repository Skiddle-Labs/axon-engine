package uci

import (
	"strconv"
	"strings"

	"github.com/Skiddle-Labs/axon-engine/internal/engine"
	"github.com/Skiddle-Labs/axon-engine/internal/search"
)

func (u *UCI) handleUCI() {
	u.send("id name Axon Engine")
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
	}
}
