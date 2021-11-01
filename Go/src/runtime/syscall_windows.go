// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	"internal/abi"
	"runtime/internal/sys"
	"unsafe"
)

// cbs stores all registered Go callbacks.
var cbs struct {
	lock  mutex
	ctxt  [cb_max]winCallback
	index map[winCallbackKey]int
	n     int
}

// winCallback records information about a registered Go callback.
type winCallback struct {
	fn     *funcval // Go function
	retPop uintptr  // For 386 cdecl, how many bytes to pop on return
	abiMap abiDesc
}

// abiPartKind is the action an abiPart should take.
type abiPartKind int

const (
	abiPartBad   abiPartKind = iota
	abiPartStack             // Move a value from memory to the stack.
	abiPartReg               // Move a value from memory to a register.
)

// abiPart encodes a step in translating between calling ABIs.
type abiPart struct {
	kind           abiPartKind
	srcStackOffset uintptr
	dstStackOffset uintptr // used if kind == abiPartStack
	dstRegister    int     // used if kind == abiPartReg
	len            uintptr
}

func (a *abiPart) tryMerge(b abiPart) bool {
	if a.kind != abiPartStack || b.kind != abiPartStack {
		return false
	}
	if a.srcStackOffset+a.len == b.srcStackOffset && a.dstStackOffset+a.len == b.dstStackOffset {
		a.len += b.len
		return true
	}
	return false
}

// abiDesc specifies how to translate from a C frame to a Go
// frame. This does not specify how to translate back because
// the result is always a uintptr. If the C ABI is fastcall,
// this assumes the four fastcall registers were first spilled
// to the shadow space.
type abiDesc struct {
	parts []abiPart

	srcStackSize uintptr // stdcall/fastcall stack space tracking
	dstStackSize uintptr // Go stack space used
	dstSpill     uintptr // Extra stack space for argument spill slots
	dstRegisters int     // Go ABI int argument registers used

	// retOffset is the offset of the uintptr-sized result in the Go
	// frame.
	retOffset uintptr
}

func (p *abiDesc) assignArg(t *_type) {
	if t.size > sys.PtrSize {
		// We don't support this right now. In
		// stdcall/cdecl, 64-bit ints and doubles are
		// passed as two words (little endian); and
		// structs are pushed on the stack. In
		// fastcall, arguments larger than the word
		// size are passed by reference. On arm,
		// 8-byte aligned arguments round up to the
		// next even register and can be split across
		// registers and the stack.
		panic("compileCallback: argument size is larger than uintptr")
	}
	if k := t.kind & kindMask; GOARCH != "386" && (k == kindFloat32 || k == kindFloat64) {
		// In fastcall, floating-point arguments in
		// the first four positions are passed in
		// floating-point registers, which we don't
		// currently spill. arm passes floating-point
		// arguments in VFP registers, which we also
		// don't support.
		// So basically we only support 386.
		panic("compileCallback: float arguments not supported")
	}

	if t.size == 0 {
		// The Go ABI aligns for zero-sized types.
		p.dstStackSize = alignUp(p.dstStackSize, uintptr(t.align))
		return
	}

	// In the C ABI, we're already on a word boundary.
	// Also, sub-word-sized fastcall register arguments
	// are stored to the least-significant bytes of the
	// argument word and all supported Windows
	// architectures are little endian, so srcStackOffset
	// is already pointing to the right place for smaller
	// arguments. The same is true on arm.

	oldParts := p.parts
	if p.tryRegAssignArg(t, 0) {
		// Account for spill space.
		//
		// TODO(mknyszek): Remove this when we no longer have
		// caller reserved spill space.
		p.dstSpill = alignUp(p.dstSpill, uintptr(t.align))
		p.dstSpill += t.size
	} else {
		// Register assignment failed.
		// Undo the work and stack assign.
		p.parts = oldParts

		// The Go ABI aligns arguments.
		p.dstStackSize = alignUp(p.dstStackSize, uintptr(t.align))

		// Copy just the size of the argument. Note that this
		// could be a small by-value struct, but C and Go
		// struct layouts are compatible, so we can copy these
		// directly, too.
		part := abiPart{
			kind:           abiPartStack,
			srcStackOffset: p.srcStackSize,
			dstStackOffset: p.dstStackSize,
			len:            t.size,
		}
		// Add this step to the adapter.
		if len(p.parts) == 0 || !p.parts[len(p.parts)-1].tryMerge(part) {
			p.parts = append(p.parts, part)
		}
		// The Go ABI packs arguments.
		p.dstStackSize += t.size
	}

	// cdecl, stdcall, fastcall, and arm pad arguments to word size.
	// TODO(rsc): On arm and arm64 do we need to skip the caller's saved LR?
	p.srcStackSize += sys.PtrSize
}

