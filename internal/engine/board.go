package engine

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Skiddle-Labs/axon-engine/internal/types"
)

type CastlingRights uint8

// State represents the irreversible state of the board, used for unmaking moves.
type State struct {
	Move          Move
	EnPassant     types.Square
	Castling      CastlingRights
	HalfMoveClock uint8
	CapturedPiece types.Piece
	Hash          uint64
	PawnHash      uint64
	Accumulators  [2]types.Accumulator
}

// Board represents the state of a chess game.
// It uses bitboards to store the positions of all pieces.
type Board struct {
	// Piece bitboards: [Color][PieceType]
	Pieces [2][7]Bitboard

	// Combined bitboards for convenience
	Colors [2]Bitboard

	// Game state variables
	SideToMove types.Color
	EnPassant  types.Square
	Castling   CastlingRights
	Hash       uint64
	PawnHash   uint64

	// NNUE Accumulators (for White and Black perspectives)
	Accumulators [2]types.Accumulator

	// History and counters
	HalfMoveClock  uint8
	FullMoveNumber uint16

	// History for unmaking moves
	History [1024]State
	Ply     int
}

// Bitboard constants for castling rights
const (
	WhiteKingside CastlingRights = 1 << iota
	WhiteQueenside
	BlackKingside
	BlackQueenside
)

// NewBoard creates a new Board with an empty state.
func NewBoard() *Board {
	return &Board{
		SideToMove: types.White,
		EnPassant:  types.NoSquare,
	}
}

// Occupancy returns a bitboard of all pieces on the board.
func (b *Board) Occupancy() Bitboard {
	return b.Colors[types.White] | b.Colors[types.Black]
}

// HasMajorPieces returns true if the given color has non-pawn/non-king pieces.
func (b *Board) HasMajorPieces(c types.Color) bool {
	return b.Pieces[c][types.Knight] != 0 ||
		b.Pieces[c][types.Bishop] != 0 ||
		b.Pieces[c][types.Rook] != 0 ||
		b.Pieces[c][types.Queen] != 0
}

// PieceAt returns the piece at the given square.
func (b *Board) PieceAt(s types.Square) types.Piece {
	for c := types.White; c <= types.Black; c++ {
		if !b.Colors[c].Test(s) {
			continue
		}
		for pt := types.Pawn; pt <= types.King; pt++ {
			if b.Pieces[c][pt].Test(s) {
				return makePiece(c, pt)
			}
		}
	}
	return types.NoPiece
}

func (b *Board) String() string {
	var sb strings.Builder
	sb.WriteString("  +---+---+---+---+---+---+---+---+\n")
	for r := 7; r >= 0; r-- {
		sb.WriteString(fmt.Sprintf("%d |", r+1))
		for f := 0; f < 8; f++ {
			p := b.PieceAt(types.NewSquare(f, r))
			char := "."
			switch p {
			case types.WhitePawn:
				char = "P"
			case types.WhiteKnight:
				char = "N"
			case types.WhiteBishop:
				char = "B"
			case types.WhiteRook:
				char = "R"
			case types.WhiteQueen:
				char = "Q"
			case types.WhiteKing:
				char = "K"
			case types.BlackPawn:
				char = "p"
			case types.BlackKnight:
				char = "n"
			case types.BlackBishop:
				char = "b"
			case types.BlackRook:
				char = "r"
			case types.BlackQueen:
				char = "q"
			case types.BlackKing:
				char = "k"
			}
			sb.WriteString(fmt.Sprintf(" %s |", char))
		}
		sb.WriteString("\n  +---+---+---+---+---+---+---+---+\n")
	}
	sb.WriteString("    a   b   c   d   e   f   g   h\n")
	return sb.String()
}

// SetFEN sets the board state from a FEN string.
func (b *Board) SetFEN(fen string) error {
	b.Clear()
	fields := strings.Fields(fen)
	if len(fields) < 4 {
		return fmt.Errorf("invalid FEN: expected at least 4 fields")
	}

	// 1. Piece placement
	rank := 7
	file := 0
	for _, char := range fields[0] {
		switch char {
		case '/':
			rank--
			file = 0
		case '1', '2', '3', '4', '5', '6', '7', '8':
			file += int(char - '0')
		default:
			piece, color, pt := pieceFromChar(char)
			if piece == types.NoPiece {
				return fmt.Errorf("invalid piece in FEN: %c", char)
			}
			sq := types.NewSquare(file, rank)
			b.Pieces[color][pt].Set(sq)
			b.Colors[color].Set(sq)
			file++
		}
	}

	// 2. Side to move
	if fields[1] == "w" {
		b.SideToMove = types.White
	} else {
		b.SideToMove = types.Black
	}

	// 3. Castling rights
	b.Castling = 0
	if fields[2] != "-" {
		for _, char := range fields[2] {
			switch char {
			case 'K':
				b.Castling |= WhiteKingside
			case 'Q':
				b.Castling |= WhiteQueenside
			case 'k':
				b.Castling |= BlackKingside
			case 'q':
				b.Castling |= BlackQueenside
			}
		}
	}

	// 4. En passant square
	if fields[3] == "-" {
		b.EnPassant = types.NoSquare
	} else {
		if len(fields[3]) == 2 {
			f := int(fields[3][0] - 'a')
			r := int(fields[3][1] - '1')
			b.EnPassant = types.NewSquare(f, r)
		}
	}

	// 5. Halfmove clock
	if len(fields) >= 5 {
		val, _ := strconv.Atoi(fields[4])
		b.HalfMoveClock = uint8(val)
	}

	// 6. Fullmove number
	if len(fields) >= 6 {
		val, _ := strconv.Atoi(fields[5])
		b.FullMoveNumber = uint16(val)
	} else {
		b.FullMoveNumber = 1
	}

	b.Hash = b.ComputeHash()
	b.PawnHash = b.ComputePawnHash()
	b.RefreshAccumulators()

	return nil
}

