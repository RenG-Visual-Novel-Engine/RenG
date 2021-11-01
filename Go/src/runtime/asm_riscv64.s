// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#include "go_asm.h"
#include "funcdata.h"
#include "textflag.h"

// func rt0_go()
TEXT runtime·rt0_go(SB),NOSPLIT|TOPFRAME,$0
	// X2 = stack; A0 = argc; A1 = argv
	ADD	$-24, X2
	MOV	A0, 8(X2)	// argc
	MOV	A1, 16(X2)	// argv

	// create istack out of the given (operating system) stack.
	// _cgo_init may update stackguard.
	MOV	$runtime·g0(SB), g
	MOV	$(-64*1024), T0
	ADD	T0, X2, T1
	MOV	T1, g_stackguard0(g)
	MOV	T1, g_stackguard1(g)
	MOV	T1, (g_stack+stack_lo)(g)
	MOV	X2, (g_stack+stack_hi)(g)

	// if there is a _cgo_init, call it using the gcc ABI.
	MOV	_cgo_init(SB), T0
	BEQ	T0, ZERO, nocgo

	MOV	ZERO, A3		// arg 3: not used
	MOV	ZERO, A2		// arg 2: not used
	MOV	$setg_gcc<>(SB), A1	// arg 1: setg
	MOV	g, A0			// arg 0: G
	JALR	RA, T0

nocgo:
	// update stackguard after _cgo_init
	MOV	(g_stack+stack_lo)(g), T0
	ADD	$const__StackGuard, T0
	MOV	T0, g_stackguard0(g)
	MOV	T0, g_stackguard1(g)

	// set the per-goroutine and per-mach "registers"
	MOV	$runtime·m0(SB), T0

	// save m->g0 = g0
	MOV	g, m_g0(T0)
	// save m0 to g0->m
	MOV	T0, g_m(g)

	CALL	runtime·check(SB)

	// args are already prepared
	CALL	runtime·args(SB)
	CALL	runtime·osinit(SB)
	CALL	runtime·schedinit(SB)

	// create a new goroutine to start program
	MOV	$runtime·mainPC(SB), T0		// entry
	ADD	$-24, X2
	MOV	T0, 16(X2)
	MOV	ZERO, 8(X2)
	MOV	ZERO, 0(X2)
	CALL	runtime·newproc(SB)
	ADD	$24, X2

	// start this M
	CALL	runtime·mstart(SB)

	WORD $0 // crash if reached
	RET

TEXT runtime·mstart(SB),NOSPLIT|TOPFRAME,$0
	CALL	runtime·mstart0(SB)
	RET // not reached

// void setg_gcc(G*); set g called from gcc with g in A0
TEXT setg_gcc<>(SB),NOSPLIT,$0-0
	MOV	A0, g
	CALL	runtime·save_g(SB)
	RET

// func cputicks() int64
TEXT runtime·cputicks(SB),NOSPLIT,$0-8
	RDTIME	A0
	MOV	A0, ret+0(FP)
	RET

// systemstack_switch is a dummy routine that systemstack leaves at the bottom
// of the G stack. We need to distinguish the routine that
// lives at the bottom of the G stack from the one that lives
// at the top of the system stack because the one at the top of
// the system stack terminates the stack walk (see topofstack()).
TEXT runtime·systemstack_switch(SB), NOSPLIT, $0-0
	UNDEF
	JALR	RA, ZERO	// make sure this function is not leaf
	RET

// func systemstack(fn func())
TEXT runtime·systemstack(SB), NOSPLIT, $0-8
	MOV	fn+0(FP), CTXT	// CTXT = fn
	MOV	g_m(g), T0	// T0 = m

	MOV	m_gsignal(T0), T1	// T1 = gsignal
	BEQ	g, T1, noswitch

	MOV	m_g0(T0), T1	// T1 = g0
	BEQ	g, T1, noswitch

	MOV	m_curg(T0), T2
	BEQ	g, T2, switch

	// Bad: g is not gsignal, not g0, not curg. What is it?
	// Hide call from linker nosplit analysis.
	MOV	$runtime·badsystemstack(SB), T1
	JALR	RA, T1

