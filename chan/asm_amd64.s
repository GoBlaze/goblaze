#include "textflag.h"
#include "go_asm.h"

#define    get_tls(r)    MOVQ    (TLS), r
#define    g(r)    0(r)

TEXT ·GetG(SB),NOSPLIT,$0-8
    get_tls(AX)
    MOVQ    g(AX), AX
    MOVQ    AX, gp+0(FP)
    RET