// tryRegAssignArg tries to register-assign a value of type t.
// If this type is nested in an aggregate type, then offset is the
// offset of this type within its parent type.
// Assumes t.size <= sys.PtrSize and t.size != 0.
//
// Returns whether the assignment succeeded.
func (p *abiDesc) tryRegAssignArg(t *_type, offset uintptr) bool {
	switch k := t.kind & kindMask; k {
	case kindBool, kindInt, kindInt8, kindInt16, kindInt32, kindUint, kindUint8, kindUint16, kindUint32, kindUintptr, kindPtr, kindUnsafePointer:
		// Assign a register for all these types.
		return p.assignReg(t.size, offset)
	case kindInt64, kindUint64:
		// Only register-assign if the registers are big enough.
		if sys.PtrSize == 8 {
			return p.assignReg(t.size, offset)
		}
	case kindArray:
		at := (*arraytype)(unsafe.Pointer(t))
		if at.len == 1 {
			return p.tryRegAssignArg(at.elem, offset)
		}
	case kindStruct:
		st := (*structtype)(unsafe.Pointer(t))
		for i := range st.fields {
			f := &st.fields[i]
			if !p.tryRegAssignArg(f.typ, offset+f.offset()) {
				return false
			}
		}
		return true
	}
	// Pointer-sized types such as maps and channels are currently
	// not supported.
	panic("compileCallabck: type " + t.string() + " is currently not supported for use in system callbacks")
}

// assignReg attempts to assign a single register for an
// argument with the given size, at the given offset into the
// value in the C ABI space.
//
// Returns whether the assignment was successful.
func (p *abiDesc) assignReg(size, offset uintptr) bool {
	if p.dstRegisters >= intArgRegs {
		return false
	}
	p.parts = append(p.parts, abiPart{
		kind:           abiPartReg,
		srcStackOffset: p.srcStackSize + offset,
		dstRegister:    p.dstRegisters,
		len:            size,
	})
	p.dstRegisters++
	return true
}

type winCallbackKey struct {
	fn    *funcval
	cdecl bool
}

func callbackasm()

// callbackasmAddr returns address of runtime.callbackasm
// function adjusted by i.
// On x86 and amd64, runtime.callbackasm is a series of CALL instructions,
// and we want callback to arrive at
// correspondent call instruction instead of start of
// runtime.callbackasm.
// On ARM, runtime.callbackasm is a series of mov and branch instructions.
// R12 is loaded with the callback index. Each entry is two instructions,
// hence 8 bytes.
func callbackasmAddr(i int) uintptr {
	var entrySize int
	switch GOARCH {
	default:
		panic("unsupported architecture")
	case "386", "amd64":
		entrySize = 5
	case "arm", "arm64":
		// On ARM and ARM64, each entry is a MOV instruction
		// followed by a branch instruction
		entrySize = 8
	}
	return funcPC(callbackasm) + uintptr(i*entrySize)
}

const callbackMaxFrame = 64 * sys.PtrSize