switch:
	// save our state in g->sched. Pretend to
	// be systemstack_switch if the G stack is scanned.
	CALL	gosave_systemstack_switch<>(SB)

	// switch to g0
	MOV	T1, g
	CALL	runtime·save_g(SB)
	MOV	(g_sched+gobuf_sp)(g), T0
	MOV	T0, X2

	// call target function
	MOV	0(CTXT), T1	// code pointer
	JALR	RA, T1

	// switch back to g
	MOV	g_m(g), T0
	MOV	m_curg(T0), g
	CALL	runtime·save_g(SB)
	MOV	(g_sched+gobuf_sp)(g), X2
	MOV	ZERO, (g_sched+gobuf_sp)(g)
	RET

noswitch:
	// already on m stack, just call directly
	// Using a tail call here cleans up tracebacks since we won't stop
	// at an intermediate systemstack.
	MOV	0(CTXT), T1	// code pointer
	ADD	$8, X2
	JMP	(T1)

TEXT runtime·getcallerpc(SB),NOSPLIT|NOFRAME,$0-8
	MOV	0(X2), T0		// LR saved by caller
	MOV	T0, ret+0(FP)
	RET

/*
 * support for morestack
 */

// Called during function prolog when more stack is needed.
// Caller has already loaded:
// R1: framesize, R2: argsize, R3: LR
//
// The traceback routines see morestack on a g0 as being
// the top of a stack (for example, morestack calling newstack
// calling the scheduler calling newm calling gc), so we must
// record an argument size. For that purpose, it has no arguments.

// func morestack()
TEXT runtime·morestack(SB),NOSPLIT|NOFRAME,$0-0
	// Cannot grow scheduler stack (m->g0).
	MOV	g_m(g), A0
	MOV	m_g0(A0), A1
	BNE	g, A1, 3(PC)
	CALL	runtime·badmorestackg0(SB)
	CALL	runtime·abort(SB)

	// Cannot grow signal stack (m->gsignal).
	MOV	m_gsignal(A0), A1
	BNE	g, A1, 3(PC)
	CALL	runtime·badmorestackgsignal(SB)
	CALL	runtime·abort(SB)

	// Called from f.
	// Set g->sched to context in f.
	MOV	X2, (g_sched+gobuf_sp)(g)
	MOV	T0, (g_sched+gobuf_pc)(g)
	MOV	RA, (g_sched+gobuf_lr)(g)
	MOV	CTXT, (g_sched+gobuf_ctxt)(g)

	// Called from f.
	// Set m->morebuf to f's caller.
	MOV	RA, (m_morebuf+gobuf_pc)(A0)	// f's caller's PC
	MOV	X2, (m_morebuf+gobuf_sp)(A0)	// f's caller's SP
	MOV	g, (m_morebuf+gobuf_g)(A0)

	// Call newstack on m->g0's stack.
	MOV	m_g0(A0), g
	CALL	runtime·save_g(SB)
	MOV	(g_sched+gobuf_sp)(g), X2
	// Create a stack frame on g0 to call newstack.
	MOV	ZERO, -8(X2)	// Zero saved LR in frame
	ADD	$-8, X2
	CALL	runtime·newstack(SB)

	// Not reached, but make sure the return PC from the call to newstack
	// is still in this function, and not the beginning of the next.
	UNDEF

// func morestack_noctxt()
TEXT runtime·morestack_noctxt(SB),NOSPLIT|NOFRAME,$0-0
	MOV	ZERO, CTXT
	JMP	runtime·morestack(SB)

// AES hashing not implemented for riscv64
TEXT runtime·memhash(SB),NOSPLIT|NOFRAME,$0-32
	JMP	runtime·memhashFallback(SB)
TEXT runtime·strhash(SB),NOSPLIT|NOFRAME,$0-24
	JMP	runtime·strhashFallback(SB)
TEXT runtime·memhash32(SB),NOSPLIT|NOFRAME,$0-24
	JMP	runtime·memhash32Fallback(SB)
TEXT runtime·memhash64(SB),NOSPLIT|NOFRAME,$0-24
	JMP	runtime·memhash64Fallback(SB)

// func return0()
TEXT runtime·return0(SB), NOSPLIT, $0
	MOV	$0, A0
	RET

// restore state from Gobuf; longjmp

// func gogo(buf *gobuf)
TEXT runtime·gogo(SB), NOSPLIT|NOFRAME, $0-8
	MOV	buf+0(FP), T0
	MOV	gobuf_g(T0), T1
	MOV	0(T1), ZERO // make sure g != nil
	JMP	gogo<>(SB)

