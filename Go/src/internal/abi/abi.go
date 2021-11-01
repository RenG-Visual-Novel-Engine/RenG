// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package abi

import "unsafe"

// RegArgs is a struct that has space for each argument
// and return value register on the current architecture.
//
// Assembly code knows the layout of the first two fields
// of RegArgs.
//
// RegArgs also contains additional space to hold pointers
// when it may not be safe to keep them only in the integer
// register space otherwise.
type RegArgs struct {
	Ints   [IntArgRegs]uintptr  // untyped integer registers
	Floats [FloatArgRegs]uint64 // untyped float registers

	// Fields above this point are known to assembly.

	// Ptrs is a space that duplicates Ints but with pointer type,
	// used to make pointers passed or returned  in registers
	// visible to the GC by making the type unsafe.Pointer.
	Ptrs [IntArgRegs]unsafe.Pointer

	// ReturnIsPtr is a bitmap that indicates which registers
	// contain or will contain pointers on the return path from
	// a reflectcall. The i'th bit indicates whether the i'th
	// register contains or will contain a valid Go pointer.
	ReturnIsPtr IntArgRegBitmap
}

// IntArgRegBitmap is a bitmap large enough to hold one bit per
// integer argument/return register.
type IntArgRegBitmap [(IntArgRegs + 7) / 8]uint8

// Set sets the i'th bit of the bitmap to 1.
func (b *IntArgRegBitmap) Set(i int) {
	b[i/8] |= uint8(1) << (i % 8)
}

// Get returns whether the i'th bit of the bitmap is set.
//
// nosplit because it's called in extremely sensitive contexts, like
// on the reflectcall return path.
//
//go:nosplit
func (b *IntArgRegBitmap) Get(i int) bool {
	return b[i/8]&(uint8(1)<<(i%8)) != 0
}

// FuncPC* intrinsics.
//
// CAREFUL: In programs with plugins, FuncPC* can return different values
// for the same function (because there are actually multiple copies of
// the same function in the address space). To be safe, don't use the
// results of this function in any == expression. It is only safe to
// use the result as an address at which to start executing code.

// FuncPCABI0 returns the entry PC of the function f, which must be a
// direct reference of a function defined as ABI0. Otherwise it is a
// compile-time error.
//
// Implemented as a compile intrinsic.
func FuncPCABI0(f interface{}) uintptr

// FuncPCABIInternal returns the entry PC of the function f. If f is a
// direct reference of a function, it must be defined as ABIInternal.
// Otherwise it is a compile-time error. If f is not a direct reference
// of a defined function, it assumes that f is a func value. Otherwise
// the behavior is undefined.
//
// Implemented as a compile intrinsic.
func FuncPCABIInternal(f interface{}) uintptr