// compileCallback converts a Go function fn into a C function pointer
// that can be passed to Windows APIs.
//
// On 386, if cdecl is true, the returned C function will use the
// cdecl calling convention; otherwise, it will use stdcall. On amd64,
// it always uses fastcall. On arm, it always uses the ARM convention.
//
//go:linkname compileCallback syscall.compileCallback
func compileCallback(fn eface, cdecl bool) (code uintptr) {
	if GOARCH != "386" {
		// cdecl is only meaningful on 386.
		cdecl = false
	}

	if fn._type == nil || (fn._type.kind&kindMask) != kindFunc {
		panic("compileCallback: expected function with one uintptr-sized result")
	}
	ft := (*functype)(unsafe.Pointer(fn._type))

	// Check arguments and construct ABI translation.
	var abiMap abiDesc
	for _, t := range ft.in() {
		abiMap.assignArg(t)
	}
	// The Go ABI aligns the result to the word size. src is
	// already aligned.
	abiMap.dstStackSize = alignUp(abiMap.dstStackSize, sys.PtrSize)
	abiMap.retOffset = abiMap.dstStackSize

	if len(ft.out()) != 1 {
		panic("compileCallback: expected function with one uintptr-sized result")
	}
	if ft.out()[0].size != sys.PtrSize {
		panic("compileCallback: expected function with one uintptr-sized result")
	}
	if k := ft.out()[0].kind & kindMask; k == kindFloat32 || k == kindFloat64 {
		// In cdecl and stdcall, float results are returned in
		// ST(0). In fastcall, they're returned in XMM0.
		// Either way, it's not AX.
		panic("compileCallback: float results not supported")
	}
	if intArgRegs == 0 {
		// Make room for the uintptr-sized result.
		// If there are argument registers, the return value will
		// be passed in the first register.
		abiMap.dstStackSize += sys.PtrSize
	}

	// TODO(mknyszek): Remove dstSpill from this calculation when we no longer have
	// caller reserved spill space.
	frameSize := alignUp(abiMap.dstStackSize, sys.PtrSize)
	frameSize += abiMap.dstSpill
	if frameSize > callbackMaxFrame {
		panic("compileCallback: function argument frame too large")
	}

	// For cdecl, the callee is responsible for popping its
	// arguments from the C stack.
	var retPop uintptr
	if cdecl {
		retPop = abiMap.srcStackSize
	}

	key := winCallbackKey{(*funcval)(fn.data), cdecl}

	lock(&cbs.lock) // We don't unlock this in a defer because this is used from the system stack.

	// Check if this callback is already registered.
	if n, ok := cbs.index[key]; ok {
		unlock(&cbs.lock)
		return callbackasmAddr(n)
	}

	// Register the callback.
	if cbs.index == nil {
		cbs.index = make(map[winCallbackKey]int)
	}
	n := cbs.n
	if n >= len(cbs.ctxt) {
		unlock(&cbs.lock)
		throw("too many callback functions")
	}
	c := winCallback{key.fn, retPop, abiMap}
	cbs.ctxt[n] = c
	cbs.index[key] = n
	cbs.n++

	unlock(&cbs.lock)
	return callbackasmAddr(n)
}

type callbackArgs struct {
	index uintptr
	// args points to the argument block.
	//
	// For cdecl and stdcall, all arguments are on the stack.
	//
	// For fastcall, the trampoline spills register arguments to
	// the reserved spill slots below the stack arguments,
	// resulting in a layout equivalent to stdcall.
	//
	// For arm, the trampoline stores the register arguments just
	// below the stack arguments, so again we can treat it as one
	// big stack arguments frame.
	args unsafe.Pointer
	// Below are out-args from callbackWrap
	result uintptr
	retPop uintptr // For 386 cdecl, how many bytes to pop on return
}