TEXT gogo<>(SB), NOSPLIT|NOFRAME, $0
	MOV	T1, g
	CALL	runtime·save_g(SB)

	MOV	gobuf_sp(T0), X2
	MOV	gobuf_lr(T0), RA
	MOV	gobuf_ret(T0), A0
	MOV	gobuf_ctxt(T0), CTXT
	MOV	ZERO, gobuf_sp(T0)
	MOV	ZERO, gobuf_ret(T0)
	MOV	ZERO, gobuf_lr(T0)
	MOV	ZERO, gobuf_ctxt(T0)
	MOV	gobuf_pc(T0), T0
	JALR	ZERO, T0

// func jmpdefer(fv *funcval, argp uintptr)
// called from deferreturn
// 1. grab stored return address from the caller's frame
// 2. sub 8 bytes to get back to JAL deferreturn
// 3. JMP to fn
TEXT runtime·jmpdefer(SB), NOSPLIT|NOFRAME, $0-16
	MOV	0(X2), RA
	ADD	$-8, RA

	MOV	fv+0(FP), CTXT
	MOV	argp+8(FP), X2
	ADD	$-8, X2
	MOV	0(CTXT), T0
	JALR	ZERO, T0

// func procyield(cycles uint32)
TEXT runtime·procyield(SB),NOSPLIT,$0-0
	RET

// Switch to m->g0's stack, call fn(g).
// Fn must never return. It should gogo(&g->sched)
// to keep running g.

// func mcall(fn func(*g))
TEXT runtime·mcall(SB), NOSPLIT|NOFRAME, $0-8
	// Save caller state in g->sched
	MOV	X2, (g_sched+gobuf_sp)(g)
	MOV	RA, (g_sched+gobuf_pc)(g)
	MOV	ZERO, (g_sched+gobuf_lr)(g)

	// Switch to m->g0 & its stack, call fn.
	MOV	g, T0
	MOV	g_m(g), T1
	MOV	m_g0(T1), g
	CALL	runtime·save_g(SB)
	BNE	g, T0, 2(PC)
	JMP	runtime·badmcall(SB)
	MOV	fn+0(FP), CTXT			// context
	MOV	0(CTXT), T1			// code pointer
	MOV	(g_sched+gobuf_sp)(g), X2	// sp = m->g0->sched.sp
	ADD	$-16, X2
	MOV	T0, 8(X2)
	MOV	ZERO, 0(X2)
	JALR	RA, T1
	JMP	runtime·badmcall2(SB)

// Save state of caller into g->sched,
// but using fake PC from systemstack_switch.
// Must only be called from functions with no locals ($0)
// or else unwinding from systemstack_switch is incorrect.
// Smashes X31.
TEXT gosave_systemstack_switch<>(SB),NOSPLIT|NOFRAME,$0
	MOV	$runtime·systemstack_switch(SB), X31
	ADD	$8, X31	// get past prologue
	MOV	X31, (g_sched+gobuf_pc)(g)
	MOV	X2, (g_sched+gobuf_sp)(g)
	MOV	ZERO, (g_sched+gobuf_lr)(g)
	MOV	ZERO, (g_sched+gobuf_ret)(g)
	// Assert ctxt is zero. See func save.
	MOV	(g_sched+gobuf_ctxt)(g), X31
	BEQ	ZERO, X31, 2(PC)
	CALL	runtime·abort(SB)
	RET

// func asmcgocall(fn, arg unsafe.Pointer) int32
// Call fn(arg) on the scheduler stack,
// aligned appropriately for the gcc ABI.
// See cgocall.go for more details.
TEXT ·asmcgocall(SB),NOSPLIT,$0-20
	MOV	fn+0(FP), X5
	MOV	arg+8(FP), X10

	MOV	X2, X8	// save original stack pointer
	MOV	g, X9

	// Figure out if we need to switch to m->g0 stack.
	// We get called to create new OS threads too, and those
	// come in on the m->g0 stack already.
	MOV	g_m(g), X6
	MOV	m_g0(X6), X7
	BEQ	X7, g, g0

	CALL	gosave_systemstack_switch<>(SB)
	MOV	X7, g
	CALL	runtime·save_g(SB)
	MOV	(g_sched+gobuf_sp)(g), X2

	// Now on a scheduling stack (a pthread-created stack).
