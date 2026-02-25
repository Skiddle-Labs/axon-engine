#include "textflag.h"

// func updateAccumulatorAVX2(acc *types.Accumulator, weights *int16)
TEXT ·updateAccumulatorAVX2(SB), NOSPLIT, $0-16
    MOVQ acc+0(FP), DI
    MOVQ weights+8(FP), SI

    MOVQ $16, CX        // 256 elements / 16 per 256-bit register = 16 iterations

update_loop:
    VMOVDQU (DI), Y0    // Load 16 accumulator weights
    VMOVDQU (SI), Y1    // Load 16 feature weights
    VPADDW Y1, Y0, Y0   // Vector add int16s
    VMOVDQU Y0, (DI)    // Store back to accumulator

    ADDQ $32, DI
    ADDQ $32, SI
    DECQ CX
    JNZ update_loop

    VZEROUPPER
    RET

// func subAccumulatorAVX2(acc *types.Accumulator, weights *int16)
TEXT ·subAccumulatorAVX2(SB), NOSPLIT, $0-16
    MOVQ acc+0(FP), DI
    MOVQ weights+8(FP), SI

    MOVQ $16, CX

sub_loop:
    VMOVDQU (DI), Y0
    VMOVDQU (SI), Y1
    VPSUBW Y1, Y0, Y0   // Vector subtract int16s
    VMOVDQU Y0, (DI)

    ADDQ $32, DI
    ADDQ $32, SI
    DECQ CX
    JNZ sub_loop

    VZEROUPPER
    RET

// func evaluateAVX2(us, them *types.Accumulator, weights *int16, bias int32) int32
TEXT ·evaluateAVX2(SB), NOSPLIT, $0-36
    MOVQ us+0(FP), DI
    MOVQ them+8(FP), SI
    MOVQ weights+16(FP), DX
    MOVL bias+24(FP), R8

    VPXOR Y4, Y4, Y4    // Main sum accumulator (int32)
    VPXOR Y6, Y6, Y6    // Constant zero for clamping

    // Load constant 127 for SCReLU clamping
    MOVL $127, AX
    VMOVQ AX, X0
    VPBROADCASTW X0, Y5 // Y5 = [127, 127, ..., 127] (int16)

    // 1. Process 'Us' perspective (256 elements)
    MOVQ $16, CX
us_loop:
    VMOVDQU (DI), Y0    // Load 16 values
    VPMAXSW Y6, Y0, Y0  // x = max(x, 0)
    VPMINSW Y5, Y0, Y0  // x = min(x, 127)

    VPMULLW Y0, Y0, Y0  // x = x * x (max 16129, fits in int16)

    VMOVDQU (DX), Y1    // Load 16 output weights
    VPMADDWD Y1, Y0, Y0 // Multiply and add pairs: Y0 = [x0*w0+x1*w1, x2*w2+x3*w3, ...] (int32)
    VPADDD Y0, Y4, Y4    // Accumulate into Y4

    ADDQ $32, DI
    ADDQ $32, DX
    DECQ CX
    JNZ us_loop

    // 2. Process 'Them' perspective (256 elements)
    MOVQ $16, CX
them_loop:
    VMOVDQU (SI), Y0
    VPMAXSW Y6, Y0, Y0
    VPMINSW Y5, Y0, Y0

    VPMULLW Y0, Y0, Y0

    VMOVDQU (DX), Y1
    VPMADDWD Y1, Y0, Y0
    VPADDD Y0, Y4, Y4

    ADDQ $32, SI
    ADDQ $32, DX
    DECQ CX
    JNZ them_loop

    // Horizontal sum of Y4 (8 x int32)
    VEXTRACTI128 $1, Y4, X0
    VPADDD X0, X4, X0
    VPHADDD X0, X0, X0
    VPHADDD X0, X0, X0
    VMOVD X0, AX         // AX = final accumulated sum (int32)

    // Final Quantization: (sum / 255 + bias) / 64
    CDQ                  // Sign extend EAX into EDX:EAX
    MOVL $255, CX
    IDIVL CX             // EAX = EAX / 255 (QA)

    ADDL R8, AX          // EAX += OutputBias

    // Truncate towards zero for division by 64 (matching Go's / 64 behavior)
    MOVL AX, DX
    SARL $31, DX         // DX = (AX < 0) ? -1 : 0
    ANDL $63, DX         // DX = (AX < 0) ? 63 : 0
    ADDL DX, AX          // Add correction for negative numbers
    SARL $6, AX          // Shift right arithmetic

    MOVL AX, ret+32(FP)
    VZEROUPPER
    RET
