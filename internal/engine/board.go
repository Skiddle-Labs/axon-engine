package engine

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Skiddle-Labs/axon-engine/internal/types"
)

type CastlingRights uint8

const StartFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

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

	// Mailbox for O(1) lookups
	PieceArray [64]types.Piece

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
	return b.PieceArray[s]
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
			b.PieceArray[sq] = piece
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
	sb.Grow(90)

	// 1. Piece placement
	for r := 7; r >= 0; r-- {
		empty := 0
		for f := 0; f < 8; f++ {
			p := b.PieceArray[r*8+f]
			if p == types.NoPiece {
				empty++
			} else {
				if empty > 0 {
					sb.WriteByte(byte('0' + empty))
					empty = 0
				}
				sb.WriteByte(".PNBRQKpnbrqk"[p])
			}
		}
		if empty > 0 {
			sb.WriteByte(byte('0' + empty))
		}
		if r > 0 {
			sb.WriteByte('/')
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
		sb.WriteByte('-')
	} else {
		if b.Castling&WhiteKingside != 0 {
			sb.WriteByte('K')
		}
		if b.Castling&WhiteQueenside != 0 {
			sb.WriteByte('Q')
		}
		if b.Castling&BlackKingside != 0 {
			sb.WriteByte('k')
		}
		if b.Castling&BlackQueenside != 0 {
			sb.WriteByte('q')
		}
	}

	// 4. En passant square
	sb.WriteByte(' ')
	if b.EnPassant == types.NoSquare {
		sb.WriteByte('-')
	} else {
		sb.WriteByte(byte('a' + (b.EnPassant & 7)))
		sb.WriteByte(byte('1' + (b.EnPassant >> 3)))
	}

	// 5. Halfmove clock
	sb.WriteByte(' ')
	sb.WriteString(strconv.Itoa(int(b.HalfMoveClock)))

	// 6. Fullmove number
	sb.WriteByte(' ')
	sb.WriteString(strconv.Itoa(int(b.FullMoveNumber)))

	return sb.String()
}

// Clear resets the board to an empty state.
// hasEnPassantCapture returns true if the current side to move can capture en passant.
// This is used to ensure the Zobrist hash only includes the en passant file when a
// capture is actually possible, reducing hash collisions.
func (b *Board) hasEnPassantCapture() bool {
	if b.EnPassant == types.NoSquare {
		return false
	}

	us := b.SideToMove
	pawns := b.Pieces[us][types.Pawn]
	epFile := b.EnPassant.File()

	if us == types.White {
		// White EP capture happens from rank 4 (index 4) to rank 5.
		// Only pawns on rank 4 can capture EP.
		rank4 := Bitboard(0xFF << 32)
		candidates := pawns & rank4
		if candidates == 0 {
			return false
		}

		// Check if any pawn on rank 4 is adjacent to the EP file.
		if epFile > 0 && candidates.Test(types.NewSquare(epFile-1, 4)) {
			return true
		}
		if epFile < 7 && candidates.Test(types.NewSquare(epFile+1, 4)) {
			return true
		}
	} else {
		// Black EP capture happens from rank 3 (index 3) to rank 2.
		rank3 := Bitboard(0xFF << 24)
		candidates := pawns & rank3
		if candidates == 0 {
			return false
		}

		if epFile > 0 && candidates.Test(types.NewSquare(epFile-1, 3)) {
			return true
		}
		if epFile < 7 && candidates.Test(types.NewSquare(epFile+1, 3)) {
			return true
		}
	}

	return false
}

// Clear resets the board to an empty state.
func (b *Board) Clear() {
	b.Pieces = [2][7]Bitboard{}
	b.Colors = [2]Bitboard{}
	b.PieceArray = [64]types.Piece{}
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
