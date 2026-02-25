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

// func evaluateAVX2(us, them *types.Accumulator, weights *int16, bias int16) int32
TEXT ·evaluateAVX2(SB), NOSPLIT, $0-36
    MOVQ us+0(FP), DI
    MOVQ them+8(FP), SI
    MOVQ weights+16(FP), DX
    MOVWQSX bias+24(FP), R8 // Load and sign-extend bias to 64-bit

    VPXOR Y4, Y4, Y4    // Main sum accumulator (int64) - Low parts
    VPXOR Y5, Y5, Y5    // Main sum accumulator (int64) - High parts
    VPXOR Y6, Y6, Y6    // Constant zero for clamping

    // Load constant 255 for SCReLU clamping (QA)
    MOVL $255, AX
    VMOVQ AX, X0
    VPBROADCASTW X0, Y7 // Y7 = [255, 255, ..., 255] (int16)

    // Perspective 'Us' (256 elements)
    // We process 8 elements at a time to allow expansion to 32-bit then 64-bit
    MOVQ $32, CX        // 256 / 8 = 32 iterations
us_loop:
    VMOVDQU (DI), X0    // Load 8 values (int16)
    VPMAXSW X6, X0, X0  // x = max(x, 0)
    VPMINSW X7, X0, X0  // x = min(x, 255)

    VPMULLW X0, X0, X0  // x^2 (low 16 bits of 255^2 is correct)
    VPMOVZXWD X0, Y2    // Y2 = [x^2_0, ..., x^2_7] (int32)

    VMOVDQU (DX), X1    // Load 8 weights (int16)
    VPMOVSXWD X1, Y3    // Y3 = [w_0, ..., w_7] (int32)

    VPMULLD Y2, Y3, Y2  // Y2 = [x^2*w_0, ..., x^2*w_7] (int32)

    VPMOVSXDQ X2, Y8    // Low 4 elements of Y2 to int64
    VEXTRACTI128 $1, Y2, X9
    VPMOVSXDQ X9, Y9    // High 4 elements of Y2 to int64

    VPADDQ Y8, Y4, Y4
    VPADDQ Y9, Y5, Y5

    ADDQ $16, DI
    ADDQ $16, DX
    DECQ CX
    JNZ us_loop

    // Perspective 'Them' (256 elements)
    MOVQ $32, CX
them_loop:
    VMOVDQU (SI), X0
    VPMAXSW X6, X0, X0
    VPMINSW X7, X0, X0

    VPMULLW X0, X0, X0
    VPMOVZXWD X0, Y2

    VMOVDQU (DX), X1
    VPMOVSXWD X1, Y3

    VPMULLD Y2, Y3, Y2

    VPMOVSXDQ X2, Y8
    VEXTRACTI128 $1, Y2, X9
    VPMOVSXDQ X9, Y9

    VPADDQ Y8, Y4, Y4
    VPADDQ Y9, Y5, Y5

    ADDQ $16, SI
    ADDQ $16, DX
    DECQ CX
    JNZ them_loop

    // Horizontal sum of Y4 and Y5 (int64)
    VPADDQ Y4, Y5, Y4    // Y4 = [sum0, sum1, sum2, sum3] (int64)
    VEXTRACTI128 $1, Y4, X0
    VPADDQ X0, X4, X0    // X0 = [sum0+2, sum1+3]
    VMOVQ X0, R9
    VPEXTRQ $1, X0, R10
    ADDQ R10, R9         // R9 = final 64-bit sum

    // Final Quantization Logic:
    // internalScore = (output / 255) + bias
    // return (internalScore * 400) / (255 * 64)

    // 1. output / 255
    MOVQ R9, AX
    MOVQ $255, CX
    CQO                  // Sign extend RAX into RDX:RAX
    IDIVQ CX             // RAX = output / 255

    // 2. Add bias
    ADDQ R8, AX          // RAX = internalScore

    // 3. Scale by EvalScale (400)
    IMULQ $400, AX       // RAX = internalScore * 400

    // 4. Divide by QAB (255 * 64 = 16320)
    MOVQ $16320, CX
    CQO
    IDIVQ CX             // RAX = final centipawn score

    MOVL AX, ret+32(FP)
    VZEROUPPER
    RET