g0:
	// Save room for two of our pointers.
	ADD	$-16, X2
	MOV	X9, 0(X2)	// save old g on stack
	MOV	(g_stack+stack_hi)(X9), X9
	SUB	X8, X9, X8
	MOV	X8, 8(X2)	// save depth in old g stack (can't just save SP, as stack might be copied during a callback)

	JALR	RA, (X5)

	// Restore g, stack pointer. X10 is return value.
	MOV	0(X2), g
	CALL	runtime·save_g(SB)
	MOV	(g_stack+stack_hi)(g), X5
	MOV	8(X2), X6
	SUB	X6, X5, X6
	MOV	X6, X2

	MOVW	X10, ret+16(FP)
	RET

// func asminit()
TEXT runtime·asminit(SB),NOSPLIT|NOFRAME,$0-0
	RET

// reflectcall: call a function with the given argument list
// func call(stackArgsType *_type, f *FuncVal, stackArgs *byte, stackArgsSize, stackRetOffset, frameSize uint32, regArgs *abi.RegArgs).
// we don't have variable-sized frames, so we use a small number
// of constant-sized-frame functions to encode a few bits of size in the pc.
// Caution: ugly multiline assembly macros in your future!

#define DISPATCH(NAME,MAXSIZE)	\
	MOV	$MAXSIZE, T1	\
	BLTU	T1, T0, 3(PC)	\
	MOV	$NAME(SB), T2;	\
	JALR	ZERO, T2
// Note: can't just "BR NAME(SB)" - bad inlining results.

// func call(stackArgsType *rtype, fn, stackArgs unsafe.Pointer, stackArgsSize, stackRetOffset, frameSize uint32, regArgs *abi.RegArgs).
TEXT reflect·call(SB), NOSPLIT, $0-0
	JMP	·reflectcall(SB)

// func call(stackArgsType *_type, fn, stackArgs unsafe.Pointer, stackArgsSize, stackRetOffset, frameSize uint32, regArgs *abi.RegArgs).
TEXT ·reflectcall(SB), NOSPLIT|NOFRAME, $0-48
	MOVWU	frameSize+32(FP), T0
	DISPATCH(runtime·call16, 16)
	DISPATCH(runtime·call32, 32)
	DISPATCH(runtime·call64, 64)
	DISPATCH(runtime·call128, 128)
	DISPATCH(runtime·call256, 256)
	DISPATCH(runtime·call512, 512)
	DISPATCH(runtime·call1024, 1024)
	DISPATCH(runtime·call2048, 2048)
	DISPATCH(runtime·call4096, 4096)
	DISPATCH(runtime·call8192, 8192)
	DISPATCH(runtime·call16384, 16384)
	DISPATCH(runtime·call32768, 32768)
	DISPATCH(runtime·call65536, 65536)
	DISPATCH(runtime·call131072, 131072)
	DISPATCH(runtime·call262144, 262144)
	DISPATCH(runtime·call524288, 524288)
	DISPATCH(runtime·call1048576, 1048576)
	DISPATCH(runtime·call2097152, 2097152)
	DISPATCH(runtime·call4194304, 4194304)
	DISPATCH(runtime·call8388608, 8388608)
	DISPATCH(runtime·call16777216, 16777216)
	DISPATCH(runtime·call33554432, 33554432)
	DISPATCH(runtime·call67108864, 67108864)
	DISPATCH(runtime·call134217728, 134217728)
	DISPATCH(runtime·call268435456, 268435456)
	DISPATCH(runtime·call536870912, 536870912)
	DISPATCH(runtime·call1073741824, 1073741824)
	MOV	$runtime·badreflectcall(SB), T2
	JALR	ZERO, T2

