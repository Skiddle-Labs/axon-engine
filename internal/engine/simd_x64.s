#include "textflag.h"

// func piecesToCharsAVX2(pieces *types.Piece, chars *byte)
// pieces: pointer to [64]int8 (mailbox array)
// chars: pointer to [64]byte buffer
TEXT ·piecesToCharsAVX2(SB), NOSPLIT, $0-16
    MOVQ pieces+0(FP), DI
    MOVQ chars+8(FP), SI

    // Load the lookup table into YMM0.
    // VPSHUFB operates lane-wise (128-bit), so we replicate the 16-byte table
    // in both the lower and upper halves of the 256-bit register.
    // Table: . P N B R Q K p n b r q k \0 \0 \0
    LEAQ pieces_table<>(SB), DX
    VMOVDQU (DX), Y0

    // Process first 32 pieces (A1-H4)
    VMOVDQU (DI), Y1
    VPSHUFB Y1, Y0, Y2
    VMOVDQU Y2, (SI)

    // Process next 32 pieces (A5-H8)
    VMOVDQU 32(DI), Y1
    VPSHUFB Y1, Y0, Y2
    VMOVDQU Y2, 32(SI)

    VZEROUPPER
    RET

// Lookup table for VPSHUFB.
// Indices 0-12 correspond to types.Piece values.
// Low 8 bytes: . P N B R Q K p
DATA pieces_table<>+0x00(SB)/8, $0x704B5152424E502E
// High 8 bytes: n b r q k \0 \0 \0
DATA pieces_table<>+0x08(SB)/8, $0x0000006B7172626E
// Replicated for the second 128-bit lane
DATA pieces_table<>+0x10(SB)/8, $0x704B5152424E502E
DATA pieces_table<>+0x18(SB)/8, $0x0000006B7172626E
GLOBL pieces_table<>(SB), RODATA, $32
