// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//
// System calls and other sys.stuff for AMD64, Linux
//

#include "go_asm.h"
#include "go_tls.h"
#include "textflag.h"
#include "cgo/abi_amd64.h"

#define AT_FDCWD -100

#define SYS_read		0
#define SYS_write		1
#define SYS_close		3
#define SYS_mmap		9
#define SYS_munmap		11
#define SYS_brk 		12
#define SYS_rt_sigaction	13
#define SYS_rt_sigprocmask	14
#define SYS_rt_sigreturn	15
#define SYS_pipe		22
#define SYS_sched_yield 	24
#define SYS_mincore		27
#define SYS_madvise		28
#define SYS_nanosleep		35
#define SYS_setittimer		38
#define SYS_getpid		39
#define SYS_socket		41
#define SYS_connect		42
#define SYS_clone		56
#define SYS_exit		60
#define SYS_kill		62
#define SYS_fcntl		72
#define SYS_sigaltstack 	131
#define SYS_arch_prctl		158
#define SYS_gettid		186
#define SYS_futex		202
#define SYS_sched_getaffinity	204
#define SYS_epoll_create	213
#define SYS_clock_gettime	228
#define SYS_exit_group		231
#define SYS_epoll_ctl		233
#define SYS_tgkill		234
#define SYS_openat		257
#define SYS_faccessat		269
#define SYS_epoll_pwait		281
#define SYS_epoll_create1	291
#define SYS_pipe2		293

TEXT runtime·exit(SB),NOSPLIT,$0-4
	MOVL	code+0(FP), DI
	MOVL	$SYS_exit_group, AX
	SYSCALL
	RET

// func exitThread(wait *uint32)
TEXT runtime·exitThread(SB),NOSPLIT,$0-8
	MOVQ	wait+0(FP), AX
	// We're done using the stack.
	MOVL	$0, (AX)
	MOVL	$0, DI	// exit code
	MOVL	$SYS_exit, AX
	SYSCALL
	// We may not even have a stack any more.
	INT	$3
	JMP	0(PC)

TEXT runtime·open(SB),NOSPLIT,$0-20
	// This uses openat instead of open, because Android O blocks open.
	MOVL	$AT_FDCWD, DI // AT_FDCWD, so this acts like open
	MOVQ	name+0(FP), SI
	MOVL	mode+8(FP), DX
	MOVL	perm+12(FP), R10
	MOVL	$SYS_openat, AX
	SYSCALL
	CMPQ	AX, $0xfffffffffffff001
	JLS	2(PC)
	MOVL	$-1, AX
	MOVL	AX, ret+16(FP)
	RET

TEXT runtime·closefd(SB),NOSPLIT,$0-12
	MOVL	fd+0(FP), DI
	MOVL	$SYS_close, AX
	SYSCALL
	CMPQ	AX, $0xfffffffffffff001
	JLS	2(PC)
	MOVL	$-1, AX
	MOVL	AX, ret+8(FP)
	RET

TEXT runtime·write1(SB),NOSPLIT,$0-28
	MOVQ	fd+0(FP), DI
	MOVQ	p+8(FP), SI
	MOVL	n+16(FP), DX
	MOVL	$SYS_write, AX
	SYSCALL
	MOVL	AX, ret+24(FP)
	RET

TEXT runtime·read(SB),NOSPLIT,$0-28
	MOVL	fd+0(FP), DI
	MOVQ	p+8(FP), SI
	MOVL	n+16(FP), DX
	MOVL	$SYS_read, AX
	SYSCALL
	MOVL	AX, ret+24(FP)
	RET

// func pipe() (r, w int32, errno int32)
TEXT runtime·pipe(SB),NOSPLIT,$0-12
	LEAQ	r+0(FP), DI
	MOVL	$SYS_pipe, AX
	SYSCALL
	MOVL	AX, errno+8(FP)
	RET

