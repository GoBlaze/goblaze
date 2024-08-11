// +build amd64,!appengine,gc

#include "textflag.h"

//  starts a timer.
TEXT ·startTimer(SB), NOSPLIT, $0
	MOVQ 0(TLS), SI
	CALL runtime·startTimer(SB)
	RET

//  stops a timer.
TEXT ·stopTimer(SB), NOSPLIT, $0
	MOVQ 0(TLS), SI
	CALL runtime·stopTimer(SB)
	RET

//  resets a timer.
TEXT ·resetTimer(SB), NOSPLIT, $0
	MOVQ 0(TLS), SI
	CALL runtime·resetTimer(SB)
	RET

// modTimer is an assembly function that modifies a timer.
TEXT ·modTimer(SB), NOSPLIT, $0
	MOVQ 0(TLS), SI
	CALL runtime·modTimer(SB)
	RET

