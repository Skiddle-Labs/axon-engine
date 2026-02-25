package uci

import (
	"strings"

	"github.com/Skiddle-Labs/axon-engine/internal/engine"
)

func (u *UCI) handlePosition(fields []string) {
	if len(fields) < 2 {
		return
	}

	index := 2
	if fields[1] == "startpos" {
		u.board.SetFEN(engine.StartFEN)
	} else if fields[1] == "fen" {
		fenParts := []string{}
		for index < len(fields) && fields[index] != "moves" {
			fenParts = append(fenParts, fields[index])
			index++
		}
		u.board.SetFEN(strings.Join(fenParts, " "))
	}

	// Process moves
	for index < len(fields) {
		if fields[index] == "moves" {
			index++
			continue
		}
		moveStr := fields[index]
		move := u.parseMove(moveStr)
		if move != engine.NoMove {
			u.board.MakeMove(move)
		}
		index++
	}
}

func (u *UCI) parseMove(moveStr string) engine.Move {
	ml := u.board.GenerateMoves()
	for i := 0; i < ml.Count; i++ {
		if ml.Moves[i].String() == moveStr {
			return ml.Moves[i]
		}
	}
	return engine.NoMove
}
