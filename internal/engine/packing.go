package engine

import (
	"encoding/binary"
	"io"

	"github.com/Skiddle-Labs/axon-engine/internal/types"
)

// PackedPos represents a chess position in a compact 32-byte binary format.
// This is used for efficient storage of training data.
// Layout:
// - Occupancy: 8 bytes (64-bit bitboard)
// - Pieces: 16 bytes (Up to 32 pieces, 4 bits each)
// - Score: 2 bytes (int16)
// - Result: 1 byte (int8: 1=Win, 0=Draw, -1=Loss)
// - Side: 1 byte (0=White, 1=Black)
// - Castling: 1 byte (4 bits)
// - EP: 1 byte (Square index or -1)
// - Padding: 2 bytes
type PackedPos struct {
	Occupancy uint64
	Pieces    [16]byte
	Score     int16
	Result    int8
	Side      int8
	Castling  uint8
	EP        int8
	Padding   [2]byte
}

// PackedPosSize is the size of the packed position in bytes.
const PackedPosSize = 32

// Pack converts a Board and its metadata into a PackedPos.
func (b *Board) Pack(score int, result int) PackedPos {
	var p PackedPos
	occ := b.Occupancy()
	p.Occupancy = uint64(occ)

	// Pack pieces (up to 32)
	pieceCount := 0
	for occ != 0 {
		sq := occ.PopLSB()
		piece := b.PieceAt(sq)
		if pieceCount < 32 {
			shift := uint((pieceCount % 2) * 4)
			p.Pieces[pieceCount/2] |= byte(piece) << shift
		}
		pieceCount++
	}

	p.Score = int16(score)
	p.Result = int8(result)
	p.Side = int8(b.SideToMove)
	p.Castling = uint8(b.Castling)
	if b.EnPassant == types.NoSquare {
		p.EP = -1
	} else {
		p.EP = int8(b.EnPassant)
	}

	return p
}

// Unpack restores a Board from a PackedPos.
// Note: This does not restore history or move counters, only the immediate state.
func (p *PackedPos) Unpack() (*Board, int, int) {
	b := NewBoard()
	b.Clear()

	occ := Bitboard(p.Occupancy)
	pieceCount := 0
	for occ != 0 {
		sq := occ.PopLSB()
		shift := uint((pieceCount % 2) * 4)
		piece := types.Piece((p.Pieces[pieceCount/2] >> shift) & 0x0F)

		if piece != types.NoPiece {
			c := piece.Color()
			pt := piece.Type()
			b.Pieces[c][pt].Set(sq)
			b.Colors[c].Set(sq)
			b.PieceArray[sq] = piece
		}
		pieceCount++
	}

	b.SideToMove = types.Color(p.Side)
	b.Castling = CastlingRights(p.Castling)
	if p.EP == -1 {
		b.EnPassant = types.NoSquare
	} else {
		b.EnPassant = types.Square(p.EP)
	}

	// Recompute hashes and accumulators
	b.Hash = b.ComputeHash()
	b.PawnHash = b.ComputePawnHash()
	b.RefreshAccumulators()

	return b, int(p.Score), int(p.Result)
}

// Serialize writes the PackedPos to an io.Writer.
func (p *PackedPos) Serialize(w io.Writer) error {
	return binary.Write(w, binary.LittleEndian, p)
}

// Deserialize reads a PackedPos from an io.Reader.
func Deserialize(r io.Reader) (PackedPos, error) {
	var p PackedPos
	err := binary.Read(r, binary.LittleEndian, &p)
	return p, err
}
