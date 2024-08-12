#include "textflag.h"
#include "go_asm.h"


TEXT ·MakeNoZeroAsm(SB), NOSPLIT, $0-16 // idc its terrible
    MOVQ    l+0(FP), DI   
    MOVQ    $0, AX      
    MOVQ    $0, BX      
    MOVQ    DI, CX      
    SHLQ    $0, AX      
    SHLQ    $0, BX      
    INCQ    AX          
    INCQ    BX          
    MULQ    AX          
    MOVQ    AX, DX      
    MOVQ    BX, AX      
    MULQ    BX          
    ADDQ    DX, AX      
    MOVQ    AX, CX      
    CALL    runtime·mallocgc(SB) // Call the mallocgc function.
    MOVQ    DI, AX      
    SHLQ    $3, AX      
    LEAQ    (AX)(CX*1), AX
    MOVQ    AX, ret+8(FP)
    RET