// func pipe2(flags int32) (r, w int32, errno int32)
TEXT runtime·pipe2(SB),NOSPLIT,$0-20
	LEAQ	r+8(FP), DI
	MOVL	flags+0(FP), SI
	MOVL	$SYS_pipe2, AX
	SYSCALL
	MOVL	AX, errno+16(FP)
	RET

TEXT runtime·usleep(SB),NOSPLIT,$16
	MOVL	$0, DX
	MOVL	usec+0(FP), AX
	MOVL	$1000000, CX
	DIVL	CX
	MOVQ	AX, 0(SP)
	MOVL	$1000, AX	// usec to nsec
	MULL	DX
	MOVQ	AX, 8(SP)

	// nanosleep(&ts, 0)
	MOVQ	SP, DI
	MOVL	$0, SI
	MOVL	$SYS_nanosleep, AX
	SYSCALL
	RET

TEXT runtime·gettid(SB),NOSPLIT,$0-4
	MOVL	$SYS_gettid, AX
	SYSCALL
	MOVL	AX, ret+0(FP)
	RET

TEXT runtime·raise(SB),NOSPLIT,$0
	MOVL	$SYS_getpid, AX
	SYSCALL
	MOVL	AX, R12
	MOVL	$SYS_gettid, AX
	SYSCALL
	MOVL	AX, SI	// arg 2 tid
	MOVL	R12, DI	// arg 1 pid
	MOVL	sig+0(FP), DX	// arg 3
	MOVL	$SYS_tgkill, AX
	SYSCALL
	RET

TEXT runtime·raiseproc(SB),NOSPLIT,$0
	MOVL	$SYS_getpid, AX
	SYSCALL
	MOVL	AX, DI	// arg 1 pid
	MOVL	sig+0(FP), SI	// arg 2
	MOVL	$SYS_kill, AX
	SYSCALL
	RET

TEXT ·getpid(SB),NOSPLIT,$0-8
	MOVL	$SYS_getpid, AX
	SYSCALL
	MOVQ	AX, ret+0(FP)
	RET

TEXT ·tgkill(SB),NOSPLIT,$0
	MOVQ	tgid+0(FP), DI
	MOVQ	tid+8(FP), SI
	MOVQ	sig+16(FP), DX
	MOVL	$SYS_tgkill, AX
	SYSCALL
	RET

TEXT runtime·setitimer(SB),NOSPLIT,$0-24
	MOVL	mode+0(FP), DI
	MOVQ	new+8(FP), SI
	MOVQ	old+16(FP), DX
	MOVL	$SYS_setittimer, AX
	SYSCALL
	RET

TEXT runtime·mincore(SB),NOSPLIT,$0-28
	MOVQ	addr+0(FP), DI
	MOVQ	n+8(FP), SI
	MOVQ	dst+16(FP), DX
	MOVL	$SYS_mincore, AX
	SYSCALL
	MOVL	AX, ret+24(FP)
	RET

// func nanotime1() int64
TEXT runtime·nanotime1(SB),NOSPLIT,$16-8
	// We don't know how much stack space the VDSO code will need,
	// so switch to g0.
	// In particular, a kernel configured with CONFIG_OPTIMIZE_INLINING=n
	// and hardening can use a full page of stack space in gettime_sym
	// due to stack probes inserted to avoid stack/heap collisions.
	// See issue #20427.

	MOVQ	SP, R12	// Save old SP; R12 unchanged by C code.

#ifdef GOEXPERIMENT_regabig
	MOVQ	g_m(R14), BX // BX unchanged by C code.
#else
	get_tls(CX)
	MOVQ	g(CX), AX
	MOVQ	g_m(AX), BX // BX unchanged by C code.
#endif

	// Set vdsoPC and vdsoSP for SIGPROF traceback.
	// Save the old values on stack and restore them on exit,
	// so this function is reentrant.
	MOVQ	m_vdsoPC(BX), CX
	MOVQ	m_vdsoSP(BX), DX
	MOVQ	CX, 0(SP)
	MOVQ	DX, 8(SP)

	LEAQ	ret+0(FP), DX
	MOVQ	-8(DX), CX
	MOVQ	CX, m_vdsoPC(BX)
	MOVQ	DX, m_vdsoSP(BX)