// callbackWrap is called by callbackasm to invoke a registered C callback.
func callbackWrap(a *callbackArgs) {
	c := cbs.ctxt[a.index]
	a.retPop = c.retPop

	// Convert from C to Go ABI.
	var regs abi.RegArgs
	var frame [callbackMaxFrame]byte
	goArgs := unsafe.Pointer(&frame)
	for _, part := range c.abiMap.parts {
		switch part.kind {
		case abiPartStack:
			memmove(add(goArgs, part.dstStackOffset), add(a.args, part.srcStackOffset), part.len)
		case abiPartReg:
			goReg := unsafe.Pointer(&regs.Ints[part.dstRegister])
			memmove(goReg, add(a.args, part.srcStackOffset), part.len)
		default:
			panic("bad ABI description")
		}
	}

	// TODO(mknyszek): Remove this when we no longer have
	// caller reserved spill space.
	frameSize := alignUp(c.abiMap.dstStackSize, sys.PtrSize)
	frameSize += c.abiMap.dstSpill

	// Even though this is copying back results, we can pass a nil
	// type because those results must not require write barriers.
	reflectcall(nil, unsafe.Pointer(c.fn), noescape(goArgs), uint32(c.abiMap.dstStackSize), uint32(c.abiMap.retOffset), uint32(frameSize), &regs)

	// Extract the result.
	//
	// There's always exactly one return value, one pointer in size.
	// If it's on the stack, then we will have reserved space for it
	// at the end of the frame, otherwise it was passed in a register.
	if c.abiMap.dstStackSize != c.abiMap.retOffset {
		a.result = *(*uintptr)(unsafe.Pointer(&frame[c.abiMap.retOffset]))
	} else {
		var zero int
		// On architectures with no registers, Ints[0] would be a compile error,
		// so we use a dynamic index. These architectures will never take this
		// branch, so this won't cause a runtime panic.
		a.result = regs.Ints[zero]
	}
}

const _LOAD_LIBRARY_SEARCH_SYSTEM32 = 0x00000800

// When available, this function will use LoadLibraryEx with the filename
// parameter and the important SEARCH_SYSTEM32 argument. But on systems that
// do not have that option, absoluteFilepath should contain a fallback
// to the full path inside of system32 for use with vanilla LoadLibrary.
//go:linkname syscall_loadsystemlibrary syscall.loadsystemlibrary
//go:nosplit
//go:cgo_unsafe_args
func syscall_loadsystemlibrary(filename *uint16, absoluteFilepath *uint16) (handle, err uintptr) {
	lockOSThread()
	c := &getg().m.syscall

	if useLoadLibraryEx {
		c.fn = getLoadLibraryEx()
		c.n = 3
		args := struct {
			lpFileName *uint16
			hFile      uintptr // always 0
			flags      uint32
		}{filename, 0, _LOAD_LIBRARY_SEARCH_SYSTEM32}
		c.args = uintptr(noescape(unsafe.Pointer(&args)))
	} else {
		c.fn = getLoadLibrary()
		c.n = 1
		c.args = uintptr(noescape(unsafe.Pointer(&absoluteFilepath)))
	}

	cgocall(asmstdcallAddr, unsafe.Pointer(c))
	handle = c.r1
	if handle == 0 {
		err = c.err
	}
	unlockOSThread() // not defer'd after the lockOSThread above to save stack frame size.
	return
}

//go:linkname syscall_loadlibrary syscall.loadlibrary
//go:nosplit
//go:cgo_unsafe_args
func syscall_loadlibrary(filename *uint16) (handle, err uintptr) {
	lockOSThread()
	defer unlockOSThread()
	c := &getg().m.syscall
	c.fn = getLoadLibrary()
	c.n = 1
	c.args = uintptr(noescape(unsafe.Pointer(&filename)))
	cgocall(asmstdcallAddr, unsafe.Pointer(c))
	handle = c.r1
	if handle == 0 {
		err = c.err
	}
	return
}

