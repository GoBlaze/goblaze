#include "textflag.h"
#include "go_asm.h"

// func bufApp(buf *[]byte, s string, w int, c byte)
TEXT ·bufApp(SB), NOSPLIT, $0-40
    // Load parameters
    MOVQ    buf+0(FP), BX     
    MOVQ    s_base+8(FP), SI  
    MOVQ    w+24(FP), DX      
    MOVB    c+32(FP), AL      

   
    MOVQ    (BX), AX         
    TESTQ   AX, AX           
    JZ      make_new_buf     

    MOVQ    8(AX), CX        
    CMPQ    CX, DX            
    JB      make_new_buf       

 
    MOVB    (AX)(DX*1), CL   
    CMPB    CL, AL           
    JE      done             

   
set_char:
    MOVB    AL, (AX)(DX*1)     
    RET

make_new_buf:
    LEAQ    1(DX), AX         
    MOVQ    AX, DI            
    CALL    ·MakeNoZero(SB)   
    MOVQ    AX, (BX)          

  
    TESTQ   AX, AX           
    JZ      done              

   
    MOVQ    SI, BX           
    MOVQ    8(AX), CX        
    ADDQ    CX, AX           
    MOVQ    AX, DX            
    MOVQ    (BX), BX          
    MOVQ    DX, CX            
    CALL    runtime·memmove(SB)

  
    MOVQ    SI, (AX)        

  
    JMP     set_char

done:
    RET
    