#ifdef GOEXPERIMENT_regabig
	CMPQ	R14, m_curg(BX)	// Only switch if on curg.
#else
	CMPQ	AX, m_curg(BX)	// Only switch if on curg.
#endif
	JNE	noswitch

	MOVQ	m_g0(BX), DX
	MOVQ	(g_sched+gobuf_sp)(DX), SP	// Set SP to g0 stack

noswitch:
	SUBQ	$16, SP		// Space for results
	ANDQ	$~15, SP	// Align for C code

	MOVL	$1, DI // CLOCK_MONOTONIC
	LEAQ	0(SP), SI
	MOVQ	runtime·vdsoClockgettimeSym(SB), AX
	CMPQ	AX, $0
	JEQ	fallback
	CALL	AX
ret:
	MOVQ	0(SP), AX	// sec
	MOVQ	8(SP), DX	// nsec
	MOVQ	R12, SP		// Restore real SP
	// Restore vdsoPC, vdsoSP
	// We don't worry about being signaled between the two stores.
	// If we are not in a signal handler, we'll restore vdsoSP to 0,
	// and no one will care about vdsoPC. If we are in a signal handler,
	// we cannot receive another signal.
	MOVQ	8(SP), CX
	MOVQ	CX, m_vdsoSP(BX)
	MOVQ	0(SP), CX
	MOVQ	CX, m_vdsoPC(BX)
	// sec is in AX, nsec in DX
	// return nsec in AX
	IMULQ	$1000000000, AX
	ADDQ	DX, AX
	MOVQ	AX, ret+0(FP)
	RET
fallback:
	MOVQ	$SYS_clock_gettime, AX
	SYSCALL
	JMP	ret

TEXT runtime·rtsigprocmask(SB),NOSPLIT,$0-28
	MOVL	how+0(FP), DI
	MOVQ	new+8(FP), SI
	MOVQ	old+16(FP), DX
	MOVL	size+24(FP), R10
	MOVL	$SYS_rt_sigprocmask, AX
	SYSCALL
	CMPQ	AX, $0xfffffffffffff001
	JLS	2(PC)
	MOVL	$0xf1, 0xf1  // crash
	RET

TEXT runtime·rt_sigaction(SB),NOSPLIT,$0-36
	MOVQ	sig+0(FP), DI
	MOVQ	new+8(FP), SI
	MOVQ	old+16(FP), DX
	MOVQ	size+24(FP), R10
	MOVL	$SYS_rt_sigaction, AX
	SYSCALL
	MOVL	AX, ret+32(FP)
	RET

// Call the function stored in _cgo_sigaction using the GCC calling convention.
TEXT runtime·callCgoSigaction(SB),NOSPLIT,$16
	MOVQ	sig+0(FP), DI
	MOVQ	new+8(FP), SI
	MOVQ	old+16(FP), DX
	MOVQ	_cgo_sigaction(SB), AX
	MOVQ	SP, BX	// callee-saved
	ANDQ	$~15, SP	// alignment as per amd64 psABI
	CALL	AX
	MOVQ	BX, SP
	MOVL	AX, ret+24(FP)
	RET

TEXT runtime·sigfwd(SB),NOSPLIT,$0-32
	MOVQ	fn+0(FP),    AX
	MOVL	sig+8(FP),   DI
	MOVQ	info+16(FP), SI
	MOVQ	ctx+24(FP),  DX
	PUSHQ	BP
	MOVQ	SP, BP
	ANDQ	$~15, SP     // alignment for x86_64 ABI
	CALL	AX
	MOVQ	BP, SP
	POPQ	BP
	RET

