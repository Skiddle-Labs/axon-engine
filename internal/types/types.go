package types

// NNUE Architecture Constants
const (
	// InputFeatures represents the total number of input features (64 squares * 12 piece types)
	InputFeatures = 768

	// L1Size is the size of the first hidden layer (Accumulator)
	L1Size = 256
)

// Accumulator represents the first hidden layer of the NNUE.
// It is updated incrementally during MakeMove and UnmakeMove.
type Accumulator [L1Size]int16

// Color represents the player color (White or Black).
type Color int8

const (
	White Color = iota
	Black
	NoColor
)

func (c Color) String() string {
	switch c {
	case White:
		return "white"
	case Black:
		return "black"
	default:
		return "none"
	}
}

// Piece represents a chess piece.
type Piece int8

const (
	NoPiece Piece = iota
	WhitePawn
	WhiteKnight
	WhiteBishop
	WhiteRook
	WhiteQueen
	WhiteKing
	BlackPawn
	BlackKnight
	BlackBishop
	BlackRook
	BlackQueen
	BlackKing
)

func (p Piece) Color() Color {
	if p == NoPiece {
		return NoColor
	}
	if p <= WhiteKing {
		return White
	}
	return Black
}

func (p Piece) Type() PieceType {
	if p == NoPiece {
		return None
	}
	if p <= WhiteKing {
		return PieceType(p)
	}
	return PieceType(p - 6)
}

// FlippedColor returns the piece with its color flipped.
func (p Piece) FlippedColor() Piece {
	if p == NoPiece {
		return NoPiece
	}
	if p <= WhiteKing {
		return p + 6
	}
	return p - 6
}

// PieceType returns the type of the piece regardless of color.
type PieceType int8

const (
	None PieceType = iota
	Pawn
	Knight
	Bishop
	Rook
	Queen
	King
)

// Square represents one of the 64 squares on a chess board.
type Square int8

const (
	A1 Square = iota
	B1
	C1
	D1
	E1
	F1
	G1
	H1
	A2
	B2
	C2
	D2
	E2
	F2
	G2
	H2
	A3
	B3
	C3
	D3
	E3
	F3
	G3
	H3
	A4
	B4
	C4
	D4
	E4
	F4
	G4
	H4
	A5
	B5
	C5
	D5
	E5
	F5
	G5
	H5
	A6
	B6
	C6
	D6
	E6
	F6
	G6
	H6
	A7
	B7
	C7
	D7
	E7
	F7
	G7
	H7
	A8
	B8
	C8
	D8
	E8
	F8
	G8
	H8
	NoSquare
)

// File returns the file (column) of the square (0-7).
func (s Square) File() int {
	return int(s % 8)
}

// Rank returns the rank (row) of the square (0-7).
func (s Square) Rank() int {
	return int(s / 8)
}

// Flipped returns the square flipped vertically (for the other player's perspective).
func (s Square) Flipped() Square {
	return s ^ 56
}

func (s Square) String() string {
	if s < A1 || s >= NoSquare {
		return "-"
	}
	return string([]byte{byte('a' + s.File()), byte('1' + s.Rank())})
}

// NewSquare returns a Square from file and rank.
func NewSquare(file, rank int) Square {
	if file < 0 || file > 7 || rank < 0 || rank > 7 {
		return NoSquare
	}
	return Square(rank*8 + file)
}