// FEN returns the FEN string representing the current board state.
func (b *Board) FEN() string {
	var sb strings.Builder

	// 1. Piece placement
	for r := 7; r >= 0; r-- {
		empty := 0
		for f := 0; f < 8; f++ {
			p := b.PieceAt(types.NewSquare(f, r))
			if p == types.NoPiece {
				empty++
			} else {
				if empty > 0 {
					sb.WriteString(strconv.Itoa(empty))
					empty = 0
				}
				char := ""
				switch p {
				case types.WhitePawn:
					char = "P"
				case types.WhiteKnight:
					char = "N"
				case types.WhiteBishop:
					char = "B"
				case types.WhiteRook:
					char = "R"
				case types.WhiteQueen:
					char = "Q"
				case types.WhiteKing:
					char = "K"
				case types.BlackPawn:
					char = "p"
				case types.BlackKnight:
					char = "n"
				case types.BlackBishop:
					char = "b"
				case types.BlackRook:
					char = "r"
				case types.BlackQueen:
					char = "q"
				case types.BlackKing:
					char = "k"
				}
				sb.WriteString(char)
			}
		}
		if empty > 0 {
			sb.WriteString(strconv.Itoa(empty))
		}
		if r > 0 {
			sb.WriteString("/")
		}
	}

	// 2. Side to move
	if b.SideToMove == types.White {
		sb.WriteString(" w ")
	} else {
		sb.WriteString(" b ")
	}

	// 3. Castling rights
	if b.Castling == 0 {
		sb.WriteString("-")
	} else {
		if b.Castling&WhiteKingside != 0 {
			sb.WriteString("K")
		}
		if b.Castling&WhiteQueenside != 0 {
			sb.WriteString("Q")
		}
		if b.Castling&BlackKingside != 0 {
			sb.WriteString("k")
		}
		if b.Castling&BlackQueenside != 0 {
			sb.WriteString("q")
		}
	}

	// 4. En passant square
	sb.WriteString(" ")
	if b.EnPassant == types.NoSquare {
		sb.WriteString("-")
	} else {
		sb.WriteString(b.EnPassant.String())
	}

	// 5. Halfmove clock
	sb.WriteString(" ")
	sb.WriteString(strconv.Itoa(int(b.HalfMoveClock)))

	// 6. Fullmove number
	sb.WriteString(" ")
	sb.WriteString(strconv.Itoa(int(b.FullMoveNumber)))

	return sb.String()
}

// Clear resets the board to an empty state.
func (b *Board) Clear() {
	b.Pieces = [2][7]Bitboard{}
	b.Colors = [2]Bitboard{}
	b.SideToMove = types.White
	b.EnPassant = types.NoSquare
	b.Castling = 0
	b.HalfMoveClock = 0
	b.FullMoveNumber = 1
	b.Ply = 0
	b.Hash = 0
	b.PawnHash = 0
	b.Accumulators = [2]types.Accumulator{}
}

func pieceFromChar(c rune) (types.Piece, types.Color, types.PieceType) {
	switch c {
	case 'P':
		return types.WhitePawn, types.White, types.Pawn
	case 'N':
		return types.WhiteKnight, types.White, types.Knight
	case 'B':
		return types.WhiteBishop, types.White, types.Bishop
	case 'R':
		return types.WhiteRook, types.White, types.Rook
	case 'Q':
		return types.WhiteQueen, types.White, types.Queen
	case 'K':
		return types.WhiteKing, types.White, types.King
	case 'p':
		return types.BlackPawn, types.Black, types.Pawn
	case 'n':
		return types.BlackKnight, types.Black, types.Knight
	case 'b':
		return types.BlackBishop, types.Black, types.Bishop
	case 'r':
		return types.BlackRook, types.Black, types.Rook
	case 'q':
		return types.BlackQueen, types.Black, types.Queen
	case 'k':
		return types.BlackKing, types.Black, types.King
	}
	return types.NoPiece, types.NoColor, types.None
}

// Helper to combine Color and PieceType into a Piece (internal use)
func makePiece(c types.Color, pt types.PieceType) types.Piece {
	if pt == types.None {
		return types.NoPiece
	}
	if c == types.White {
		return types.Piece(pt)
	}
	return types.Piece(pt + 6)
}