// Defined as ABIInternal since it does not use the stack-based Go ABI.
// Called using C ABI.
TEXT runtime·sigtramp<ABIInternal>(SB),NOSPLIT,$0
	// Transition from C ABI to Go ABI.
	PUSH_REGS_HOST_TO_ABI0()

	// Call into the Go signal handler
	NOP	SP		// disable vet stack checking
        ADJSP   $24
	MOVQ	DI, 0(SP)	// sig
	MOVQ	SI, 8(SP)	// info
	MOVQ	DX, 16(SP)	// ctx
	CALL	·sigtrampgo(SB)
	ADJSP	$-24

        POP_REGS_HOST_TO_ABI0()
	RET

// Used instead of sigtramp in programs that use cgo.
// Arguments from kernel are in DI, SI, DX.
// Defined as ABIInternal since it does not use the stack-based Go ABI.
TEXT runtime·cgoSigtramp<ABIInternal>(SB),NOSPLIT,$0
	// If no traceback function, do usual sigtramp.
	MOVQ	runtime·cgoTraceback(SB), AX
	TESTQ	AX, AX
	JZ	sigtramp

	// If no traceback support function, which means that
	// runtime/cgo was not linked in, do usual sigtramp.
	MOVQ	_cgo_callers(SB), AX
	TESTQ	AX, AX
	JZ	sigtramp

	// Figure out if we are currently in a cgo call.
	// If not, just do usual sigtramp.
	get_tls(CX)
	MOVQ	g(CX),AX
	TESTQ	AX, AX
	JZ	sigtrampnog     // g == nil
	MOVQ	g_m(AX), AX
	TESTQ	AX, AX
	JZ	sigtramp        // g.m == nil
	MOVL	m_ncgo(AX), CX
	TESTL	CX, CX
	JZ	sigtramp        // g.m.ncgo == 0
	MOVQ	m_curg(AX), CX
	TESTQ	CX, CX
	JZ	sigtramp        // g.m.curg == nil
	MOVQ	g_syscallsp(CX), CX
	TESTQ	CX, CX
	JZ	sigtramp        // g.m.curg.syscallsp == 0
	MOVQ	m_cgoCallers(AX), R8
	TESTQ	R8, R8
	JZ	sigtramp        // g.m.cgoCallers == nil
	MOVL	m_cgoCallersUse(AX), CX
	TESTL	CX, CX
	JNZ	sigtramp	// g.m.cgoCallersUse != 0

	// Jump to a function in runtime/cgo.
	// That function, written in C, will call the user's traceback
	// function with proper unwind info, and will then call back here.
	// The first three arguments, and the fifth, are already in registers.
	// Set the two remaining arguments now.
	MOVQ	runtime·cgoTraceback(SB), CX
	MOVQ	$runtime·sigtramp<ABIInternal>(SB), R9
	MOVQ	_cgo_callers(SB), AX
	JMP	AX

sigtramp:
	JMP	runtime·sigtramp<ABIInternal>(SB)

sigtrampnog:
	// Signal arrived on a non-Go thread. If this is SIGPROF, get a
	// stack trace.
	CMPL	DI, $27 // 27 == SIGPROF
	JNZ	sigtramp

	// Lock sigprofCallersUse.
	MOVL	$0, AX
	MOVL	$1, CX
	MOVQ	$runtime·sigprofCallersUse(SB), R11
	LOCK
	CMPXCHGL	CX, 0(R11)
	JNZ	sigtramp  // Skip stack trace if already locked.

	// Jump to the traceback function in runtime/cgo.
	// It will call back to sigprofNonGo, which will ignore the
	// arguments passed in registers.
	// First three arguments to traceback function are in registers already.
	MOVQ	runtime·cgoTraceback(SB), CX
	MOVQ	$runtime·sigprofCallers(SB), R8
	MOVQ	$runtime·sigprofNonGo(SB), R9
	MOVQ	_cgo_callers(SB), AX
	JMP	AX