#define CALLFN(NAME,MAXSIZE)			\
TEXT NAME(SB), WRAPPER, $MAXSIZE-48;		\
	NO_LOCAL_POINTERS;			\
	/* copy arguments to stack */		\
	MOV	stackArgs+16(FP), A1;			\
	MOVWU	stackArgsSize+24(FP), A2;		\
	MOV	X2, A3;				\
	ADD	$8, A3;				\
	ADD	A3, A2;				\
	BEQ	A3, A2, 6(PC);			\
	MOVBU	(A1), A4;			\
	ADD	$1, A1;				\
	MOVB	A4, (A3);			\
	ADD	$1, A3;				\
	JMP	-5(PC);				\
	/* call function */			\
	MOV	f+8(FP), CTXT;			\
	MOV	(CTXT), A4;			\
	PCDATA  $PCDATA_StackMapIndex, $0;	\
	JALR	RA, A4;				\
	/* copy return values back */		\
	MOV	stackArgsType+0(FP), A5;		\
	MOV	stackArgs+16(FP), A1;			\
	MOVWU	stackArgsSize+24(FP), A2;			\
	MOVWU	stackRetOffset+28(FP), A4;		\
	ADD	$8, X2, A3;			\
	ADD	A4, A3; 			\
	ADD	A4, A1;				\
	SUB	A4, A2;				\
	CALL	callRet<>(SB);			\
	RET

// callRet copies return values back at the end of call*. This is a
// separate function so it can allocate stack space for the arguments
// to reflectcallmove. It does not follow the Go ABI; it expects its
// arguments in registers.
TEXT callRet<>(SB), NOSPLIT, $40-0
	MOV	A5, 8(X2)
	MOV	A1, 16(X2)
	MOV	A3, 24(X2)
	MOV	A2, 32(X2)
	MOV	ZERO, 40(X2)
	CALL	runtime·reflectcallmove(SB)
	RET

CALLFN(·call16, 16)
CALLFN(·call32, 32)
CALLFN(·call64, 64)
CALLFN(·call128, 128)
CALLFN(·call256, 256)
CALLFN(·call512, 512)
CALLFN(·call1024, 1024)
CALLFN(·call2048, 2048)
CALLFN(·call4096, 4096)
CALLFN(·call8192, 8192)
CALLFN(·call16384, 16384)
CALLFN(·call32768, 32768)
CALLFN(·call65536, 65536)
CALLFN(·call131072, 131072)
CALLFN(·call262144, 262144)
CALLFN(·call524288, 524288)
CALLFN(·call1048576, 1048576)
CALLFN(·call2097152, 2097152)
CALLFN(·call4194304, 4194304)
CALLFN(·call8388608, 8388608)
CALLFN(·call16777216, 16777216)
CALLFN(·call33554432, 33554432)
CALLFN(·call67108864, 67108864)
CALLFN(·call134217728, 134217728)
CALLFN(·call268435456, 268435456)
CALLFN(·call536870912, 536870912)
CALLFN(·call1073741824, 1073741824)

// Called from cgo wrappers, this function returns g->m->curg.stack.hi.
// Must obey the gcc calling convention.
TEXT _cgo_topofstack(SB),NOSPLIT,$8
	// g (X27) and REG_TMP (X31) might be clobbered by load_g.
	// X27 is callee-save in the gcc calling convention, so save it.
	MOV	g, savedX27-8(SP)

	CALL	runtime·load_g(SB)
	MOV	g_m(g), X5
	MOV	m_curg(X5), X5
	MOV	(g_stack+stack_hi)(X5), X10 // return value in X10

	MOV	savedX27-8(SP), g
	RET

// func goexit(neverCallThisFunction)
// The top-most function running on a goroutine
// returns to goexit+PCQuantum.
TEXT runtime·goexit(SB),NOSPLIT|NOFRAME|TOPFRAME,$0-0
	MOV	ZERO, ZERO	// NOP
	JMP	runtime·goexit1(SB)	// does not return
	// traceback from goexit1 must hit code range of goexit
	MOV	ZERO, ZERO	// NOP

// func cgocallback(fn, frame unsafe.Pointer, ctxt uintptr)
// See cgocall.go for more details.
TEXT ·cgocallback(SB),NOSPLIT,$24-24
	NO_LOCAL_POINTERS

	// Load m and g from thread-local storage.
	MOVBU	runtime·iscgo(SB), X5
	BEQ	ZERO, X5, nocgo
	CALL	runtime·load_g(SB)
nocgo:

	// If g is nil, Go did not create the current thread.
	// Call needm to obtain one for temporary use.
	// In this case, we're running on the thread stack, so there's
	// lots of space, but the linker doesn't know. Hide the call from
	// the linker analysis by using an indirect call.
	BEQ	ZERO, g, needm

	MOV	g_m(g), X5
	MOV	X5, savedm-8(SP)
	JMP	havem

