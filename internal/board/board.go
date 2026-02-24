package board

import (
	"fmt"
	"strconv"
	"strings"
)

type CastlingRights uint8

// State represents the irreversible state of the board, used for unmaking moves.
type State struct {
	EnPassant     Square
	Castling      CastlingRights
	HalfMoveClock uint8
	CapturedPiece Piece
	Hash          uint64
}

// Board represents the state of a chess game.
// It uses bitboards to store the positions of all pieces.
type Board struct {
	// Piece bitboards: [Color][PieceType]
	Pieces [2][7]Bitboard

	// Combined bitboards for convenience
	Colors [2]Bitboard

	// Game state variables
	SideToMove Color
	EnPassant  Square
	Castling   CastlingRights
	Hash       uint64

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
		SideToMove: White,
		EnPassant:  NoSquare,
	}
}

// Occupancy returns a bitboard of all pieces on the board.
func (b *Board) Occupancy() Bitboard {
	return b.Colors[White] | b.Colors[Black]
}

// HasMajorPieces returns true if the given color has non-pawn/non-king pieces.
func (b *Board) HasMajorPieces(c Color) bool {
	return b.Pieces[c][Knight] != 0 ||
		b.Pieces[c][Bishop] != 0 ||
		b.Pieces[c][Rook] != 0 ||
		b.Pieces[c][Queen] != 0
}

// PieceAt returns the piece at the given square.
func (b *Board) PieceAt(s Square) Piece {
	for c := White; c <= Black; c++ {
		if !b.Colors[c].Test(s) {
			continue
		}
		for pt := Pawn; pt <= King; pt++ {
			if b.Pieces[c][pt].Test(s) {
				return makePiece(c, pt)
			}
		}
	}
	return NoPiece
}

func (b *Board) String() string {
	var sb strings.Builder
	sb.WriteString("  +---+---+---+---+---+---+---+---+\n")
	for r := 7; r >= 0; r-- {
		sb.WriteString(fmt.Sprintf("%d |", r+1))
		for f := 0; f < 8; f++ {
			p := b.PieceAt(NewSquare(f, r))
			char := "."
			switch p {
			case WhitePawn:
				char = "P"
			case WhiteKnight:
				char = "N"
			case WhiteBishop:
				char = "B"
			case WhiteRook:
				char = "R"
			case WhiteQueen:
				char = "Q"
			case WhiteKing:
				char = "K"
			case BlackPawn:
				char = "p"
			case BlackKnight:
				char = "n"
			case BlackBishop:
				char = "b"
			case BlackRook:
				char = "r"
			case BlackQueen:
				char = "q"
			case BlackKing:
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
			if piece == NoPiece {
				return fmt.Errorf("invalid piece in FEN: %c", char)
			}
			sq := NewSquare(file, rank)
			b.Pieces[color][pt].Set(sq)
			b.Colors[color].Set(sq)
			file++
		}
	}

	// 2. Side to move
	if fields[1] == "w" {
		b.SideToMove = White
	} else {
		b.SideToMove = Black
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
		b.EnPassant = NoSquare
	} else {
		if len(fields[3]) == 2 {
			f := int(fields[3][0] - 'a')
			r := int(fields[3][1] - '1')
			b.EnPassant = NewSquare(f, r)
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

	return nil
}

// Clear resets the board to an empty state.
func (b *Board) Clear() {
	b.Pieces = [2][7]Bitboard{}
	b.Colors = [2]Bitboard{}
	b.SideToMove = White
	b.EnPassant = NoSquare
	b.Castling = 0
	b.HalfMoveClock = 0
	b.FullMoveNumber = 1
	b.Ply = 0
	b.Hash = 0
}

func pieceFromChar(c rune) (Piece, Color, PieceType) {
	switch c {
	case 'P':
		return WhitePawn, White, Pawn
	case 'N':
		return WhiteKnight, White, Knight
	case 'B':
		return WhiteBishop, White, Bishop
	case 'R':
		return WhiteRook, White, Rook
	case 'Q':
		return WhiteQueen, White, Queen
	case 'K':
		return WhiteKing, White, King
	case 'p':
		return BlackPawn, Black, Pawn
	case 'n':
		return BlackKnight, Black, Knight
	case 'b':
		return BlackBishop, Black, Bishop
	case 'r':
		return BlackRook, Black, Rook
	case 'q':
		return BlackQueen, Black, Queen
	case 'k':
		return BlackKing, Black, King
	}
	return NoPiece, NoColor, None
}

// Helper to combine Color and PieceType into a Piece (internal use)
func makePiece(c Color, pt PieceType) Piece {
	if pt == None {
		return NoPiece
	}
	if c == White {
		return Piece(pt)
	}
	return Piece(pt + 6)
}