// For cgo unwinding to work, this function must look precisely like
// the one in glibc.  The glibc source code is:
// https://sourceware.org/git/?p=glibc.git;a=blob;f=sysdeps/unix/sysv/linux/x86_64/sigaction.c
// The code that cares about the precise instructions used is:
// https://gcc.gnu.org/viewcvs/gcc/trunk/libgcc/config/i386/linux-unwind.h?revision=219188&view=markup
// Defined as ABIInternal since it does not use the stack-based Go ABI.
TEXT runtime·sigreturn<ABIInternal>(SB),NOSPLIT,$0
	MOVQ	$SYS_rt_sigreturn, AX
	SYSCALL
	INT $3	// not reached

TEXT runtime·sysMmap(SB),NOSPLIT,$0
	MOVQ	addr+0(FP), DI
	MOVQ	n+8(FP), SI
	MOVL	prot+16(FP), DX
	MOVL	flags+20(FP), R10
	MOVL	fd+24(FP), R8
	MOVL	off+28(FP), R9

	MOVL	$SYS_mmap, AX
	SYSCALL
	CMPQ	AX, $0xfffffffffffff001
	JLS	ok
	NOTQ	AX
	INCQ	AX
	MOVQ	$0, p+32(FP)
	MOVQ	AX, err+40(FP)
	RET
ok:
	MOVQ	AX, p+32(FP)
	MOVQ	$0, err+40(FP)
	RET

// Call the function stored in _cgo_mmap using the GCC calling convention.
// This must be called on the system stack.
TEXT runtime·callCgoMmap(SB),NOSPLIT,$16
	MOVQ	addr+0(FP), DI
	MOVQ	n+8(FP), SI
	MOVL	prot+16(FP), DX
	MOVL	flags+20(FP), CX
	MOVL	fd+24(FP), R8
	MOVL	off+28(FP), R9
	MOVQ	_cgo_mmap(SB), AX
	MOVQ	SP, BX
	ANDQ	$~15, SP	// alignment as per amd64 psABI
	MOVQ	BX, 0(SP)
	CALL	AX
	MOVQ	0(SP), SP
	MOVQ	AX, ret+32(FP)
	RET

TEXT runtime·sysMunmap(SB),NOSPLIT,$0
	MOVQ	addr+0(FP), DI
	MOVQ	n+8(FP), SI
	MOVQ	$SYS_munmap, AX
	SYSCALL
	CMPQ	AX, $0xfffffffffffff001
	JLS	2(PC)
	MOVL	$0xf1, 0xf1  // crash
	RET

// Call the function stored in _cgo_munmap using the GCC calling convention.
// This must be called on the system stack.
TEXT runtime·callCgoMunmap(SB),NOSPLIT,$16-16
	MOVQ	addr+0(FP), DI
	MOVQ	n+8(FP), SI
	MOVQ	_cgo_munmap(SB), AX
	MOVQ	SP, BX
	ANDQ	$~15, SP	// alignment as per amd64 psABI
	MOVQ	BX, 0(SP)
	CALL	AX
	MOVQ	0(SP), SP
	RET

TEXT runtime·madvise(SB),NOSPLIT,$0
	MOVQ	addr+0(FP), DI
	MOVQ	n+8(FP), SI
	MOVL	flags+16(FP), DX
	MOVQ	$SYS_madvise, AX
	SYSCALL
	MOVL	AX, ret+24(FP)
	RET

// int64 futex(int32 *uaddr, int32 op, int32 val,
//	struct timespec *timeout, int32 *uaddr2, int32 val2);
TEXT runtime·futex(SB),NOSPLIT,$0
	MOVQ	addr+0(FP), DI
	MOVL	op+8(FP), SI
	MOVL	val+12(FP), DX
	MOVQ	ts+16(FP), R10
	MOVQ	addr2+24(FP), R8
	MOVL	val3+32(FP), R9
	MOVL	$SYS_futex, AX
	SYSCALL
	MOVL	AX, ret+40(FP)
	RET

