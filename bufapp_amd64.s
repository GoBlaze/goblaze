#include "textflag.h"
#include "go_asm.h"

// func bufApp(buf *[]byte, s string, w int, c byte)
TEXT ·bufAppAsm(SB), NOSPLIT, $0-40 // i think its slower than go function :(
    // Load parameters
    MOVQ    buf+0(FP), BX
    MOVQ    s_base+8(FP), SI
    MOVQ    w+24(FP), DX
    MOVB    c+32(FP), AL
  
   

    // Check if buf is nil or len(buf) <= w
    MOVQ    (BX), AX
    CMPQ    AX, $0
    JEQ     make_new_buf
 

    MOVQ    8(AX), CX
    CMPQ    CX, DX
    JBE     make_new_buf
 
    MOVB    (AX)(DX*1), CL
    CMPB    CL, AL
    JE      done
check_char:
  
    MOVQ    16(AX), CX
    CMPQ    CX, DX
    JA      set_char
   

make_new_buf:
    LEAQ    1(DX), AX
    MOVQ    AX, DI
    CALL    ·MakeNoZero(SB)
    MOVQ    AX, (BX)
   

    TESTQ   AX, AX
    JZ      copy_str
   

    MOVQ    SI, BX
    MOVQ    8(AX), CX
    ADDQ    CX, AX
    MOVQ    AX, DX
    MOVQ    (BX), BX
    MOVQ    DX, CX
    CALL    runtime·memmove(SB)
  
copy_str:
    MOVQ    SI, (AX)
   
set_char:
    MOVB    AL, (AX)(DX*1)
   

done:
    RET