needm:
	MOV	g, savedm-8(SP) // g is zero, so is m.
	MOV	$runtime·needm(SB), X6
	JALR	RA, X6

	// Set m->sched.sp = SP, so that if a panic happens
	// during the function we are about to execute, it will
	// have a valid SP to run on the g0 stack.
	// The next few lines (after the havem label)
	// will save this SP onto the stack and then write
	// the same SP back to m->sched.sp. That seems redundant,
	// but if an unrecovered panic happens, unwindm will
	// restore the g->sched.sp from the stack location
	// and then systemstack will try to use it. If we don't set it here,
	// that restored SP will be uninitialized (typically 0) and
	// will not be usable.
	MOV	g_m(g), X5
	MOV	m_g0(X5), X6
	MOV	X2, (g_sched+gobuf_sp)(X6)

havem:
	// Now there's a valid m, and we're running on its m->g0.
	// Save current m->g0->sched.sp on stack and then set it to SP.
	// Save current sp in m->g0->sched.sp in preparation for
	// switch back to m->curg stack.
	// NOTE: unwindm knows that the saved g->sched.sp is at 8(X2) aka savedsp-24(SP).
	MOV	m_g0(X5), X6
	MOV	(g_sched+gobuf_sp)(X6), X7
	MOV	X7, savedsp-24(SP)	// must match frame size
	MOV	X2, (g_sched+gobuf_sp)(X6)

	// Switch to m->curg stack and call runtime.cgocallbackg.
	// Because we are taking over the execution of m->curg
	// but *not* resuming what had been running, we need to
	// save that information (m->curg->sched) so we can restore it.
	// We can restore m->curg->sched.sp easily, because calling
	// runtime.cgocallbackg leaves SP unchanged upon return.
	// To save m->curg->sched.pc, we push it onto the curg stack and
	// open a frame the same size as cgocallback's g0 frame.
	// Once we switch to the curg stack, the pushed PC will appear
	// to be the return PC of cgocallback, so that the traceback
	// will seamlessly trace back into the earlier calls.
	MOV	m_curg(X5), g
	CALL	runtime·save_g(SB)
	MOV	(g_sched+gobuf_sp)(g), X6 // prepare stack as X6
	MOV	(g_sched+gobuf_pc)(g), X7
	MOV	X7, -(24+8)(X6)		// "saved LR"; must match frame size
	// Gather our arguments into registers.
	MOV	fn+0(FP), X7
	MOV	frame+8(FP), X8
	MOV	ctxt+16(FP), X9
	MOV	$-(24+8)(X6), X2	// switch stack; must match frame size
	MOV	X7, 8(X2)
	MOV	X8, 16(X2)
	MOV	X9, 24(X2)
	CALL	runtime·cgocallbackg(SB)

	// Restore g->sched (== m->curg->sched) from saved values.
	MOV	0(X2), X7
	MOV	X7, (g_sched+gobuf_pc)(g)
	MOV	$(24+8)(X2), X6		// must match frame size
	MOV	X6, (g_sched+gobuf_sp)(g)

	// Switch back to m->g0's stack and restore m->g0->sched.sp.
	// (Unlike m->curg, the g0 goroutine never uses sched.pc,
	// so we do not have to restore it.)
	MOV	g_m(g), X5
	MOV	m_g0(X5), g
	CALL	runtime·save_g(SB)
	MOV	(g_sched+gobuf_sp)(g), X2
	MOV	savedsp-24(SP), X6	// must match frame size
	MOV	X6, (g_sched+gobuf_sp)(g)

	// If the m on entry was nil, we called needm above to borrow an m
	// for the duration of the call. Since the call is over, return it with dropm.
	MOV	savedm-8(SP), X5
	BNE	ZERO, X5, droppedm
	MOV	$runtime·dropm(SB), X6
	JALR	RA, X6
droppedm:

	// Done!
	RET

TEXT runtime·breakpoint(SB),NOSPLIT|NOFRAME,$0-0
	EBREAK
	RET

TEXT runtime·abort(SB),NOSPLIT|NOFRAME,$0-0
	EBREAK
	RET