// int32 clone(int32 flags, void *stk, M *mp, G *gp, void (*fn)(void));
TEXT runtime·clone(SB),NOSPLIT,$0
	MOVL	flags+0(FP), DI
	MOVQ	stk+8(FP), SI
	MOVQ	$0, DX
	MOVQ	$0, R10
	MOVQ    $0, R8
	// Copy mp, gp, fn off parent stack for use by child.
	// Careful: Linux system call clobbers CX and R11.
	MOVQ	mp+16(FP), R13
	MOVQ	gp+24(FP), R9
	MOVQ	fn+32(FP), R12
	CMPQ	R13, $0    // m
	JEQ	nog1
	CMPQ	R9, $0    // g
	JEQ	nog1
	LEAQ	m_tls(R13), R8
#ifdef GOOS_android
	// Android stores the TLS offset in runtime·tls_g.
	SUBQ	runtime·tls_g(SB), R8
#else
	ADDQ	$8, R8	// ELF wants to use -8(FS)
#endif
	ORQ 	$0x00080000, DI //add flag CLONE_SETTLS(0x00080000) to call clone
nog1:
	MOVL	$SYS_clone, AX
	SYSCALL

	// In parent, return.
	CMPQ	AX, $0
	JEQ	3(PC)
	MOVL	AX, ret+40(FP)
	RET

	// In child, on new stack.
	MOVQ	SI, SP

	// If g or m are nil, skip Go-related setup.
	CMPQ	R13, $0    // m
	JEQ	nog2
	CMPQ	R9, $0    // g
	JEQ	nog2

	// Initialize m->procid to Linux tid
	MOVL	$SYS_gettid, AX
	SYSCALL
	MOVQ	AX, m_procid(R13)

	// In child, set up new stack
	get_tls(CX)
	MOVQ	R13, g_m(R9)
	MOVQ	R9, g(CX)
	MOVQ	R9, R14 // set g register
	CALL	runtime·stackcheck(SB)

nog2:
	// Call fn. This is the PC of an ABI0 function.
	CALL	R12

	// It shouldn't return. If it does, exit that thread.
	MOVL	$111, DI
	MOVL	$SYS_exit, AX
	SYSCALL
	JMP	-3(PC)	// keep exiting

TEXT runtime·sigaltstack(SB),NOSPLIT,$-8
	MOVQ	new+0(FP), DI
	MOVQ	old+8(FP), SI
	MOVQ	$SYS_sigaltstack, AX
	SYSCALL
	CMPQ	AX, $0xfffffffffffff001
	JLS	2(PC)
	MOVL	$0xf1, 0xf1  // crash
	RET

// set tls base to DI
TEXT runtime·settls(SB),NOSPLIT,$32
#ifdef GOOS_android
	// Android stores the TLS offset in runtime·tls_g.
	SUBQ	runtime·tls_g(SB), DI
#else
	ADDQ	$8, DI	// ELF wants to use -8(FS)
#endif
	MOVQ	DI, SI
	MOVQ	$0x1002, DI	// ARCH_SET_FS
	MOVQ	$SYS_arch_prctl, AX
	SYSCALL
	CMPQ	AX, $0xfffffffffffff001
	JLS	2(PC)
	MOVL	$0xf1, 0xf1  // crash
	RET

TEXT runtime·osyield(SB),NOSPLIT,$0
	MOVL	$SYS_sched_yield, AX
	SYSCALL
	RET

TEXT runtime·sched_getaffinity(SB),NOSPLIT,$0
	MOVQ	pid+0(FP), DI
	MOVQ	len+8(FP), SI
	MOVQ	buf+16(FP), DX
	MOVL	$SYS_sched_getaffinity, AX
	SYSCALL
	MOVL	AX, ret+24(FP)
	RET

// int32 runtime·epollcreate(int32 size);
TEXT runtime·epollcreate(SB),NOSPLIT,$0
	MOVL    size+0(FP), DI
	MOVL    $SYS_epoll_create, AX
	SYSCALL
	MOVL	AX, ret+8(FP)
	RET