//go:linkname syscall_getprocaddress syscall.getprocaddress
//go:nosplit
//go:cgo_unsafe_args
func syscall_getprocaddress(handle uintptr, procname *byte) (outhandle, err uintptr) {
	lockOSThread()
	defer unlockOSThread()
	c := &getg().m.syscall
	c.fn = getGetProcAddress()
	c.n = 2
	c.args = uintptr(noescape(unsafe.Pointer(&handle)))
	cgocall(asmstdcallAddr, unsafe.Pointer(c))
	outhandle = c.r1
	if outhandle == 0 {
		err = c.err
	}
	return
}

//go:linkname syscall_Syscall syscall.Syscall
//go:nosplit
//go:cgo_unsafe_args
func syscall_Syscall(fn, nargs, a1, a2, a3 uintptr) (r1, r2, err uintptr) {
	lockOSThread()
	defer unlockOSThread()
	c := &getg().m.syscall
	c.fn = fn
	c.n = nargs
	c.args = uintptr(noescape(unsafe.Pointer(&a1)))
	cgocall(asmstdcallAddr, unsafe.Pointer(c))
	return c.r1, c.r2, c.err
}

//go:linkname syscall_Syscall6 syscall.Syscall6
//go:nosplit
//go:cgo_unsafe_args
func syscall_Syscall6(fn, nargs, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2, err uintptr) {
	lockOSThread()
	defer unlockOSThread()
	c := &getg().m.syscall
	c.fn = fn
	c.n = nargs
	c.args = uintptr(noescape(unsafe.Pointer(&a1)))
	cgocall(asmstdcallAddr, unsafe.Pointer(c))
	return c.r1, c.r2, c.err
}

//go:linkname syscall_Syscall9 syscall.Syscall9
//go:nosplit
//go:cgo_unsafe_args
func syscall_Syscall9(fn, nargs, a1, a2, a3, a4, a5, a6, a7, a8, a9 uintptr) (r1, r2, err uintptr) {
	lockOSThread()
	c := &getg().m.syscall
	c.fn = fn
	c.n = nargs
	c.args = uintptr(noescape(unsafe.Pointer(&a1)))
	cgocall(asmstdcallAddr, unsafe.Pointer(c))
	unlockOSThread()
	return c.r1, c.r2, c.err
}

//go:linkname syscall_Syscall12 syscall.Syscall12
//go:nosplit
//go:cgo_unsafe_args
func syscall_Syscall12(fn, nargs, a1, a2, a3, a4, a5, a6, a7, a8, a9, a10, a11, a12 uintptr) (r1, r2, err uintptr) {
	lockOSThread()
	c := &getg().m.syscall
	c.fn = fn
	c.n = nargs
	c.args = uintptr(noescape(unsafe.Pointer(&a1)))
	cgocall(asmstdcallAddr, unsafe.Pointer(c))
	unlockOSThread()
	return c.r1, c.r2, c.err
}

//go:linkname syscall_Syscall15 syscall.Syscall15
//go:nosplit
//go:cgo_unsafe_args
func syscall_Syscall15(fn, nargs, a1, a2, a3, a4, a5, a6, a7, a8, a9, a10, a11, a12, a13, a14, a15 uintptr) (r1, r2, err uintptr) {
	lockOSThread()
	c := &getg().m.syscall
	c.fn = fn
	c.n = nargs
	c.args = uintptr(noescape(unsafe.Pointer(&a1)))
	cgocall(asmstdcallAddr, unsafe.Pointer(c))
	unlockOSThread()
	return c.r1, c.r2, c.err
}

//go:linkname syscall_Syscall18 syscall.Syscall18
//go:nosplit
//go:cgo_unsafe_args
func syscall_Syscall18(fn, nargs, a1, a2, a3, a4, a5, a6, a7, a8, a9, a10, a11, a12, a13, a14, a15, a16, a17, a18 uintptr) (r1, r2, err uintptr) {
	lockOSThread()
	c := &getg().m.syscall
	c.fn = fn
	c.n = nargs
	c.args = uintptr(noescape(unsafe.Pointer(&a1)))
	cgocall(asmstdcallAddr, unsafe.Pointer(c))
	unlockOSThread()
	return c.r1, c.r2, c.err
}