// void setg(G*); set g. for use by needm.
TEXT runtime·setg(SB), NOSPLIT, $0-8
	MOV	gg+0(FP), g
	// This only happens if iscgo, so jump straight to save_g
	CALL	runtime·save_g(SB)
	RET

TEXT ·checkASM(SB),NOSPLIT,$0-1
	MOV	$1, T0
	MOV	T0, ret+0(FP)
	RET

// gcWriteBarrier performs a heap pointer write and informs the GC.
//
// gcWriteBarrier does NOT follow the Go ABI. It takes two arguments:
// - T0 is the destination of the write
// - T1 is the value being written at T0.
// It clobbers R30 (the linker temp register - REG_TMP).
// The act of CALLing gcWriteBarrier will clobber RA (LR).
// It does not clobber any other general-purpose registers,
// but may clobber others (e.g., floating point registers).
TEXT runtime·gcWriteBarrier(SB),NOSPLIT,$216
	// Save the registers clobbered by the fast path.
	MOV	A0, 25*8(X2)
	MOV	A1, 26*8(X2)
	MOV	g_m(g), A0
	MOV	m_p(A0), A0
	MOV	(p_wbBuf+wbBuf_next)(A0), A1
	// Increment wbBuf.next position.
	ADD	$16, A1
	MOV	A1, (p_wbBuf+wbBuf_next)(A0)
	MOV	(p_wbBuf+wbBuf_end)(A0), A0
	MOV	A0, T6		// T6 is linker temp register (REG_TMP)
	// Record the write.
	MOV	T1, -16(A1)	// Record value
	MOV	(T0), A0	// TODO: This turns bad writes into bad reads.
	MOV	A0, -8(A1)	// Record *slot
	// Is the buffer full?
	BEQ	A1, T6, flush
ret:
	MOV	25*8(X2), A0
	MOV	26*8(X2), A1
	// Do the write.
	MOV	T1, (T0)
	RET

flush:
	// Save all general purpose registers since these could be
	// clobbered by wbBufFlush and were not saved by the caller.
	MOV	T0, 1*8(X2)	// Also first argument to wbBufFlush
	MOV	T1, 2*8(X2)	// Also second argument to wbBufFlush
	// X0 is zero register
	// X1 is LR, saved by prologue
	// X2 is SP
	MOV	X3, 3*8(X2)
	// X4 is TP
	// X5 is first arg to wbBufFlush (T0)
	// X6 is second arg to wbBufFlush (T1)
	MOV	X7, 4*8(X2)
	MOV	X8, 5*8(X2)
	MOV	X9, 6*8(X2)
	// X10 already saved (A0)
	// X11 already saved (A1)
	MOV	X12, 7*8(X2)
	MOV	X13, 8*8(X2)
	MOV	X14, 9*8(X2)
	MOV	X15, 10*8(X2)
	MOV	X16, 11*8(X2)
	MOV	X17, 12*8(X2)
	MOV	X18, 13*8(X2)
	MOV	X19, 14*8(X2)
	MOV	X20, 15*8(X2)
	MOV	X21, 16*8(X2)
	MOV	X22, 17*8(X2)
	MOV	X23, 18*8(X2)
	MOV	X24, 19*8(X2)
	MOV	X25, 20*8(X2)
	MOV	X26, 21*8(X2)
	// X27 is g.
	MOV	X28, 22*8(X2)
	MOV	X29, 23*8(X2)
	MOV	X30, 24*8(X2)
	// X31 is tmp register.

	// This takes arguments T0 and T1.
	CALL	runtime·wbBufFlush(SB)

	MOV	1*8(X2), T0
	MOV	2*8(X2), T1
	MOV	3*8(X2), X3
	MOV	4*8(X2), X7
	MOV	5*8(X2), X8
	MOV	6*8(X2), X9
	MOV	7*8(X2), X12
	MOV	8*8(X2), X13
	MOV	9*8(X2), X14
	MOV	10*8(X2), X15
	MOV	11*8(X2), X16
	MOV	12*8(X2), X17
	MOV	13*8(X2), X18
	MOV	14*8(X2), X19
	MOV	15*8(X2), X20
	MOV	16*8(X2), X21
	MOV	17*8(X2), X22
	MOV	18*8(X2), X23
	MOV	19*8(X2), X24
	MOV	20*8(X2), X25
	MOV	21*8(X2), X26
	MOV	22*8(X2), X28
	MOV	23*8(X2), X29
	MOV	24*8(X2), X30

	JMP	ret