// int32 runtime·epollcreate1(int32 flags);
TEXT runtime·epollcreate1(SB),NOSPLIT,$0
	MOVL	flags+0(FP), DI
	MOVL	$SYS_epoll_create1, AX
	SYSCALL
	MOVL	AX, ret+8(FP)
	RET

// func epollctl(epfd, op, fd int32, ev *epollEvent) int
TEXT runtime·epollctl(SB),NOSPLIT,$0
	MOVL	epfd+0(FP), DI
	MOVL	op+4(FP), SI
	MOVL	fd+8(FP), DX
	MOVQ	ev+16(FP), R10
	MOVL	$SYS_epoll_ctl, AX
	SYSCALL
	MOVL	AX, ret+24(FP)
	RET

// int32 runtime·epollwait(int32 epfd, EpollEvent *ev, int32 nev, int32 timeout);
TEXT runtime·epollwait(SB),NOSPLIT,$0
	// This uses pwait instead of wait, because Android O blocks wait.
	MOVL	epfd+0(FP), DI
	MOVQ	ev+8(FP), SI
	MOVL	nev+16(FP), DX
	MOVL	timeout+20(FP), R10
	MOVQ	$0, R8
	MOVL	$SYS_epoll_pwait, AX
	SYSCALL
	MOVL	AX, ret+24(FP)
	RET

// void runtime·closeonexec(int32 fd);
TEXT runtime·closeonexec(SB),NOSPLIT,$0
	MOVL    fd+0(FP), DI  // fd
	MOVQ    $2, SI  // F_SETFD
	MOVQ    $1, DX  // FD_CLOEXEC
	MOVL	$SYS_fcntl, AX
	SYSCALL
	RET

// func runtime·setNonblock(int32 fd)
TEXT runtime·setNonblock(SB),NOSPLIT,$0-4
	MOVL    fd+0(FP), DI  // fd
	MOVQ    $3, SI  // F_GETFL
	MOVQ    $0, DX
	MOVL	$SYS_fcntl, AX
	SYSCALL
	MOVL	fd+0(FP), DI // fd
	MOVQ	$4, SI // F_SETFL
	MOVQ	$0x800, DX // O_NONBLOCK
	ORL	AX, DX
	MOVL	$SYS_fcntl, AX
	SYSCALL
	RET

// int access(const char *name, int mode)
TEXT runtime·access(SB),NOSPLIT,$0
	// This uses faccessat instead of access, because Android O blocks access.
	MOVL	$AT_FDCWD, DI // AT_FDCWD, so this acts like access
	MOVQ	name+0(FP), SI
	MOVL	mode+8(FP), DX
	MOVL	$0, R10
	MOVL	$SYS_faccessat, AX
	SYSCALL
	MOVL	AX, ret+16(FP)
	RET

// int connect(int fd, const struct sockaddr *addr, socklen_t addrlen)
TEXT runtime·connect(SB),NOSPLIT,$0-28
	MOVL	fd+0(FP), DI
	MOVQ	addr+8(FP), SI
	MOVL	len+16(FP), DX
	MOVL	$SYS_connect, AX
	SYSCALL
	MOVL	AX, ret+24(FP)
	RET

// int socket(int domain, int type, int protocol)
TEXT runtime·socket(SB),NOSPLIT,$0-20
	MOVL	domain+0(FP), DI
	MOVL	typ+4(FP), SI
	MOVL	prot+8(FP), DX
	MOVL	$SYS_socket, AX
	SYSCALL
	MOVL	AX, ret+16(FP)
	RET

// func sbrk0() uintptr
TEXT runtime·sbrk0(SB),NOSPLIT,$0-8
	// Implemented as brk(NULL).
	MOVQ	$0, DI
	MOVL	$SYS_brk, AX
	SYSCALL
	MOVQ	AX, ret+0(FP)
	RET