// Note: these functions use a special calling convention to save generated code space.
// Arguments are passed in registers, but the space for those arguments are allocated
// in the caller's stack frame. These stubs write the args into that stack space and
// then tail call to the corresponding runtime handler.
// The tail call makes these stubs disappear in backtraces.
TEXT runtime·panicIndex(SB),NOSPLIT,$0-16
	MOV	T0, x+0(FP)
	MOV	T1, y+8(FP)
	JMP	runtime·goPanicIndex(SB)
TEXT runtime·panicIndexU(SB),NOSPLIT,$0-16
	MOV	T0, x+0(FP)
	MOV	T1, y+8(FP)
	JMP	runtime·goPanicIndexU(SB)
TEXT runtime·panicSliceAlen(SB),NOSPLIT,$0-16
	MOV	T1, x+0(FP)
	MOV	T2, y+8(FP)
	JMP	runtime·goPanicSliceAlen(SB)
TEXT runtime·panicSliceAlenU(SB),NOSPLIT,$0-16
	MOV	T1, x+0(FP)
	MOV	T2, y+8(FP)
	JMP	runtime·goPanicSliceAlenU(SB)
TEXT runtime·panicSliceAcap(SB),NOSPLIT,$0-16
	MOV	T1, x+0(FP)
	MOV	T2, y+8(FP)
	JMP	runtime·goPanicSliceAcap(SB)
TEXT runtime·panicSliceAcapU(SB),NOSPLIT,$0-16
	MOV	T1, x+0(FP)
	MOV	T2, y+8(FP)
	JMP	runtime·goPanicSliceAcapU(SB)
TEXT runtime·panicSliceB(SB),NOSPLIT,$0-16
	MOV	T0, x+0(FP)
	MOV	T1, y+8(FP)
	JMP	runtime·goPanicSliceB(SB)
TEXT runtime·panicSliceBU(SB),NOSPLIT,$0-16
	MOV	T0, x+0(FP)
	MOV	T1, y+8(FP)
	JMP	runtime·goPanicSliceBU(SB)
TEXT runtime·panicSlice3Alen(SB),NOSPLIT,$0-16
	MOV	T2, x+0(FP)
	MOV	T3, y+8(FP)
	JMP	runtime·goPanicSlice3Alen(SB)
TEXT runtime·panicSlice3AlenU(SB),NOSPLIT,$0-16
	MOV	T2, x+0(FP)
	MOV	T3, y+8(FP)
	JMP	runtime·goPanicSlice3AlenU(SB)
TEXT runtime·panicSlice3Acap(SB),NOSPLIT,$0-16
	MOV	T2, x+0(FP)
	MOV	T3, y+8(FP)
	JMP	runtime·goPanicSlice3Acap(SB)
TEXT runtime·panicSlice3AcapU(SB),NOSPLIT,$0-16
	MOV	T2, x+0(FP)
	MOV	T3, y+8(FP)
	JMP	runtime·goPanicSlice3AcapU(SB)
TEXT runtime·panicSlice3B(SB),NOSPLIT,$0-16
	MOV	T1, x+0(FP)
	MOV	T2, y+8(FP)
	JMP	runtime·goPanicSlice3B(SB)
TEXT runtime·panicSlice3BU(SB),NOSPLIT,$0-16
	MOV	T1, x+0(FP)
	MOV	T2, y+8(FP)
	JMP	runtime·goPanicSlice3BU(SB)
TEXT runtime·panicSlice3C(SB),NOSPLIT,$0-16
	MOV	T0, x+0(FP)
	MOV	T1, y+8(FP)
	JMP	runtime·goPanicSlice3C(SB)
TEXT runtime·panicSlice3CU(SB),NOSPLIT,$0-16
	MOV	T0, x+0(FP)
	MOV	T1, y+8(FP)
	JMP	runtime·goPanicSlice3CU(SB)
TEXT runtime·panicSliceConvert(SB),NOSPLIT,$0-16
	MOV	T2, x+0(FP)
	MOV	T3, y+8(FP)
	JMP	runtime·goPanicSliceConvert(SB)

DATA	runtime·mainPC+0(SB)/8,$runtime·main(SB)
GLOBL	runtime·mainPC(SB),RODATA,$8
