// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ssagen

import (
	"bufio"
	"bytes"
	"cmd/compile/internal/abi"
	"fmt"
	"go/constant"
	"html"
	"internal/buildcfg"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"cmd/compile/internal/base"
	"cmd/compile/internal/ir"
	"cmd/compile/internal/liveness"
	"cmd/compile/internal/objw"
	"cmd/compile/internal/reflectdata"
	"cmd/compile/internal/ssa"
	"cmd/compile/internal/staticdata"
	"cmd/compile/internal/typecheck"
	"cmd/compile/internal/types"
	"cmd/internal/obj"
	"cmd/internal/obj/x86"
	"cmd/internal/objabi"
	"cmd/internal/src"
	"cmd/internal/sys"
)

var ssaConfig *ssa.Config
var ssaCaches []ssa.Cache

var ssaDump string     // early copy of $GOSSAFUNC; the func name to dump output for
var ssaDir string      // optional destination for ssa dump file
var ssaDumpStdout bool // whether to dump to stdout
var ssaDumpCFG string  // generate CFGs for these phases
const ssaDumpFile = "ssa.html"

// ssaDumpInlined holds all inlined functions when ssaDump contains a function name.
var ssaDumpInlined []*ir.Func

func DumpInline(fn *ir.Func) {
	if ssaDump != "" && ssaDump == ir.FuncName(fn) {
		ssaDumpInlined = append(ssaDumpInlined, fn)
	}
}

func InitEnv() {
	ssaDump = os.Getenv("GOSSAFUNC")
	ssaDir = os.Getenv("GOSSADIR")
	if ssaDump != "" {
		if strings.HasSuffix(ssaDump, "+") {
			ssaDump = ssaDump[:len(ssaDump)-1]
			ssaDumpStdout = true
		}
		spl := strings.Split(ssaDump, ":")
		if len(spl) > 1 {
			ssaDump = spl[0]
			ssaDumpCFG = spl[1]
		}
	}
}

func InitConfig() {
	types_ := ssa.NewTypes()

	if Arch.SoftFloat {
		softfloatInit()
	}

	// Generate a few pointer types that are uncommon in the frontend but common in the backend.
	// Caching is disabled in the backend, so generating these here avoids allocations.
	_ = types.NewPtr(types.Types[types.TINTER])                             // *interface{}
	_ = types.NewPtr(types.NewPtr(types.Types[types.TSTRING]))              // **string
	_ = types.NewPtr(types.NewSlice(types.Types[types.TINTER]))             // *[]interface{}
	_ = types.NewPtr(types.NewPtr(types.ByteType))                          // **byte
	_ = types.NewPtr(types.NewSlice(types.ByteType))                        // *[]byte
	_ = types.NewPtr(types.NewSlice(types.Types[types.TSTRING]))            // *[]string
	_ = types.NewPtr(types.NewPtr(types.NewPtr(types.Types[types.TUINT8]))) // ***uint8
	_ = types.NewPtr(types.Types[types.TINT16])                             // *int16
	_ = types.NewPtr(types.Types[types.TINT64])                             // *int64
	_ = types.NewPtr(types.ErrorType)                                       // *error
	types.NewPtrCacheEnabled = false
	ssaConfig = ssa.NewConfig(base.Ctxt.Arch.Name, *types_, base.Ctxt, base.Flag.N == 0)
	ssaConfig.SoftFloat = Arch.SoftFloat
	ssaConfig.Race = base.Flag.Race
	ssaCaches = make([]ssa.Cache, base.Flag.LowerC)

	// Set up some runtime functions we'll need to call.
	ir.Syms.AssertE2I = typecheck.LookupRuntimeFunc("assertE2I")
	ir.Syms.AssertE2I2 = typecheck.LookupRuntimeFunc("assertE2I2")
	ir.Syms.AssertI2I = typecheck.LookupRuntimeFunc("assertI2I")
	ir.Syms.AssertI2I2 = typecheck.LookupRuntimeFunc("assertI2I2")
	ir.Syms.Deferproc = typecheck.LookupRuntimeFunc("deferproc")
	ir.Syms.DeferprocStack = typecheck.LookupRuntimeFunc("deferprocStack")
	ir.Syms.Deferreturn = typecheck.LookupRuntimeFunc("deferreturn")
	ir.Syms.Duffcopy = typecheck.LookupRuntimeFunc("duffcopy")
	ir.Syms.Duffzero = typecheck.LookupRuntimeFunc("duffzero")
	ir.Syms.GCWriteBarrier = typecheck.LookupRuntimeFunc("gcWriteBarrier")
	ir.Syms.Goschedguarded = typecheck.LookupRuntimeFunc("goschedguarded")
	ir.Syms.Growslice = typecheck.LookupRuntimeFunc("growslice")
	ir.Syms.Msanread = typecheck.LookupRuntimeFunc("msanread")
	ir.Syms.Msanwrite = typecheck.LookupRuntimeFunc("msanwrite")
	ir.Syms.Msanmove = typecheck.LookupRuntimeFunc("msanmove")
	ir.Syms.Newobject = typecheck.LookupRuntimeFunc("newobject")
	ir.Syms.Newproc = typecheck.LookupRuntimeFunc("newproc")
	ir.Syms.Panicdivide = typecheck.LookupRuntimeFunc("panicdivide")
	ir.Syms.PanicdottypeE = typecheck.LookupRuntimeFunc("panicdottypeE")
	ir.Syms.PanicdottypeI = typecheck.LookupRuntimeFunc("panicdottypeI")
	ir.Syms.Panicnildottype = typecheck.LookupRuntimeFunc("panicnildottype")
	ir.Syms.Panicoverflow = typecheck.LookupRuntimeFunc("panicoverflow")
	ir.Syms.Panicshift = typecheck.LookupRuntimeFunc("panicshift")
	ir.Syms.Raceread = typecheck.LookupRuntimeFunc("raceread")
	ir.Syms.Racereadrange = typecheck.LookupRuntimeFunc("racereadrange")
	ir.Syms.Racewrite = typecheck.LookupRuntimeFunc("racewrite")
	ir.Syms.Racewriterange = typecheck.LookupRuntimeFunc("racewriterange")
	ir.Syms.X86HasPOPCNT = typecheck.LookupRuntimeVar("x86HasPOPCNT")       // bool
	ir.Syms.X86HasSSE41 = typecheck.LookupRuntimeVar("x86HasSSE41")         // bool
	ir.Syms.X86HasFMA = typecheck.LookupRuntimeVar("x86HasFMA")             // bool
	ir.Syms.ARMHasVFPv4 = typecheck.LookupRuntimeVar("armHasVFPv4")         // bool
	ir.Syms.ARM64HasATOMICS = typecheck.LookupRuntimeVar("arm64HasATOMICS") // bool
	ir.Syms.Staticuint64s = typecheck.LookupRuntimeVar("staticuint64s")
	ir.Syms.Typedmemclr = typecheck.LookupRuntimeFunc("typedmemclr")
	ir.Syms.Typedmemmove = typecheck.LookupRuntimeFunc("typedmemmove")
	ir.Syms.Udiv = typecheck.LookupRuntimeVar("udiv")                 // asm func with special ABI
	ir.Syms.WriteBarrier = typecheck.LookupRuntimeVar("writeBarrier") // struct { bool; ... }
	ir.Syms.Zerobase = typecheck.LookupRuntimeVar("zerobase")

	// asm funcs with special ABI
	if base.Ctxt.Arch.Name == "amd64" {
		GCWriteBarrierReg = map[int16]*obj.LSym{
			x86.REG_AX: typecheck.LookupRuntimeFunc("gcWriteBarrier"),
			x86.REG_CX: typecheck.LookupRuntimeFunc("gcWriteBarrierCX"),
			x86.REG_DX: typecheck.LookupRuntimeFunc("gcWriteBarrierDX"),
			x86.REG_BX: typecheck.LookupRuntimeFunc("gcWriteBarrierBX"),
			x86.REG_BP: typecheck.LookupRuntimeFunc("gcWriteBarrierBP"),
			x86.REG_SI: typecheck.LookupRuntimeFunc("gcWriteBarrierSI"),
			x86.REG_R8: typecheck.LookupRuntimeFunc("gcWriteBarrierR8"),
			x86.REG_R9: typecheck.LookupRuntimeFunc("gcWriteBarrierR9"),
		}
	}

	if Arch.LinkArch.Family == sys.Wasm {
		BoundsCheckFunc[ssa.BoundsIndex] = typecheck.LookupRuntimeFunc("goPanicIndex")
		BoundsCheckFunc[ssa.BoundsIndexU] = typecheck.LookupRuntimeFunc("goPanicIndexU")
		BoundsCheckFunc[ssa.BoundsSliceAlen] = typecheck.LookupRuntimeFunc("goPanicSliceAlen")
		BoundsCheckFunc[ssa.BoundsSliceAlenU] = typecheck.LookupRuntimeFunc("goPanicSliceAlenU")
		BoundsCheckFunc[ssa.BoundsSliceAcap] = typecheck.LookupRuntimeFunc("goPanicSliceAcap")
		BoundsCheckFunc[ssa.BoundsSliceAcapU] = typecheck.LookupRuntimeFunc("goPanicSliceAcapU")
		BoundsCheckFunc[ssa.BoundsSliceB] = typecheck.LookupRuntimeFunc("goPanicSliceB")
		BoundsCheckFunc[ssa.BoundsSliceBU] = typecheck.LookupRuntimeFunc("goPanicSliceBU")
		BoundsCheckFunc[ssa.BoundsSlice3Alen] = typecheck.LookupRuntimeFunc("goPanicSlice3Alen")
		BoundsCheckFunc[ssa.BoundsSlice3AlenU] = typecheck.LookupRuntimeFunc("goPanicSlice3AlenU")
		BoundsCheckFunc[ssa.BoundsSlice3Acap] = typecheck.LookupRuntimeFunc("goPanicSlice3Acap")
		BoundsCheckFunc[ssa.BoundsSlice3AcapU] = typecheck.LookupRuntimeFunc("goPanicSlice3AcapU")
		BoundsCheckFunc[ssa.BoundsSlice3B] = typecheck.LookupRuntimeFunc("goPanicSlice3B")
		BoundsCheckFunc[ssa.BoundsSlice3BU] = typecheck.LookupRuntimeFunc("goPanicSlice3BU")
		BoundsCheckFunc[ssa.BoundsSlice3C] = typecheck.LookupRuntimeFunc("goPanicSlice3C")
		BoundsCheckFunc[ssa.BoundsSlice3CU] = typecheck.LookupRuntimeFunc("goPanicSlice3CU")
		BoundsCheckFunc[ssa.BoundsConvert] = typecheck.LookupRuntimeFunc("goPanicSliceConvert")
	} else {
		BoundsCheckFunc[ssa.BoundsIndex] = typecheck.LookupRuntimeFunc("panicIndex")
		BoundsCheckFunc[ssa.BoundsIndexU] = typecheck.LookupRuntimeFunc("panicIndexU")
		BoundsCheckFunc[ssa.BoundsSliceAlen] = typecheck.LookupRuntimeFunc("panicSliceAlen")
		BoundsCheckFunc[ssa.BoundsSliceAlenU] = typecheck.LookupRuntimeFunc("panicSliceAlenU")
		BoundsCheckFunc[ssa.BoundsSliceAcap] = typecheck.LookupRuntimeFunc("panicSliceAcap")
		BoundsCheckFunc[ssa.BoundsSliceAcapU] = typecheck.LookupRuntimeFunc("panicSliceAcapU")
		BoundsCheckFunc[ssa.BoundsSliceB] = typecheck.LookupRuntimeFunc("panicSliceB")
		BoundsCheckFunc[ssa.BoundsSliceBU] = typecheck.LookupRuntimeFunc("panicSliceBU")
		BoundsCheckFunc[ssa.BoundsSlice3Alen] = typecheck.LookupRuntimeFunc("panicSlice3Alen")
		BoundsCheckFunc[ssa.BoundsSlice3AlenU] = typecheck.LookupRuntimeFunc("panicSlice3AlenU")
		BoundsCheckFunc[ssa.BoundsSlice3Acap] = typecheck.LookupRuntimeFunc("panicSlice3Acap")
		BoundsCheckFunc[ssa.BoundsSlice3AcapU] = typecheck.LookupRuntimeFunc("panicSlice3AcapU")
		BoundsCheckFunc[ssa.BoundsSlice3B] = typecheck.LookupRuntimeFunc("panicSlice3B")
		BoundsCheckFunc[ssa.BoundsSlice3BU] = typecheck.LookupRuntimeFunc("panicSlice3BU")
		BoundsCheckFunc[ssa.BoundsSlice3C] = typecheck.LookupRuntimeFunc("panicSlice3C")
		BoundsCheckFunc[ssa.BoundsSlice3CU] = typecheck.LookupRuntimeFunc("panicSlice3CU")
		BoundsCheckFunc[ssa.BoundsConvert] = typecheck.LookupRuntimeFunc("panicSliceConvert")
	}
	if Arch.LinkArch.PtrSize == 4 {
		ExtendCheckFunc[ssa.BoundsIndex] = typecheck.LookupRuntimeVar("panicExtendIndex")
		ExtendCheckFunc[ssa.BoundsIndexU] = typecheck.LookupRuntimeVar("panicExtendIndexU")
		ExtendCheckFunc[ssa.BoundsSliceAlen] = typecheck.LookupRuntimeVar("panicExtendSliceAlen")
		ExtendCheckFunc[ssa.BoundsSliceAlenU] = typecheck.LookupRuntimeVar("panicExtendSliceAlenU")
		ExtendCheckFunc[ssa.BoundsSliceAcap] = typecheck.LookupRuntimeVar("panicExtendSliceAcap")
		ExtendCheckFunc[ssa.BoundsSliceAcapU] = typecheck.LookupRuntimeVar("panicExtendSliceAcapU")
		ExtendCheckFunc[ssa.BoundsSliceB] = typecheck.LookupRuntimeVar("panicExtendSliceB")
		ExtendCheckFunc[ssa.BoundsSliceBU] = typecheck.LookupRuntimeVar("panicExtendSliceBU")
		ExtendCheckFunc[ssa.BoundsSlice3Alen] = typecheck.LookupRuntimeVar("panicExtendSlice3Alen")
		ExtendCheckFunc[ssa.BoundsSlice3AlenU] = typecheck.LookupRuntimeVar("panicExtendSlice3AlenU")
		ExtendCheckFunc[ssa.BoundsSlice3Acap] = typecheck.LookupRuntimeVar("panicExtendSlice3Acap")
		ExtendCheckFunc[ssa.BoundsSlice3AcapU] = typecheck.LookupRuntimeVar("panicExtendSlice3AcapU")
		ExtendCheckFunc[ssa.BoundsSlice3B] = typecheck.LookupRuntimeVar("panicExtendSlice3B")
		ExtendCheckFunc[ssa.BoundsSlice3BU] = typecheck.LookupRuntimeVar("panicExtendSlice3BU")
		ExtendCheckFunc[ssa.BoundsSlice3C] = typecheck.LookupRuntimeVar("panicExtendSlice3C")
		ExtendCheckFunc[ssa.BoundsSlice3CU] = typecheck.LookupRuntimeVar("panicExtendSlice3CU")
	}

	// Wasm (all asm funcs with special ABIs)
	ir.Syms.WasmMove = typecheck.LookupRuntimeVar("wasmMove")
	ir.Syms.WasmZero = typecheck.LookupRuntimeVar("wasmZero")
	ir.Syms.WasmDiv = typecheck.LookupRuntimeVar("wasmDiv")
	ir.Syms.WasmTruncS = typecheck.LookupRuntimeVar("wasmTruncS")
	ir.Syms.WasmTruncU = typecheck.LookupRuntimeVar("wasmTruncU")
	ir.Syms.SigPanic = typecheck.LookupRuntimeFunc("sigpanic")
}

// AbiForBodylessFuncStackMap returns the ABI for a bodyless function's stack map.
// This is not necessarily the ABI used to call it.
// Currently (1.17 dev) such a stack map is always ABI0;
// any ABI wrapper that is present is nosplit, hence a precise
// stack map is not needed there (the parameters survive only long
// enough to call the wrapped assembly function).
// This always returns a freshly copied ABI.
func AbiForBodylessFuncStackMap(fn *ir.Func) *abi.ABIConfig {
	return ssaConfig.ABI0.Copy() // No idea what races will result, be safe
}

// These are disabled but remain ready for use in case they are needed for the next regabi port.
// TODO if they are not needed for 1.18 / next register abi port, delete them.
const magicNameDotSuffix = ".*disabled*MagicMethodNameForTestingRegisterABI"
const magicLastTypeName = "*disabled*MagicLastTypeNameForTestingRegisterABI"

// abiForFunc implements ABI policy for a function, but does not return a copy of the ABI.
// Passing a nil function returns the default ABI based on experiment configuration.
func abiForFunc(fn *ir.Func, abi0, abi1 *abi.ABIConfig) *abi.ABIConfig {
	if buildcfg.Experiment.RegabiArgs {
		// Select the ABI based on the function's defining ABI.
		if fn == nil {
			return abi1
		}
		switch fn.ABI {
		case obj.ABI0:
			return abi0
		case obj.ABIInternal:
			// TODO(austin): Clean up the nomenclature here.
			// It's not clear that "abi1" is ABIInternal.
			return abi1
		}
		base.Fatalf("function %v has unknown ABI %v", fn, fn.ABI)
		panic("not reachable")
	}

	a := abi0
	if fn != nil {
		name := ir.FuncName(fn)
		magicName := strings.HasSuffix(name, magicNameDotSuffix)
		if fn.Pragma&ir.RegisterParams != 0 { // TODO(register args) remove after register abi is working
			if strings.Contains(name, ".") {
				if !magicName {
					base.ErrorfAt(fn.Pos(), "Calls to //go:registerparams method %s won't work, remove the pragma from the declaration.", name)
				}
			}
			a = abi1
		} else if magicName {
			if base.FmtPos(fn.Pos()) == "<autogenerated>:1" {
				// no way to put a pragma here, and it will error out in the real source code if they did not do it there.
				a = abi1
			} else {
				base.ErrorfAt(fn.Pos(), "Methods with magic name %s (method %s) must also specify //go:registerparams", magicNameDotSuffix[1:], name)
			}
		}
		if regAbiForFuncType(fn.Type().FuncType()) {
			// fmt.Printf("Saw magic last type name for function %s\n", name)
			a = abi1
		}
	}
	return a
}

func regAbiForFuncType(ft *types.Func) bool {
	np := ft.Params.NumFields()
	return np > 0 && strings.Contains(ft.Params.FieldType(np-1).String(), magicLastTypeName)
}

// getParam returns the Field of ith param of node n (which is a
// function/method/interface call), where the receiver of a method call is
// considered as the 0th parameter. This does not include the receiver of an
// interface call.
func getParam(n *ir.CallExpr, i int) *types.Field {
	t := n.X.Type()
	if n.Op() == ir.OCALLMETH {
		base.Fatalf("OCALLMETH missed by walkCall")
	}
	return t.Params().Field(i)
}

// dvarint writes a varint v to the funcdata in symbol x and returns the new offset
func dvarint(x *obj.LSym, off int, v int64) int {
	if v < 0 || v > 1e9 {
		panic(fmt.Sprintf("dvarint: bad offset for funcdata - %v", v))
	}
	if v < 1<<7 {
		return objw.Uint8(x, off, uint8(v))
	}
	off = objw.Uint8(x, off, uint8((v&127)|128))
	if v < 1<<14 {
		return objw.Uint8(x, off, uint8(v>>7))
	}
	off = objw.Uint8(x, off, uint8(((v>>7)&127)|128))
	if v < 1<<21 {
		return objw.Uint8(x, off, uint8(v>>14))
	}
	off = objw.Uint8(x, off, uint8(((v>>14)&127)|128))
	if v < 1<<28 {
		return objw.Uint8(x, off, uint8(v>>21))
	}
	off = objw.Uint8(x, off, uint8(((v>>21)&127)|128))
	return objw.Uint8(x, off, uint8(v>>28))
}

// emitOpenDeferInfo emits FUNCDATA information about the defers in a function
// that is using open-coded defers.  This funcdata is used to determine the active
// defers in a function and execute those defers during panic processing.
//
// The funcdata is all encoded in varints (since values will almost always be less than
// 128, but stack offsets could potentially be up to 2Gbyte). All "locations" (offsets)
// for stack variables are specified as the number of bytes below varp (pointer to the
// top of the local variables) for their starting address. The format is:
//
//  - Max total argument size among all the defers
//  - Offset of the deferBits variable
//  - Number of defers in the function
//  - Information about each defer call, in reverse order of appearance in the function:
//    - Total argument size of the call
//    - Offset of the closure value to call
//    - Number of arguments (including interface receiver or method receiver as first arg)
//    - Information about each argument
//      - Offset of the stored defer argument in this function's frame
//      - Size of the argument
//      - Offset of where argument should be placed in the args frame when making call
func (s *state) emitOpenDeferInfo() {
	x := base.Ctxt.Lookup(s.curfn.LSym.Name + ".opendefer")
	s.curfn.LSym.Func().OpenCodedDeferInfo = x
	off := 0

	// Compute maxargsize (max size of arguments for all defers)
	// first, so we can output it first to the funcdata
	var maxargsize int64
	for i := len(s.openDefers) - 1; i >= 0; i-- {
		r := s.openDefers[i]
		argsize := r.n.X.Type().ArgWidth() // TODO register args: but maybe use of abi0 will make this easy
		if argsize > maxargsize {
			maxargsize = argsize
		}
	}
	off = dvarint(x, off, maxargsize)
	off = dvarint(x, off, -s.deferBitsTemp.FrameOffset())
	off = dvarint(x, off, int64(len(s.openDefers)))

	// Write in reverse-order, for ease of running in that order at runtime
	for i := len(s.openDefers) - 1; i >= 0; i-- {
		r := s.openDefers[i]
		off = dvarint(x, off, r.n.X.Type().ArgWidth())
		off = dvarint(x, off, -r.closureNode.FrameOffset())
		numArgs := len(r.argNodes)
		if r.rcvrNode != nil {
			// If there's an interface receiver, treat/place it as the first
			// arg. (If there is a method receiver, it's already included as
			// first arg in r.argNodes.)
			numArgs++
		}
		off = dvarint(x, off, int64(numArgs))
		argAdjust := 0 // presence of receiver offsets the parameter count.
		if r.rcvrNode != nil {
			off = dvarint(x, off, -okOffset(r.rcvrNode.FrameOffset()))
			off = dvarint(x, off, s.config.PtrSize)
			off = dvarint(x, off, 0) // This is okay because defer records use ABI0 (for now)
			argAdjust++
		}

		// TODO(register args) assume abi0 for this?
		ab := s.f.ABI0
		pri := ab.ABIAnalyzeFuncType(r.n.X.Type().FuncType())
		for j, arg := range r.argNodes {
			f := getParam(r.n, j)
			off = dvarint(x, off, -okOffset(arg.FrameOffset()))
			off = dvarint(x, off, f.Type.Size())
			off = dvarint(x, off, okOffset(pri.InParam(j+argAdjust).FrameOffset(pri)))
		}
	}
}

func okOffset(offset int64) int64 {
	if offset == types.BOGUS_FUNARG_OFFSET {
		panic(fmt.Errorf("Bogus offset %d", offset))
	}
	return offset
}

// buildssa builds an SSA function for fn.
// worker indicates which of the backend workers is doing the processing.
func buildssa(fn *ir.Func, worker int) *ssa.Func {
	name := ir.FuncName(fn)
	printssa := false
	if ssaDump != "" { // match either a simple name e.g. "(*Reader).Reset", package.name e.g. "compress/gzip.(*Reader).Reset", or subpackage name "gzip.(*Reader).Reset"
		pkgDotName := base.Ctxt.Pkgpath + "." + name
		printssa = name == ssaDump ||
			strings.HasSuffix(pkgDotName, ssaDump) && (pkgDotName == ssaDump || strings.HasSuffix(pkgDotName, "/"+ssaDump))
	}
	var astBuf *bytes.Buffer
	if printssa {
		astBuf = &bytes.Buffer{}
		ir.FDumpList(astBuf, "buildssa-enter", fn.Enter)
		ir.FDumpList(astBuf, "buildssa-body", fn.Body)
		ir.FDumpList(astBuf, "buildssa-exit", fn.Exit)
		if ssaDumpStdout {
			fmt.Println("generating SSA for", name)
			fmt.Print(astBuf.String())
		}
	}

	var s state
	s.pushLine(fn.Pos())
	defer s.popLine()

	s.hasdefer = fn.HasDefer()
	if fn.Pragma&ir.CgoUnsafeArgs != 0 {
		s.cgoUnsafeArgs = true
	}

	fe := ssafn{
		curfn: fn,
		log:   printssa && ssaDumpStdout,
	}
	s.curfn = fn

	s.f = ssa.NewFunc(&fe)
	s.config = ssaConfig
	s.f.Type = fn.Type()
	s.f.Config = ssaConfig
	s.f.Cache = &ssaCaches[worker]
	s.f.Cache.Reset()
	s.f.Name = name
	s.f.DebugTest = s.f.DebugHashMatch("GOSSAHASH")
	s.f.PrintOrHtmlSSA = printssa
	if fn.Pragma&ir.Nosplit != 0 {
		s.f.NoSplit = true
	}
	s.f.ABI0 = ssaConfig.ABI0.Copy() // Make a copy to avoid racy map operations in type-register-width cache.
	s.f.ABI1 = ssaConfig.ABI1.Copy()
	s.f.ABIDefault = abiForFunc(nil, s.f.ABI0, s.f.ABI1)
	s.f.ABISelf = abiForFunc(fn, s.f.ABI0, s.f.ABI1)

	s.panics = map[funcLine]*ssa.Block{}
	s.softFloat = s.config.SoftFloat

	// Allocate starting block
	s.f.Entry = s.f.NewBlock(ssa.BlockPlain)
	s.f.Entry.Pos = fn.Pos()

	if printssa {
		ssaDF := ssaDumpFile
		if ssaDir != "" {
			ssaDF = filepath.Join(ssaDir, base.Ctxt.Pkgpath+"."+name+".html")
			ssaD := filepath.Dir(ssaDF)
			os.MkdirAll(ssaD, 0755)
		}
		s.f.HTMLWriter = ssa.NewHTMLWriter(ssaDF, s.f, ssaDumpCFG)
		// TODO: generate and print a mapping from nodes to values and blocks
		dumpSourcesColumn(s.f.HTMLWriter, fn)
		s.f.HTMLWriter.WriteAST("AST", astBuf)
	}

	// Allocate starting values
	s.labels = map[string]*ssaLabel{}
	s.fwdVars = map[ir.Node]*ssa.Value{}
	s.startmem = s.entryNewValue0(ssa.OpInitMem, types.TypeMem)

	s.hasOpenDefers = base.Flag.N == 0 && s.hasdefer && !s.curfn.OpenCodedDeferDisallowed()
	switch {
	case base.Debug.NoOpenDefer != 0:
		s.hasOpenDefers = false
	case s.hasOpenDefers && (base.Ctxt.Flag_shared || base.Ctxt.Flag_dynlink) && base.Ctxt.Arch.Name == "386":
		// Don't support open-coded defers for 386 ONLY when using shared
		// libraries, because there is extra code (added by rewriteToUseGot())
		// preceding the deferreturn/ret code that we don't track correctly.
		s.hasOpenDefers = false
	}
	if s.hasOpenDefers && len(s.curfn.Exit) > 0 {
		// Skip doing open defers if there is any extra exit code (likely
		// race detection), since we will not generate that code in the
		// case of the extra deferreturn/ret segment.
		s.hasOpenDefers = false
	}
	if s.hasOpenDefers {
		// Similarly, skip if there are any heap-allocated result
		// parameters that need to be copied back to their stack slots.
		for _, f := range s.curfn.Type().Results().FieldSlice() {
			if !f.Nname.(*ir.Name).OnStack() {
				s.hasOpenDefers = false
				break
			}
		}
	}
	if s.hasOpenDefers &&
		s.curfn.NumReturns*s.curfn.NumDefers > 15 {
		// Since we are generating defer calls at every exit for
		// open-coded defers, skip doing open-coded defers if there are
		// too many returns (especially if there are multiple defers).
		// Open-coded defers are most important for improving performance
		// for smaller functions (which don't have many returns).
		s.hasOpenDefers = false
	}

	s.sp = s.entryNewValue0(ssa.OpSP, types.Types[types.TUINTPTR]) // TODO: use generic pointer type (unsafe.Pointer?) instead
	s.sb = s.entryNewValue0(ssa.OpSB, types.Types[types.TUINTPTR])

	s.startBlock(s.f.Entry)
	s.vars[memVar] = s.startmem
	if s.hasOpenDefers {
		// Create the deferBits variable and stack slot.  deferBits is a
		// bitmask showing which of the open-coded defers in this function
		// have been activated.
		deferBitsTemp := typecheck.TempAt(src.NoXPos, s.curfn, types.Types[types.TUINT8])
		deferBitsTemp.SetAddrtaken(true)
		s.deferBitsTemp = deferBitsTemp
		// For this value, AuxInt is initialized to zero by default
		startDeferBits := s.entryNewValue0(ssa.OpConst8, types.Types[types.TUINT8])
		s.vars[deferBitsVar] = startDeferBits
		s.deferBitsAddr = s.addr(deferBitsTemp)
		s.store(types.Types[types.TUINT8], s.deferBitsAddr, startDeferBits)
		// Make sure that the deferBits stack slot is kept alive (for use
		// by panics) and stores to deferBits are not eliminated, even if
		// all checking code on deferBits in the function exit can be
		// eliminated, because the defer statements were all
		// unconditional.
		s.vars[memVar] = s.newValue1Apos(ssa.OpVarLive, types.TypeMem, deferBitsTemp, s.mem(), false)
	}

	var params *abi.ABIParamResultInfo
	params = s.f.ABISelf.ABIAnalyze(fn.Type(), true)

	// Generate addresses of local declarations
	s.decladdrs = map[*ir.Name]*ssa.Value{}
	for _, n := range fn.Dcl {
		switch n.Class {
		case ir.PPARAM:
			// Be aware that blank and unnamed input parameters will not appear here, but do appear in the type
			s.decladdrs[n] = s.entryNewValue2A(ssa.OpLocalAddr, types.NewPtr(n.Type()), n, s.sp, s.startmem)
		case ir.PPARAMOUT:
			s.decladdrs[n] = s.entryNewValue2A(ssa.OpLocalAddr, types.NewPtr(n.Type()), n, s.sp, s.startmem)
		case ir.PAUTO:
			// processed at each use, to prevent Addr coming
			// before the decl.
		default:
			s.Fatalf("local variable with class %v unimplemented", n.Class)
		}
	}

	s.f.OwnAux = ssa.OwnAuxCall(fn.LSym, params)

	// Populate SSAable arguments.
	for _, n := range fn.Dcl {
		if n.Class == ir.PPARAM {
			if s.canSSA(n) {
				v := s.newValue0A(ssa.OpArg, n.Type(), n)
				s.vars[n] = v
				s.addNamedValue(n, v) // This helps with debugging information, not needed for compilation itself.
			} else { // address was taken AND/OR too large for SSA
				paramAssignment := ssa.ParamAssignmentForArgName(s.f, n)
				if len(paramAssignment.Registers) > 0 {
					if TypeOK(n.Type()) { // SSA-able type, so address was taken -- receive value in OpArg, DO NOT bind to var, store immediately to memory.
						v := s.newValue0A(ssa.OpArg, n.Type(), n)
						s.store(n.Type(), s.decladdrs[n], v)
					} else { // Too big for SSA.
						// Brute force, and early, do a bunch of stores from registers
						// TODO fix the nasty storeArgOrLoad recursion in ssa/expand_calls.go so this Just Works with store of a big Arg.
						s.storeParameterRegsToStack(s.f.ABISelf, paramAssignment, n, s.decladdrs[n], false)
					}
				}
			}
		}
	}

	// Populate closure variables.
	if !fn.ClosureCalled() {
		clo := s.entryNewValue0(ssa.OpGetClosurePtr, s.f.Config.Types.BytePtr)
		offset := int64(types.PtrSize) // PtrSize to skip past function entry PC field
		for _, n := range fn.ClosureVars {
			typ := n.Type()
			if !n.Byval() {
				typ = types.NewPtr(typ)
			}

			offset = types.Rnd(offset, typ.Alignment())
			ptr := s.newValue1I(ssa.OpOffPtr, types.NewPtr(typ), offset, clo)
			offset += typ.Size()

			// If n is a small variable captured by value, promote
			// it to PAUTO so it can be converted to SSA.
			//
			// Note: While we never capture a variable by value if
			// the user took its address, we may have generated
			// runtime calls that did (#43701). Since we don't
			// convert Addrtaken variables to SSA anyway, no point
			// in promoting them either.
			if n.Byval() && !n.Addrtaken() && TypeOK(n.Type()) {
				n.Class = ir.PAUTO
				fn.Dcl = append(fn.Dcl, n)
				s.assign(n, s.load(n.Type(), ptr), false, 0)
				continue
			}

			if !n.Byval() {
				ptr = s.load(typ, ptr)
			}
			s.setHeapaddr(fn.Pos(), n, ptr)
		}
	}

	// Convert the AST-based IR to the SSA-based IR
	s.stmtList(fn.Enter)
	s.zeroResults()
	s.paramsToHeap()
	s.stmtList(fn.Body)

	// fallthrough to exit
	if s.curBlock != nil {
		s.pushLine(fn.Endlineno)
		s.exit()
		s.popLine()
	}

	for _, b := range s.f.Blocks {
		if b.Pos != src.NoXPos {
			s.updateUnsetPredPos(b)
		}
	}

	s.f.HTMLWriter.WritePhase("before insert phis", "before insert phis")

	s.insertPhis()

	// Main call to ssa package to compile function
	ssa.Compile(s.f)

	if s.hasOpenDefers {
		s.emitOpenDeferInfo()
	}

	// Record incoming parameter spill information for morestack calls emitted in the assembler.
	// This is done here, using all the parameters (used, partially used, and unused) because
	// it mimics the behavior of the former ABI (everything stored) and because it's not 100%
	// clear if naming conventions are respected in autogenerated code.
	// TODO figure out exactly what's unused, don't spill it. Make liveness fine-grained, also.
	// TODO non-amd64 architectures have link registers etc that may require adjustment here.
	for _, p := range params.InParams() {
		typs, offs := p.RegisterTypesAndOffsets()
		for i, t := range typs {
			o := offs[i]                // offset within parameter
			fo := p.FrameOffset(params) // offset of parameter in frame
			reg := ssa.ObjRegForAbiReg(p.Registers[i], s.f.Config)
			s.f.RegArgs = append(s.f.RegArgs, ssa.Spill{Reg: reg, Offset: fo + o, Type: t})
		}
	}

	return s.f
}

func (s *state) storeParameterRegsToStack(abi *abi.ABIConfig, paramAssignment *abi.ABIParamAssignment, n *ir.Name, addr *ssa.Value, pointersOnly bool) {
	typs, offs := paramAssignment.RegisterTypesAndOffsets()
	for i, t := range typs {
		if pointersOnly && !t.IsPtrShaped() {
			continue
		}
		r := paramAssignment.Registers[i]
		o := offs[i]
		op, reg := ssa.ArgOpAndRegisterFor(r, abi)
		aux := &ssa.AuxNameOffset{Name: n, Offset: o}
		v := s.newValue0I(op, t, reg)
		v.Aux = aux
		p := s.newValue1I(ssa.OpOffPtr, types.NewPtr(t), o, addr)
		s.store(t, p, v)
	}
}

// zeroResults zeros the return values at the start of the function.
// We need to do this very early in the function.  Defer might stop a
// panic and show the return values as they exist at the time of
// panic.  For precise stacks, the garbage collector assumes results
// are always live, so we need to zero them before any allocations,
// even allocations to move params/results to the heap.
func (s *state) zeroResults() {
	for _, f := range s.curfn.Type().Results().FieldSlice() {
		n := f.Nname.(*ir.Name)
		if !n.OnStack() {
			// The local which points to the return value is the
			// thing that needs zeroing. This is already handled
			// by a Needzero annotation in plive.go:(*liveness).epilogue.
			continue
		}
		// Zero the stack location containing f.
		if typ := n.Type(); TypeOK(typ) {
			s.assign(n, s.zeroVal(typ), false, 0)
		} else {
			s.vars[memVar] = s.newValue1A(ssa.OpVarDef, types.TypeMem, n, s.mem())
			s.zero(n.Type(), s.decladdrs[n])
		}
	}
}

// paramsToHeap produces code to allocate memory for heap-escaped parameters
// and to copy non-result parameters' values from the stack.
func (s *state) paramsToHeap() {
	do := func(params *types.Type) {
		for _, f := range params.FieldSlice() {
			if f.Nname == nil {
				continue // anonymous or blank parameter
			}
			n := f.Nname.(*ir.Name)
			if ir.IsBlank(n) || n.OnStack() {
				continue
			}
			s.newHeapaddr(n)
			if n.Class == ir.PPARAM {
				s.move(n.Type(), s.expr(n.Heapaddr), s.decladdrs[n])
			}
		}
	}

	typ := s.curfn.Type()
	do(typ.Recvs())
	do(typ.Params())
	do(typ.Results())
}

// newHeapaddr allocates heap memory for n and sets its heap address.
func (s *state) newHeapaddr(n *ir.Name) {
	s.setHeapaddr(n.Pos(), n, s.newObject(n.Type()))
}

// setHeapaddr allocates a new PAUTO variable to store ptr (which must be non-nil)
// and then sets it as n's heap address.
func (s *state) setHeapaddr(pos src.XPos, n *ir.Name, ptr *ssa.Value) {
	if !ptr.Type.IsPtr() || !types.Identical(n.Type(), ptr.Type.Elem()) {
		base.FatalfAt(n.Pos(), "setHeapaddr %L with type %v", n, ptr.Type)
	}

	// Declare variable to hold address.
	addr := ir.NewNameAt(pos, &types.Sym{Name: "&" + n.Sym().Name, Pkg: types.LocalPkg})
	addr.SetType(types.NewPtr(n.Type()))
	addr.Class = ir.PAUTO
	addr.SetUsed(true)
	addr.Curfn = s.curfn
	s.curfn.Dcl = append(s.curfn.Dcl, addr)
	types.CalcSize(addr.Type())

	if n.Class == ir.PPARAMOUT {
		addr.SetIsOutputParamHeapAddr(true)
	}

	n.Heapaddr = addr
	s.assign(addr, ptr, false, 0)
}

// newObject returns an SSA value denoting new(typ).
func (s *state) newObject(typ *types.Type) *ssa.Value {
	if typ.Size() == 0 {
		return s.newValue1A(ssa.OpAddr, types.NewPtr(typ), ir.Syms.Zerobase, s.sb)
	}
	return s.rtcall(ir.Syms.Newobject, true, []*types.Type{types.NewPtr(typ)}, s.reflectType(typ))[0]
}

// reflectType returns an SSA value representing a pointer to typ's
// reflection type descriptor.
func (s *state) reflectType(typ *types.Type) *ssa.Value {
	lsym := reflectdata.TypeLinksym(typ)
	return s.entryNewValue1A(ssa.OpAddr, types.NewPtr(types.Types[types.TUINT8]), lsym, s.sb)
}

func dumpSourcesColumn(writer *ssa.HTMLWriter, fn *ir.Func) {
	// Read sources of target function fn.
	fname := base.Ctxt.PosTable.Pos(fn.Pos()).Filename()
	targetFn, err := readFuncLines(fname, fn.Pos().Line(), fn.Endlineno.Line())
	if err != nil {
		writer.Logf("cannot read sources for function %v: %v", fn, err)
	}

	// Read sources of inlined functions.
	var inlFns []*ssa.FuncLines
	for _, fi := range ssaDumpInlined {
		elno := fi.Endlineno
		fname := base.Ctxt.PosTable.Pos(fi.Pos()).Filename()
		fnLines, err := readFuncLines(fname, fi.Pos().Line(), elno.Line())
		if err != nil {
			writer.Logf("cannot read sources for inlined function %v: %v", fi, err)
			continue
		}
		inlFns = append(inlFns, fnLines)
	}

	sort.Sort(ssa.ByTopo(inlFns))
	if targetFn != nil {
		inlFns = append([]*ssa.FuncLines{targetFn}, inlFns...)
	}

	writer.WriteSources("sources", inlFns)
}

func readFuncLines(file string, start, end uint) (*ssa.FuncLines, error) {
	f, err := os.Open(os.ExpandEnv(file))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var lines []string
	ln := uint(1)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() && ln <= end {
		if ln >= start {
			lines = append(lines, scanner.Text())
		}
		ln++
	}
	return &ssa.FuncLines{Filename: file, StartLineno: start, Lines: lines}, nil
}

// updateUnsetPredPos propagates the earliest-value position information for b
// towards all of b's predecessors that need a position, and recurs on that
// predecessor if its position is updated. B should have a non-empty position.
func (s *state) updateUnsetPredPos(b *ssa.Block) {
	if b.Pos == src.NoXPos {
		s.Fatalf("Block %s should have a position", b)
	}
	bestPos := src.NoXPos
	for _, e := range b.Preds {
		p := e.Block()
		if !p.LackingPos() {
			continue
		}
		if bestPos == src.NoXPos {
			bestPos = b.Pos
			for _, v := range b.Values {
				if v.LackingPos() {
					continue
				}
				if v.Pos != src.NoXPos {
					// Assume values are still in roughly textual order;
					// TODO: could also seek minimum position?
					bestPos = v.Pos
					break
				}
			}
		}
		p.Pos = bestPos
		s.updateUnsetPredPos(p) // We do not expect long chains of these, thus recursion is okay.
	}
}

// Information about each open-coded defer.
type openDeferInfo struct {
	// The node representing the call of the defer
	n *ir.CallExpr
	// If defer call is closure call, the address of the argtmp where the
	// closure is stored.
	closure *ssa.Value
	// The node representing the argtmp where the closure is stored - used for
	// function, method, or interface call, to store a closure that panic
	// processing can use for this defer.
	closureNode *ir.Name
	// If defer call is interface call, the address of the argtmp where the
	// receiver is stored
	rcvr *ssa.Value
	// The node representing the argtmp where the receiver is stored
	rcvrNode *ir.Name
	// The addresses of the argtmps where the evaluated arguments of the defer
	// function call are stored.
	argVals []*ssa.Value
	// The nodes representing the argtmps where the args of the defer are stored
	argNodes []*ir.Name
}

type state struct {
	// configuration (arch) information
	config *ssa.Config

	// function we're building
	f *ssa.Func

	// Node for function
	curfn *ir.Func

	// labels in f
	labels map[string]*ssaLabel

	// unlabeled break and continue statement tracking
	breakTo    *ssa.Block // current target for plain break statement
	continueTo *ssa.Block // current target for plain continue statement

	// current location where we're interpreting the AST
	curBlock *ssa.Block

	// variable assignments in the current block (map from variable symbol to ssa value)
	// *Node is the unique identifier (an ONAME Node) for the variable.
	// TODO: keep a single varnum map, then make all of these maps slices instead?
	vars map[ir.Node]*ssa.Value

	// fwdVars are variables that are used before they are defined in the current block.
	// This map exists just to coalesce multiple references into a single FwdRef op.
	// *Node is the unique identifier (an ONAME Node) for the variable.
	fwdVars map[ir.Node]*ssa.Value

	// all defined variables at the end of each block. Indexed by block ID.
	defvars []map[ir.Node]*ssa.Value

	// addresses of PPARAM and PPARAMOUT variables on the stack.
	decladdrs map[*ir.Name]*ssa.Value

	// starting values. Memory, stack pointer, and globals pointer
	startmem *ssa.Value
	sp       *ssa.Value
	sb       *ssa.Value
	// value representing address of where deferBits autotmp is stored
	deferBitsAddr *ssa.Value
	deferBitsTemp *ir.Name

	// line number stack. The current line number is top of stack
	line []src.XPos
	// the last line number processed; it may have been popped
	lastPos src.XPos

	// list of panic calls by function name and line number.
	// Used to deduplicate panic calls.
	panics map[funcLine]*ssa.Block

	cgoUnsafeArgs bool
	hasdefer      bool // whether the function contains a defer statement
	softFloat     bool
	hasOpenDefers bool // whether we are doing open-coded defers

	// If doing open-coded defers, list of info about the defer calls in
	// scanning order. Hence, at exit we should run these defers in reverse
	// order of this list
	openDefers []*openDeferInfo
	// For open-coded defers, this is the beginning and end blocks of the last
	// defer exit code that we have generated so far. We use these to share
	// code between exits if the shareDeferExits option (disabled by default)
	// is on.
	lastDeferExit       *ssa.Block // Entry block of last defer exit code we generated
	lastDeferFinalBlock *ssa.Block // Final block of last defer exit code we generated
	lastDeferCount      int        // Number of defers encountered at that point

	prevCall *ssa.Value // the previous call; use this to tie results to the call op.
}

type funcLine struct {
	f    *obj.LSym
	base *src.PosBase
	line uint
}

type ssaLabel struct {
	target         *ssa.Block // block identified by this label
	breakTarget    *ssa.Block // block to break to in control flow node identified by this label
	continueTarget *ssa.Block // block to continue to in control flow node identified by this label
}

// label returns the label associated with sym, creating it if necessary.
func (s *state) label(sym *types.Sym) *ssaLabel {
	lab := s.labels[sym.Name]
	if lab == nil {
		lab = new(ssaLabel)
		s.labels[sym.Name] = lab
	}
	return lab
}

func (s *state) Logf(msg string, args ...interface{}) { s.f.Logf(msg, args...) }
func (s *state) Log() bool                            { return s.f.Log() }
func (s *state) Fatalf(msg string, args ...interface{}) {
	s.f.Frontend().Fatalf(s.peekPos(), msg, args...)
}
func (s *state) Warnl(pos src.XPos, msg string, args ...interface{}) { s.f.Warnl(pos, msg, args...) }
func (s *state) Debug_checknil() bool                                { return s.f.Frontend().Debug_checknil() }

func ssaMarker(name string) *ir.Name {
	return typecheck.NewName(&types.Sym{Name: name})
}

var (
	// marker node for the memory variable
	memVar = ssaMarker("mem")

	// marker nodes for temporary variables
	ptrVar       = ssaMarker("ptr")
	lenVar       = ssaMarker("len")
	newlenVar    = ssaMarker("newlen")
	capVar       = ssaMarker("cap")
	typVar       = ssaMarker("typ")
	okVar        = ssaMarker("ok")
	deferBitsVar = ssaMarker("deferBits")
)

// startBlock sets the current block we're generating code in to b.
func (s *state) startBlock(b *ssa.Block) {
	if s.curBlock != nil {
		s.Fatalf("starting block %v when block %v has not ended", b, s.curBlock)
	}
	s.curBlock = b
	s.vars = map[ir.Node]*ssa.Value{}
	for n := range s.fwdVars {
		delete(s.fwdVars, n)
	}
}

// endBlock marks the end of generating code for the current block.
// Returns the (former) current block. Returns nil if there is no current
// block, i.e. if no code flows to the current execution point.
func (s *state) endBlock() *ssa.Block {
	b := s.curBlock
	if b == nil {
		return nil
	}
	for len(s.defvars) <= int(b.ID) {
		s.defvars = append(s.defvars, nil)
	}
	s.defvars[b.ID] = s.vars
	s.curBlock = nil
	s.vars = nil
	if b.LackingPos() {
		// Empty plain blocks get the line of their successor (handled after all blocks created),
		// except for increment blocks in For statements (handled in ssa conversion of OFOR),
		// and for blocks ending in GOTO/BREAK/CONTINUE.
		b.Pos = src.NoXPos
	} else {
		b.Pos = s.lastPos
	}
	return b
}

// pushLine pushes a line number on the line number stack.
func (s *state) pushLine(line src.XPos) {
	if !line.IsKnown() {
		// the frontend may emit node with line number missing,
		// use the parent line number in this case.
		line = s.peekPos()
		if base.Flag.K != 0 {
			base.Warn("buildssa: unknown position (line 0)")
		}
	} else {
		s.lastPos = line
	}

	s.line = append(s.line, line)
}

// popLine pops the top of the line number stack.
func (s *state) popLine() {
	s.line = s.line[:len(s.line)-1]
}

// peekPos peeks the top of the line number stack.
func (s *state) peekPos() src.XPos {
	return s.line[len(s.line)-1]
}

// newValue0 adds a new value with no arguments to the current block.
func (s *state) newValue0(op ssa.Op, t *types.Type) *ssa.Value {
	return s.curBlock.NewValue0(s.peekPos(), op, t)
}

// newValue0A adds a new value with no arguments and an aux value to the current block.
func (s *state) newValue0A(op ssa.Op, t *types.Type, aux ssa.Aux) *ssa.Value {
	return s.curBlock.NewValue0A(s.peekPos(), op, t, aux)
}

// newValue0I adds a new value with no arguments and an auxint value to the current block.
func (s *state) newValue0I(op ssa.Op, t *types.Type, auxint int64) *ssa.Value {
	return s.curBlock.NewValue0I(s.peekPos(), op, t, auxint)
}

// newValue1 adds a new value with one argument to the current block.
func (s *state) newValue1(op ssa.Op, t *types.Type, arg *ssa.Value) *ssa.Value {
	return s.curBlock.NewValue1(s.peekPos(), op, t, arg)
}

// newValue1A adds a new value with one argument and an aux value to the current block.
func (s *state) newValue1A(op ssa.Op, t *types.Type, aux ssa.Aux, arg *ssa.Value) *ssa.Value {
	return s.curBlock.NewValue1A(s.peekPos(), op, t, aux, arg)
}

// newValue1Apos adds a new value with one argument and an aux value to the current block.
// isStmt determines whether the created values may be a statement or not
// (i.e., false means never, yes means maybe).
func (s *state) newValue1Apos(op ssa.Op, t *types.Type, aux ssa.Aux, arg *ssa.Value, isStmt bool) *ssa.Value {
	if isStmt {
		return s.curBlock.NewValue1A(s.peekPos(), op, t, aux, arg)
	}
	return s.curBlock.NewValue1A(s.peekPos().WithNotStmt(), op, t, aux, arg)
}

// newValue1I adds a new value with one argument and an auxint value to the current block.
func (s *state) newValue1I(op ssa.Op, t *types.Type, aux int64, arg *ssa.Value) *ssa.Value {
	return s.curBlock.NewValue1I(s.peekPos(), op, t, aux, arg)
}

// newValue2 adds a new value with two arguments to the current block.
func (s *state) newValue2(op ssa.Op, t *types.Type, arg0, arg1 *ssa.Value) *ssa.Value {
	return s.curBlock.NewValue2(s.peekPos(), op, t, arg0, arg1)
}

// newValue2A adds a new value with two arguments and an aux value to the current block.
func (s *state) newValue2A(op ssa.Op, t *types.Type, aux ssa.Aux, arg0, arg1 *ssa.Value) *ssa.Value {
	return s.curBlock.NewValue2A(s.peekPos(), op, t, aux, arg0, arg1)
}

// newValue2Apos adds a new value with two arguments and an aux value to the current block.
// isStmt determines whether the created values may be a statement or not
// (i.e., false means never, yes means maybe).
func (s *state) newValue2Apos(op ssa.Op, t *types.Type, aux ssa.Aux, arg0, arg1 *ssa.Value, isStmt bool) *ssa.Value {
	if isStmt {
		return s.curBlock.NewValue2A(s.peekPos(), op, t, aux, arg0, arg1)
	}
	return s.curBlock.NewValue2A(s.peekPos().WithNotStmt(), op, t, aux, arg0, arg1)
}

// newValue2I adds a new value with two arguments and an auxint value to the current block.
func (s *state) newValue2I(op ssa.Op, t *types.Type, aux int64, arg0, arg1 *ssa.Value) *ssa.Value {
	return s.curBlock.NewValue2I(s.peekPos(), op, t, aux, arg0, arg1)
}

// newValue3 adds a new value with three arguments to the current block.
func (s *state) newValue3(op ssa.Op, t *types.Type, arg0, arg1, arg2 *ssa.Value) *ssa.Value {
	return s.curBlock.NewValue3(s.peekPos(), op, t, arg0, arg1, arg2)
}

// newValue3I adds a new value with three arguments and an auxint value to the current block.
func (s *state) newValue3I(op ssa.Op, t *types.Type, aux int64, arg0, arg1, arg2 *ssa.Value) *ssa.Value {
	return s.curBlock.NewValue3I(s.peekPos(), op, t, aux, arg0, arg1, arg2)
}

// newValue3A adds a new value with three arguments and an aux value to the current block.
func (s *state) newValue3A(op ssa.Op, t *types.Type, aux ssa.Aux, arg0, arg1, arg2 *ssa.Value) *ssa.Value {
	return s.curBlock.NewValue3A(s.peekPos(), op, t, aux, arg0, arg1, arg2)
}

// newValue3Apos adds a new value with three arguments and an aux value to the current block.
// isStmt determines whether the created values may be a statement or not
// (i.e., false means never, yes means maybe).
func (s *state) newValue3Apos(op ssa.Op, t *types.Type, aux ssa.Aux, arg0, arg1, arg2 *ssa.Value, isStmt bool) *ssa.Value {
	if isStmt {
		return s.curBlock.NewValue3A(s.peekPos(), op, t, aux, arg0, arg1, arg2)
	}
	return s.curBlock.NewValue3A(s.peekPos().WithNotStmt(), op, t, aux, arg0, arg1, arg2)
}

// newValue4 adds a new value with four arguments to the current block.
func (s *state) newValue4(op ssa.Op, t *types.Type, arg0, arg1, arg2, arg3 *ssa.Value) *ssa.Value {
	return s.curBlock.NewValue4(s.peekPos(), op, t, arg0, arg1, arg2, arg3)
}

// newValue4 adds a new value with four arguments and an auxint value to the current block.
func (s *state) newValue4I(op ssa.Op, t *types.Type, aux int64, arg0, arg1, arg2, arg3 *ssa.Value) *ssa.Value {
	return s.curBlock.NewValue4I(s.peekPos(), op, t, aux, arg0, arg1, arg2, arg3)
}

func (s *state) entryBlock() *ssa.Block {
	b := s.f.Entry
	if base.Flag.N > 0 && s.curBlock != nil {
		// If optimizations are off, allocate in current block instead. Since with -N
		// we're not doing the CSE or tighten passes, putting lots of stuff in the
		// entry block leads to O(n^2) entries in the live value map during regalloc.
		// See issue 45897.
		b = s.curBlock
	}
	return b
}

// entryNewValue0 adds a new value with no arguments to the entry block.
func (s *state) entryNewValue0(op ssa.Op, t *types.Type) *ssa.Value {
	return s.entryBlock().NewValue0(src.NoXPos, op, t)
}

// entryNewValue0A adds a new value with no arguments and an aux value to the entry block.
func (s *state) entryNewValue0A(op ssa.Op, t *types.Type, aux ssa.Aux) *ssa.Value {
	return s.entryBlock().NewValue0A(src.NoXPos, op, t, aux)
}

// entryNewValue1 adds a new value with one argument to the entry block.
func (s *state) entryNewValue1(op ssa.Op, t *types.Type, arg *ssa.Value) *ssa.Value {
	return s.entryBlock().NewValue1(src.NoXPos, op, t, arg)
}

// entryNewValue1 adds a new value with one argument and an auxint value to the entry block.
func (s *state) entryNewValue1I(op ssa.Op, t *types.Type, auxint int64, arg *ssa.Value) *ssa.Value {
	return s.entryBlock().NewValue1I(src.NoXPos, op, t, auxint, arg)
}

// entryNewValue1A adds a new value with one argument and an aux value to the entry block.
func (s *state) entryNewValue1A(op ssa.Op, t *types.Type, aux ssa.Aux, arg *ssa.Value) *ssa.Value {
	return s.entryBlock().NewValue1A(src.NoXPos, op, t, aux, arg)
}

// entryNewValue2 adds a new value with two arguments to the entry block.
func (s *state) entryNewValue2(op ssa.Op, t *types.Type, arg0, arg1 *ssa.Value) *ssa.Value {
	return s.entryBlock().NewValue2(src.NoXPos, op, t, arg0, arg1)
}

// entryNewValue2A adds a new value with two arguments and an aux value to the entry block.
func (s *state) entryNewValue2A(op ssa.Op, t *types.Type, aux ssa.Aux, arg0, arg1 *ssa.Value) *ssa.Value {
	return s.entryBlock().NewValue2A(src.NoXPos, op, t, aux, arg0, arg1)
}

// const* routines add a new const value to the entry block.
func (s *state) constSlice(t *types.Type) *ssa.Value {
	return s.f.ConstSlice(t)
}
func (s *state) constInterface(t *types.Type) *ssa.Value {
	return s.f.ConstInterface(t)
}
func (s *state) constNil(t *types.Type) *ssa.Value { return s.f.ConstNil(t) }
func (s *state) constEmptyString(t *types.Type) *ssa.Value {
	return s.f.ConstEmptyString(t)
}
func (s *state) constBool(c bool) *ssa.Value {
	return s.f.ConstBool(types.Types[types.TBOOL], c)
}
func (s *state) constInt8(t *types.Type, c int8) *ssa.Value {
	return s.f.ConstInt8(t, c)
}
func (s *state) constInt16(t *types.Type, c int16) *ssa.Value {
	return s.f.ConstInt16(t, c)
}
func (s *state) constInt32(t *types.Type, c int32) *ssa.Value {
	return s.f.ConstInt32(t, c)
}
func (s *state) constInt64(t *types.Type, c int64) *ssa.Value {
	return s.f.ConstInt64(t, c)
}
func (s *state) constFloat32(t *types.Type, c float64) *ssa.Value {
	return s.f.ConstFloat32(t, c)
}
func (s *state) constFloat64(t *types.Type, c float64) *ssa.Value {
	return s.f.ConstFloat64(t, c)
}
func (s *state) constInt(t *types.Type, c int64) *ssa.Value {
	if s.config.PtrSize == 8 {
		return s.constInt64(t, c)
	}
	if int64(int32(c)) != c {
		s.Fatalf("integer constant too big %d", c)
	}
	return s.constInt32(t, int32(c))
}
func (s *state) constOffPtrSP(t *types.Type, c int64) *ssa.Value {
	return s.f.ConstOffPtrSP(t, c, s.sp)
}

// newValueOrSfCall* are wrappers around newValue*, which may create a call to a
// soft-float runtime function instead (when emitting soft-float code).
func (s *state) newValueOrSfCall1(op ssa.Op, t *types.Type, arg *ssa.Value) *ssa.Value {
	if s.softFloat {
		if c, ok := s.sfcall(op, arg); ok {
			return c
		}
	}
	return s.newValue1(op, t, arg)
}
func (s *state) newValueOrSfCall2(op ssa.Op, t *types.Type, arg0, arg1 *ssa.Value) *ssa.Value {
	if s.softFloat {
		if c, ok := s.sfcall(op, arg0, arg1); ok {
			return c
		}
	}
	return s.newValue2(op, t, arg0, arg1)
}

type instrumentKind uint8

const (
	instrumentRead = iota
	instrumentWrite
	instrumentMove
)

func (s *state) instrument(t *types.Type, addr *ssa.Value, kind instrumentKind) {
	s.instrument2(t, addr, nil, kind)
}

// instrumentFields instruments a read/write operation on addr.
// If it is instrumenting for MSAN and t is a struct type, it instruments
// operation for each field, instead of for the whole struct.
func (s *state) instrumentFields(t *types.Type, addr *ssa.Value, kind instrumentKind) {
	if !base.Flag.MSan || !t.IsStruct() {
		s.instrument(t, addr, kind)
		return
	}
	for _, f := range t.Fields().Slice() {
		if f.Sym.IsBlank() {
			continue
		}
		offptr := s.newValue1I(ssa.OpOffPtr, types.NewPtr(f.Type), f.Offset, addr)
		s.instrumentFields(f.Type, offptr, kind)
	}
}

func (s *state) instrumentMove(t *types.Type, dst, src *ssa.Value) {
	if base.Flag.MSan {
		s.instrument2(t, dst, src, instrumentMove)
	} else {
		s.instrument(t, src, instrumentRead)
		s.instrument(t, dst, instrumentWrite)
	}
}

func (s *state) instrument2(t *types.Type, addr, addr2 *ssa.Value, kind instrumentKind) {
	if !s.curfn.InstrumentBody() {
		return
	}

	w := t.Size()
	if w == 0 {
		return // can't race on zero-sized things
	}

	if ssa.IsSanitizerSafeAddr(addr) {
		return
	}

	var fn *obj.LSym
	needWidth := false

	if addr2 != nil && kind != instrumentMove {
		panic("instrument2: non-nil addr2 for non-move instrumentation")
	}

	if base.Flag.MSan {
		switch kind {
		case instrumentRead:
			fn = ir.Syms.Msanread
		case instrumentWrite:
			fn = ir.Syms.Msanwrite
		case instrumentMove:
			fn = ir.Syms.Msanmove
		default:
			panic("unreachable")
		}
		needWidth = true
	} else if base.Flag.Race && t.NumComponents(types.CountBlankFields) > 1 {
		// for composite objects we have to write every address
		// because a write might happen to any subobject.
		// composites with only one element don't have subobjects, though.
		switch kind {
		case instrumentRead:
			fn = ir.Syms.Racereadrange
		case instrumentWrite:
			fn = ir.Syms.Racewriterange
		default:
			panic("unreachable")
		}
		needWidth = true
	} else if base.Flag.Race {
		// for non-composite objects we can write just the start
		// address, as any write must write the first byte.
		switch kind {
		case instrumentRead:
			fn = ir.Syms.Raceread
		case instrumentWrite:
			fn = ir.Syms.Racewrite
		default:
			panic("unreachable")
		}
	} else {
		panic("unreachable")
	}

	args := []*ssa.Value{addr}
	if addr2 != nil {
		args = append(args, addr2)
	}
	if needWidth {
		args = append(args, s.constInt(types.Types[types.TUINTPTR], w))
	}
	s.rtcall(fn, true, nil, args...)
}

func (s *state) load(t *types.Type, src *ssa.Value) *ssa.Value {
	s.instrumentFields(t, src, instrumentRead)
	return s.rawLoad(t, src)
}

func (s *state) rawLoad(t *types.Type, src *ssa.Value) *ssa.Value {
	return s.newValue2(ssa.OpLoad, t, src, s.mem())
}

func (s *state) store(t *types.Type, dst, val *ssa.Value) {
	s.vars[memVar] = s.newValue3A(ssa.OpStore, types.TypeMem, t, dst, val, s.mem())
}

func (s *state) zero(t *types.Type, dst *ssa.Value) {
	s.instrument(t, dst, instrumentWrite)
	store := s.newValue2I(ssa.OpZero, types.TypeMem, t.Size(), dst, s.mem())
	store.Aux = t
	s.vars[memVar] = store
}

func (s *state) move(t *types.Type, dst, src *ssa.Value) {
	s.instrumentMove(t, dst, src)
	store := s.newValue3I(ssa.OpMove, types.TypeMem, t.Size(), dst, src, s.mem())
	store.Aux = t
	s.vars[memVar] = store
}

// stmtList converts the statement list n to SSA and adds it to s.
func (s *state) stmtList(l ir.Nodes) {
	for _, n := range l {
		s.stmt(n)
	}
}

// stmt converts the statement n to SSA and adds it to s.
func (s *state) stmt(n ir.Node) {
	if !(n.Op() == ir.OVARKILL || n.Op() == ir.OVARLIVE || n.Op() == ir.OVARDEF) {
		// OVARKILL, OVARLIVE, and OVARDEF are invisible to the programmer, so we don't use their line numbers to avoid confusion in debugging.
		s.pushLine(n.Pos())
		defer s.popLine()
	}

	// If s.curBlock is nil, and n isn't a label (which might have an associated goto somewhere),
	// then this code is dead. Stop here.
	if s.curBlock == nil && n.Op() != ir.OLABEL {
		return
	}

	s.stmtList(n.Init())
	switch n.Op() {

	case ir.OBLOCK:
		n := n.(*ir.BlockStmt)
		s.stmtList(n.List)

	// No-ops
	case ir.ODCLCONST, ir.ODCLTYPE, ir.OFALL:

	// Expression statements
	case ir.OCALLFUNC:
		n := n.(*ir.CallExpr)
		if ir.IsIntrinsicCall(n) {
			s.intrinsicCall(n)
			return
		}
		fallthrough

	case ir.OCALLINTER:
		n := n.(*ir.CallExpr)
		s.callResult(n, callNormal)
		if n.Op() == ir.OCALLFUNC && n.X.Op() == ir.ONAME && n.X.(*ir.Name).Class == ir.PFUNC {
			if fn := n.X.Sym().Name; base.Flag.CompilingRuntime && fn == "throw" ||
				n.X.Sym().Pkg == ir.Pkgs.Runtime && (fn == "throwinit" || fn == "gopanic" || fn == "panicwrap" || fn == "block" || fn == "panicmakeslicelen" || fn == "panicmakeslicecap") {
				m := s.mem()
				b := s.endBlock()
				b.Kind = ssa.BlockExit
				b.SetControl(m)
				// TODO: never rewrite OPANIC to OCALLFUNC in the
				// first place. Need to wait until all backends
				// go through SSA.
			}
		}
	case ir.ODEFER:
		n := n.(*ir.GoDeferStmt)
		if base.Debug.Defer > 0 {
			var defertype string
			if s.hasOpenDefers {
				defertype = "open-coded"
			} else if n.Esc() == ir.EscNever {
				defertype = "stack-allocated"
			} else {
				defertype = "heap-allocated"
			}
			base.WarnfAt(n.Pos(), "%s defer", defertype)
		}
		if s.hasOpenDefers {
			s.openDeferRecord(n.Call.(*ir.CallExpr))
		} else {
			d := callDefer
			if n.Esc() == ir.EscNever {
				d = callDeferStack
			}
			s.callResult(n.Call.(*ir.CallExpr), d)
		}
	case ir.OGO:
		n := n.(*ir.GoDeferStmt)
		s.callResult(n.Call.(*ir.CallExpr), callGo)

	case ir.OAS2DOTTYPE:
		n := n.(*ir.AssignListStmt)
		res, resok := s.dottype(n.Rhs[0].(*ir.TypeAssertExpr), true)
		deref := false
		if !TypeOK(n.Rhs[0].Type()) {
			if res.Op != ssa.OpLoad {
				s.Fatalf("dottype of non-load")
			}
			mem := s.mem()
			if mem.Op == ssa.OpVarKill {
				mem = mem.Args[0]
			}
			if res.Args[1] != mem {
				s.Fatalf("memory no longer live from 2-result dottype load")
			}
			deref = true
			res = res.Args[0]
		}
		s.assign(n.Lhs[0], res, deref, 0)
		s.assign(n.Lhs[1], resok, false, 0)
		return

	case ir.OAS2FUNC:
		// We come here only when it is an intrinsic call returning two values.
		n := n.(*ir.AssignListStmt)
		call := n.Rhs[0].(*ir.CallExpr)
		if !ir.IsIntrinsicCall(call) {
			s.Fatalf("non-intrinsic AS2FUNC not expanded %v", call)
		}
		v := s.intrinsicCall(call)
		v1 := s.newValue1(ssa.OpSelect0, n.Lhs[0].Type(), v)
		v2 := s.newValue1(ssa.OpSelect1, n.Lhs[1].Type(), v)
		s.assign(n.Lhs[0], v1, false, 0)
		s.assign(n.Lhs[1], v2, false, 0)
		return

	case ir.ODCL:
		n := n.(*ir.Decl)
		if v := n.X; v.Esc() == ir.EscHeap {
			s.newHeapaddr(v)
		}

	case ir.OLABEL:
		n := n.(*ir.LabelStmt)
		sym := n.Label
		lab := s.label(sym)

		// The label might already have a target block via a goto.
		if lab.target == nil {
			lab.target = s.f.NewBlock(ssa.BlockPlain)
		}

		// Go to that label.
		// (We pretend "label:" is preceded by "goto label", unless the predecessor is unreachable.)
		if s.curBlock != nil {
			b := s.endBlock()
			b.AddEdgeTo(lab.target)
		}
		s.startBlock(lab.target)

	case ir.OGOTO:
		n := n.(*ir.BranchStmt)
		sym := n.Label

		lab := s.label(sym)
		if lab.target == nil {
			lab.target = s.f.NewBlock(ssa.BlockPlain)
		}

		b := s.endBlock()
		b.Pos = s.lastPos.WithIsStmt() // Do this even if b is an empty block.
		b.AddEdgeTo(lab.target)

	case ir.OAS:
		n := n.(*ir.AssignStmt)
		if n.X == n.Y && n.X.Op() == ir.ONAME {
			// An x=x assignment. No point in doing anything
			// here. In addition, skipping this assignment
			// prevents generating:
			//   VARDEF x
			//   COPY x -> x
			// which is bad because x is incorrectly considered
			// dead before the vardef. See issue #14904.
			return
		}

		// Evaluate RHS.
		rhs := n.Y
		if rhs != nil {
			switch rhs.Op() {
			case ir.OSTRUCTLIT, ir.OARRAYLIT, ir.OSLICELIT:
				// All literals with nonzero fields have already been
				// rewritten during walk. Any that remain are just T{}
				// or equivalents. Use the zero value.
				if !ir.IsZero(rhs) {
					s.Fatalf("literal with nonzero value in SSA: %v", rhs)
				}
				rhs = nil
			case ir.OAPPEND:
				rhs := rhs.(*ir.CallExpr)
				// Check whether we're writing the result of an append back to the same slice.
				// If so, we handle it specially to avoid write barriers on the fast
				// (non-growth) path.
				if !ir.SameSafeExpr(n.X, rhs.Args[0]) || base.Flag.N != 0 {
					break
				}
				// If the slice can be SSA'd, it'll be on the stack,
				// so there will be no write barriers,
				// so there's no need to attempt to prevent them.
				if s.canSSA(n.X) {
					if base.Debug.Append > 0 { // replicating old diagnostic message
						base.WarnfAt(n.Pos(), "append: len-only update (in local slice)")
					}
					break
				}
				if base.Debug.Append > 0 {
					base.WarnfAt(n.Pos(), "append: len-only update")
				}
				s.append(rhs, true)
				return
			}
		}

		if ir.IsBlank(n.X) {
			// _ = rhs
			// Just evaluate rhs for side-effects.
			if rhs != nil {
				s.expr(rhs)
			}
			return
		}

		var t *types.Type
		if n.Y != nil {
			t = n.Y.Type()
		} else {
			t = n.X.Type()
		}

		var r *ssa.Value
		deref := !TypeOK(t)
		if deref {
			if rhs == nil {
				r = nil // Signal assign to use OpZero.
			} else {
				r = s.addr(rhs)
			}
		} else {
			if rhs == nil {
				r = s.zeroVal(t)
			} else {
				r = s.expr(rhs)
			}
		}

		var skip skipMask
		if rhs != nil && (rhs.Op() == ir.OSLICE || rhs.Op() == ir.OSLICE3 || rhs.Op() == ir.OSLICESTR) && ir.SameSafeExpr(rhs.(*ir.SliceExpr).X, n.X) {
			// We're assigning a slicing operation back to its source.
			// Don't write back fields we aren't changing. See issue #14855.
			rhs := rhs.(*ir.SliceExpr)
			i, j, k := rhs.Low, rhs.High, rhs.Max
			if i != nil && (i.Op() == ir.OLITERAL && i.Val().Kind() == constant.Int && ir.Int64Val(i) == 0) {
				// [0:...] is the same as [:...]
				i = nil
			}
			// TODO: detect defaults for len/cap also.
			// Currently doesn't really work because (*p)[:len(*p)] appears here as:
			//    tmp = len(*p)
			//    (*p)[:tmp]
			//if j != nil && (j.Op == OLEN && SameSafeExpr(j.Left, n.Left)) {
			//      j = nil
			//}
			//if k != nil && (k.Op == OCAP && SameSafeExpr(k.Left, n.Left)) {
			//      k = nil
			//}
			if i == nil {
				skip |= skipPtr
				if j == nil {
					skip |= skipLen
				}
				if k == nil {
					skip |= skipCap
				}
			}
		}

		s.assign(n.X, r, deref, skip)

	case ir.OIF:
		n := n.(*ir.IfStmt)
		if ir.IsConst(n.Cond, constant.Bool) {
			s.stmtList(n.Cond.Init())
			if ir.BoolVal(n.Cond) {
				s.stmtList(n.Body)
			} else {
				s.stmtList(n.Else)
			}
			break
		}

		bEnd := s.f.NewBlock(ssa.BlockPlain)
		var likely int8
		if n.Likely {
			likely = 1
		}
		var bThen *ssa.Block
		if len(n.Body) != 0 {
			bThen = s.f.NewBlock(ssa.BlockPlain)
		} else {
			bThen = bEnd
		}
		var bElse *ssa.Block
		if len(n.Else) != 0 {
			bElse = s.f.NewBlock(ssa.BlockPlain)
		} else {
			bElse = bEnd
		}
		s.condBranch(n.Cond, bThen, bElse, likely)

		if len(n.Body) != 0 {
			s.startBlock(bThen)
			s.stmtList(n.Body)
			if b := s.endBlock(); b != nil {
				b.AddEdgeTo(bEnd)
			}
		}
		if len(n.Else) != 0 {
			s.startBlock(bElse)
			s.stmtList(n.Else)
			if b := s.endBlock(); b != nil {
				b.AddEdgeTo(bEnd)
			}
		}
		s.startBlock(bEnd)

	case ir.ORETURN:
		n := n.(*ir.ReturnStmt)
		s.stmtList(n.Results)
		b := s.exit()
		b.Pos = s.lastPos.WithIsStmt()

	case ir.OTAILCALL:
		n := n.(*ir.TailCallStmt)
		b := s.exit()
		b.Kind = ssa.BlockRetJmp // override BlockRet
		b.Aux = callTargetLSym(n.Target)

	case ir.OCONTINUE, ir.OBREAK:
		n := n.(*ir.BranchStmt)
		var to *ssa.Block
		if n.Label == nil {
			// plain break/continue
			switch n.Op() {
			case ir.OCONTINUE:
				to = s.continueTo
			case ir.OBREAK:
				to = s.breakTo
			}
		} else {
			// labeled break/continue; look up the target
			sym := n.Label
			lab := s.label(sym)
			switch n.Op() {
			case ir.OCONTINUE:
				to = lab.continueTarget
			case ir.OBREAK:
				to = lab.breakTarget
			}
		}

		b := s.endBlock()
		b.Pos = s.lastPos.WithIsStmt() // Do this even if b is an empty block.
		b.AddEdgeTo(to)

	case ir.OFOR, ir.OFORUNTIL:
		// OFOR: for Ninit; Left; Right { Nbody }
		// cond (Left); body (Nbody); incr (Right)
		//
		// OFORUNTIL: for Ninit; Left; Right; List { Nbody }
		// => body: { Nbody }; incr: Right; if Left { lateincr: List; goto body }; end:
		n := n.(*ir.ForStmt)
		bCond := s.f.NewBlock(ssa.BlockPlain)
		bBody := s.f.NewBlock(ssa.BlockPlain)
		bIncr := s.f.NewBlock(ssa.BlockPlain)
		bEnd := s.f.NewBlock(ssa.BlockPlain)

		// ensure empty for loops have correct position; issue #30167
		bBody.Pos = n.Pos()

		// first, jump to condition test (OFOR) or body (OFORUNTIL)
		b := s.endBlock()
		if n.Op() == ir.OFOR {
			b.AddEdgeTo(bCond)
			// generate code to test condition
			s.startBlock(bCond)
			if n.Cond != nil {
				s.condBranch(n.Cond, bBody, bEnd, 1)
			} else {
				b := s.endBlock()
				b.Kind = ssa.BlockPlain
				b.AddEdgeTo(bBody)
			}

		} else {
			b.AddEdgeTo(bBody)
		}

		// set up for continue/break in body
		prevContinue := s.continueTo
		prevBreak := s.breakTo
		s.continueTo = bIncr
		s.breakTo = bEnd
		var lab *ssaLabel
		if sym := n.Label; sym != nil {
			// labeled for loop
			lab = s.label(sym)
			lab.continueTarget = bIncr
			lab.breakTarget = bEnd
		}

		// generate body
		s.startBlock(bBody)
		s.stmtList(n.Body)

		// tear down continue/break
		s.continueTo = prevContinue
		s.breakTo = prevBreak
		if lab != nil {
			lab.continueTarget = nil
			lab.breakTarget = nil
		}

		// done with body, goto incr
		if b := s.endBlock(); b != nil {
			b.AddEdgeTo(bIncr)
		}

		// generate incr (and, for OFORUNTIL, condition)
		s.startBlock(bIncr)
		if n.Post != nil {
			s.stmt(n.Post)
		}
		if n.Op() == ir.OFOR {
			if b := s.endBlock(); b != nil {
				b.AddEdgeTo(bCond)
				// It can happen that bIncr ends in a block containing only VARKILL,
				// and that muddles the debugging experience.
				if b.Pos == src.NoXPos {
					b.Pos = bCond.Pos
				}
			}
		} else {
			// bCond is unused in OFORUNTIL, so repurpose it.
			bLateIncr := bCond
			// test condition
			s.condBranch(n.Cond, bLateIncr, bEnd, 1)
			// generate late increment
			s.startBlock(bLateIncr)
			s.stmtList(n.Late)
			s.endBlock().AddEdgeTo(bBody)
		}

		s.startBlock(bEnd)

	case ir.OSWITCH, ir.OSELECT:
		// These have been mostly rewritten by the front end into their Nbody fields.
		// Our main task is to correctly hook up any break statements.
		bEnd := s.f.NewBlock(ssa.BlockPlain)

		prevBreak := s.breakTo
		s.breakTo = bEnd
		var sym *types.Sym
		var body ir.Nodes
		if n.Op() == ir.OSWITCH {
			n := n.(*ir.SwitchStmt)
			sym = n.Label
			body = n.Compiled
		} else {
			n := n.(*ir.SelectStmt)
			sym = n.Label
			body = n.Compiled
		}

		var lab *ssaLabel
		if sym != nil {
			// labeled
			lab = s.label(sym)
			lab.breakTarget = bEnd
		}

		// generate body code
		s.stmtList(body)

		s.breakTo = prevBreak
		if lab != nil {
			lab.breakTarget = nil
		}

		// walk adds explicit OBREAK nodes to the end of all reachable code paths.
		// If we still have a current block here, then mark it unreachable.
		if s.curBlock != nil {
			m := s.mem()
			b := s.endBlock()
			b.Kind = ssa.BlockExit
			b.SetControl(m)
		}
		s.startBlock(bEnd)

	case ir.OVARDEF:
		n := n.(*ir.UnaryExpr)
		if !s.canSSA(n.X) {
			s.vars[memVar] = s.newValue1Apos(ssa.OpVarDef, types.TypeMem, n.X.(*ir.Name), s.mem(), false)
		}
	case ir.OVARKILL:
		// Insert a varkill op to record that a variable is no longer live.
		// We only care about liveness info at call sites, so putting the
		// varkill in the store chain is enough to keep it correctly ordered
		// with respect to call ops.
		n := n.(*ir.UnaryExpr)
		if !s.canSSA(n.X) {
			s.vars[memVar] = s.newValue1Apos(ssa.OpVarKill, types.TypeMem, n.X.(*ir.Name), s.mem(), false)
		}

	case ir.OVARLIVE:
		// Insert a varlive op to record that a variable is still live.
		n := n.(*ir.UnaryExpr)
		v := n.X.(*ir.Name)
		if !v.Addrtaken() {
			s.Fatalf("VARLIVE variable %v must have Addrtaken set", v)
		}
		switch v.Class {
		case ir.PAUTO, ir.PPARAM, ir.PPARAMOUT:
		default:
			s.Fatalf("VARLIVE variable %v must be Auto or Arg", v)
		}
		s.vars[memVar] = s.newValue1A(ssa.OpVarLive, types.TypeMem, v, s.mem())

	case ir.OCHECKNIL:
		n := n.(*ir.UnaryExpr)
		p := s.expr(n.X)
		s.nilCheck(p)

	case ir.OINLMARK:
		n := n.(*ir.InlineMarkStmt)
		s.newValue1I(ssa.OpInlMark, types.TypeVoid, n.Index, s.mem())

	default:
		s.Fatalf("unhandled stmt %v", n.Op())
	}
}

// If true, share as many open-coded defer exits as possible (with the downside of
// worse line-number information)
const shareDeferExits = false

// exit processes any code that needs to be generated just before returning.
// It returns a BlockRet block that ends the control flow. Its control value
// will be set to the final memory state.
func (s *state) exit() *ssa.Block {
	if s.hasdefer {
		if s.hasOpenDefers {
			if shareDeferExits && s.lastDeferExit != nil && len(s.openDefers) == s.lastDeferCount {
				if s.curBlock.Kind != ssa.BlockPlain {
					panic("Block for an exit should be BlockPlain")
				}
				s.curBlock.AddEdgeTo(s.lastDeferExit)
				s.endBlock()
				return s.lastDeferFinalBlock
			}
			s.openDeferExit()
		} else {
			s.rtcall(ir.Syms.Deferreturn, true, nil)
		}
	}

	var b *ssa.Block
	var m *ssa.Value
	// Do actual return.
	// These currently turn into self-copies (in many cases).
	resultFields := s.curfn.Type().Results().FieldSlice()
	results := make([]*ssa.Value, len(resultFields)+1, len(resultFields)+1)
	m = s.newValue0(ssa.OpMakeResult, s.f.OwnAux.LateExpansionResultType())
	// Store SSAable and heap-escaped PPARAMOUT variables back to stack locations.
	for i, f := range resultFields {
		n := f.Nname.(*ir.Name)
		if s.canSSA(n) { // result is in some SSA variable
			if !n.IsOutputParamInRegisters() {
				// We are about to store to the result slot.
				s.vars[memVar] = s.newValue1A(ssa.OpVarDef, types.TypeMem, n, s.mem())
			}
			results[i] = s.variable(n, n.Type())
		} else if !n.OnStack() { // result is actually heap allocated
			// We are about to copy the in-heap result to the result slot.
			s.vars[memVar] = s.newValue1A(ssa.OpVarDef, types.TypeMem, n, s.mem())
			ha := s.expr(n.Heapaddr)
			s.instrumentFields(n.Type(), ha, instrumentRead)
			results[i] = s.newValue2(ssa.OpDereference, n.Type(), ha, s.mem())
		} else { // result is not SSA-able; not escaped, so not on heap, but too large for SSA.
			// Before register ABI this ought to be a self-move, home=dest,
			// With register ABI, it's still a self-move if parameter is on stack (i.e., too big or overflowed)
			// No VarDef, as the result slot is already holding live value.
			results[i] = s.newValue2(ssa.OpDereference, n.Type(), s.addr(n), s.mem())
		}
	}

	// Run exit code. Today, this is just racefuncexit, in -race mode.
	// TODO(register args) this seems risky here with a register-ABI, but not clear it is right to do it earlier either.
	// Spills in register allocation might just fix it.
	s.stmtList(s.curfn.Exit)

	results[len(results)-1] = s.mem()
	m.AddArgs(results...)

	b = s.endBlock()
	b.Kind = ssa.BlockRet
	b.SetControl(m)
	if s.hasdefer && s.hasOpenDefers {
		s.lastDeferFinalBlock = b
	}
	return b
}

type opAndType struct {
	op    ir.Op
	etype types.Kind
}

var opToSSA = map[opAndType]ssa.Op{
	opAndType{ir.OADD, types.TINT8}:    ssa.OpAdd8,
	opAndType{ir.OADD, types.TUINT8}:   ssa.OpAdd8,
	opAndType{ir.OADD, types.TINT16}:   ssa.OpAdd16,
	opAndType{ir.OADD, types.TUINT16}:  ssa.OpAdd16,
	opAndType{ir.OADD, types.TINT32}:   ssa.OpAdd32,
	opAndType{ir.OADD, types.TUINT32}:  ssa.OpAdd32,
	opAndType{ir.OADD, types.TINT64}:   ssa.OpAdd64,
	opAndType{ir.OADD, types.TUINT64}:  ssa.OpAdd64,
	opAndType{ir.OADD, types.TFLOAT32}: ssa.OpAdd32F,
	opAndType{ir.OADD, types.TFLOAT64}: ssa.OpAdd64F,

	opAndType{ir.OSUB, types.TINT8}:    ssa.OpSub8,
	opAndType{ir.OSUB, types.TUINT8}:   ssa.OpSub8,
	opAndType{ir.OSUB, types.TINT16}:   ssa.OpSub16,
	opAndType{ir.OSUB, types.TUINT16}:  ssa.OpSub16,
	opAndType{ir.OSUB, types.TINT32}:   ssa.OpSub32,
	opAndType{ir.OSUB, types.TUINT32}:  ssa.OpSub32,
	opAndType{ir.OSUB, types.TINT64}:   ssa.OpSub64,
	opAndType{ir.OSUB, types.TUINT64}:  ssa.OpSub64,
	opAndType{ir.OSUB, types.TFLOAT32}: ssa.OpSub32F,
	opAndType{ir.OSUB, types.TFLOAT64}: ssa.OpSub64F,

	opAndType{ir.ONOT, types.TBOOL}: ssa.OpNot,

	opAndType{ir.ONEG, types.TINT8}:    ssa.OpNeg8,
	opAndType{ir.ONEG, types.TUINT8}:   ssa.OpNeg8,
	opAndType{ir.ONEG, types.TINT16}:   ssa.OpNeg16,
	opAndType{ir.ONEG, types.TUINT16}:  ssa.OpNeg16,
	opAndType{ir.ONEG, types.TINT32}:   ssa.OpNeg32,
	opAndType{ir.ONEG, types.TUINT32}:  ssa.OpNeg32,
	opAndType{ir.ONEG, types.TINT64}:   ssa.OpNeg64,
	opAndType{ir.ONEG, types.TUINT64}:  ssa.OpNeg64,
	opAndType{ir.ONEG, types.TFLOAT32}: ssa.OpNeg32F,
	opAndType{ir.ONEG, types.TFLOAT64}: ssa.OpNeg64F,

	opAndType{ir.OBITNOT, types.TINT8}:   ssa.OpCom8,
	opAndType{ir.OBITNOT, types.TUINT8}:  ssa.OpCom8,
	opAndType{ir.OBITNOT, types.TINT16}:  ssa.OpCom16,
	opAndType{ir.OBITNOT, types.TUINT16}: ssa.OpCom16,
	opAndType{ir.OBITNOT, types.TINT32}:  ssa.OpCom32,
	opAndType{ir.OBITNOT, types.TUINT32}: ssa.OpCom32,
	opAndType{ir.OBITNOT, types.TINT64}:  ssa.OpCom64,
	opAndType{ir.OBITNOT, types.TUINT64}: ssa.OpCom64,

	opAndType{ir.OIMAG, types.TCOMPLEX64}:  ssa.OpComplexImag,
	opAndType{ir.OIMAG, types.TCOMPLEX128}: ssa.OpComplexImag,
	opAndType{ir.OREAL, types.TCOMPLEX64}:  ssa.OpComplexReal,
	opAndType{ir.OREAL, types.TCOMPLEX128}: ssa.OpComplexReal,

	opAndType{ir.OMUL, types.TINT8}:    ssa.OpMul8,
	opAndType{ir.OMUL, types.TUINT8}:   ssa.OpMul8,
	opAndType{ir.OMUL, types.TINT16}:   ssa.OpMul16,
	opAndType{ir.OMUL, types.TUINT16}:  ssa.OpMul16,
	opAndType{ir.OMUL, types.TINT32}:   ssa.OpMul32,
	opAndType{ir.OMUL, types.TUINT32}:  ssa.OpMul32,
	opAndType{ir.OMUL, types.TINT64}:   ssa.OpMul64,
	opAndType{ir.OMUL, types.TUINT64}:  ssa.OpMul64,
	opAndType{ir.OMUL, types.TFLOAT32}: ssa.OpMul32F,
	opAndType{ir.OMUL, types.TFLOAT64}: ssa.OpMul64F,

	opAndType{ir.ODIV, types.TFLOAT32}: ssa.OpDiv32F,
	opAndType{ir.ODIV, types.TFLOAT64}: ssa.OpDiv64F,

	opAndType{ir.ODIV, types.TINT8}:   ssa.OpDiv8,
	opAndType{ir.ODIV, types.TUINT8}:  ssa.OpDiv8u,
	opAndType{ir.ODIV, types.TINT16}:  ssa.OpDiv16,
	opAndType{ir.ODIV, types.TUINT16}: ssa.OpDiv16u,
	opAndType{ir.ODIV, types.TINT32}:  ssa.OpDiv32,
	opAndType{ir.ODIV, types.TUINT32}: ssa.OpDiv32u,
	opAndType{ir.ODIV, types.TINT64}:  ssa.OpDiv64,
	opAndType{ir.ODIV, types.TUINT64}: ssa.OpDiv64u,

	opAndType{ir.OMOD, types.TINT8}:   ssa.OpMod8,
	opAndType{ir.OMOD, types.TUINT8}:  ssa.OpMod8u,
	opAndType{ir.OMOD, types.TINT16}:  ssa.OpMod16,
	opAndType{ir.OMOD, types.TUINT16}: ssa.OpMod16u,
	opAndType{ir.OMOD, types.TINT32}:  ssa.OpMod32,
	opAndType{ir.OMOD, types.TUINT32}: ssa.OpMod32u,
	opAndType{ir.OMOD, types.TINT64}:  ssa.OpMod64,
	opAndType{ir.OMOD, types.TUINT64}: ssa.OpMod64u,

	opAndType{ir.OAND, types.TINT8}:   ssa.OpAnd8,
	opAndType{ir.OAND, types.TUINT8}:  ssa.OpAnd8,
	opAndType{ir.OAND, types.TINT16}:  ssa.OpAnd16,
	opAndType{ir.OAND, types.TUINT16}: ssa.OpAnd16,
	opAndType{ir.OAND, types.TINT32}:  ssa.OpAnd32,
	opAndType{ir.OAND, types.TUINT32}: ssa.OpAnd32,
	opAndType{ir.OAND, types.TINT64}:  ssa.OpAnd64,
	opAndType{ir.OAND, types.TUINT64}: ssa.OpAnd64,

	opAndType{ir.OOR, types.TINT8}:   ssa.OpOr8,
	opAndType{ir.OOR, types.TUINT8}:  ssa.OpOr8,
	opAndType{ir.OOR, types.TINT16}:  ssa.OpOr16,
	opAndType{ir.OOR, types.TUINT16}: ssa.OpOr16,
	opAndType{ir.OOR, types.TINT32}:  ssa.OpOr32,
	opAndType{ir.OOR, types.TUINT32}: ssa.OpOr32,
	opAndType{ir.OOR, types.TINT64}:  ssa.OpOr64,
	opAndType{ir.OOR, types.TUINT64}: ssa.OpOr64,

	opAndType{ir.OXOR, types.TINT8}:   ssa.OpXor8,
	opAndType{ir.OXOR, types.TUINT8}:  ssa.OpXor8,
	opAndType{ir.OXOR, types.TINT16}:  ssa.OpXor16,
	opAndType{ir.OXOR, types.TUINT16}: ssa.OpXor16,
	opAndType{ir.OXOR, types.TINT32}:  ssa.OpXor32,
	opAndType{ir.OXOR, types.TUINT32}: ssa.OpXor32,
	opAndType{ir.OXOR, types.TINT64}:  ssa.OpXor64,
	opAndType{ir.OXOR, types.TUINT64}: ssa.OpXor64,

	opAndType{ir.OEQ, types.TBOOL}:      ssa.OpEqB,
	opAndType{ir.OEQ, types.TINT8}:      ssa.OpEq8,
	opAndType{ir.OEQ, types.TUINT8}:     ssa.OpEq8,
	opAndType{ir.OEQ, types.TINT16}:     ssa.OpEq16,
	opAndType{ir.OEQ, types.TUINT16}:    ssa.OpEq16,
	opAndType{ir.OEQ, types.TINT32}:     ssa.OpEq32,
	opAndType{ir.OEQ, types.TUINT32}:    ssa.OpEq32,
	opAndType{ir.OEQ, types.TINT64}:     ssa.OpEq64,
	opAndType{ir.OEQ, types.TUINT64}:    ssa.OpEq64,
	opAndType{ir.OEQ, types.TINTER}:     ssa.OpEqInter,
	opAndType{ir.OEQ, types.TSLICE}:     ssa.OpEqSlice,
	opAndType{ir.OEQ, types.TFUNC}:      ssa.OpEqPtr,
	opAndType{ir.OEQ, types.TMAP}:       ssa.OpEqPtr,
	opAndType{ir.OEQ, types.TCHAN}:      ssa.OpEqPtr,
	opAndType{ir.OEQ, types.TPTR}:       ssa.OpEqPtr,
	opAndType{ir.OEQ, types.TUINTPTR}:   ssa.OpEqPtr,
	opAndType{ir.OEQ, types.TUNSAFEPTR}: ssa.OpEqPtr,
	opAndType{ir.OEQ, types.TFLOAT64}:   ssa.OpEq64F,
	opAndType{ir.OEQ, types.TFLOAT32}:   ssa.OpEq32F,

	opAndType{ir.ONE, types.TBOOL}:      ssa.OpNeqB,
	opAndType{ir.ONE, types.TINT8}:      ssa.OpNeq8,
	opAndType{ir.ONE, types.TUINT8}:     ssa.OpNeq8,
	opAndType{ir.ONE, types.TINT16}:     ssa.OpNeq16,
	opAndType{ir.ONE, types.TUINT16}:    ssa.OpNeq16,
	opAndType{ir.ONE, types.TINT32}:     ssa.OpNeq32,
	opAndType{ir.ONE, types.TUINT32}:    ssa.OpNeq32,
	opAndType{ir.ONE, types.TINT64}:     ssa.OpNeq64,
	opAndType{ir.ONE, types.TUINT64}:    ssa.OpNeq64,
	opAndType{ir.ONE, types.TINTER}:     ssa.OpNeqInter,
	opAndType{ir.ONE, types.TSLICE}:     ssa.OpNeqSlice,
	opAndType{ir.ONE, types.TFUNC}:      ssa.OpNeqPtr,
	opAndType{ir.ONE, types.TMAP}:       ssa.OpNeqPtr,
	opAndType{ir.ONE, types.TCHAN}:      ssa.OpNeqPtr,
	opAndType{ir.ONE, types.TPTR}:       ssa.OpNeqPtr,
	opAndType{ir.ONE, types.TUINTPTR}:   ssa.OpNeqPtr,
	opAndType{ir.ONE, types.TUNSAFEPTR}: ssa.OpNeqPtr,
	opAndType{ir.ONE, types.TFLOAT64}:   ssa.OpNeq64F,
	opAndType{ir.ONE, types.TFLOAT32}:   ssa.OpNeq32F,

	opAndType{ir.OLT, types.TINT8}:    ssa.OpLess8,
	opAndType{ir.OLT, types.TUINT8}:   ssa.OpLess8U,
	opAndType{ir.OLT, types.TINT16}:   ssa.OpLess16,
	opAndType{ir.OLT, types.TUINT16}:  ssa.OpLess16U,
	opAndType{ir.OLT, types.TINT32}:   ssa.OpLess32,
	opAndType{ir.OLT, types.TUINT32}:  ssa.OpLess32U,
	opAndType{ir.OLT, types.TINT64}:   ssa.OpLess64,
	opAndType{ir.OLT, types.TUINT64}:  ssa.OpLess64U,
	opAndType{ir.OLT, types.TFLOAT64}: ssa.OpLess64F,
	opAndType{ir.OLT, types.TFLOAT32}: ssa.OpLess32F,

	opAndType{ir.OLE, types.TINT8}:    ssa.OpLeq8,
	opAndType{ir.OLE, types.TUINT8}:   ssa.OpLeq8U,
	opAndType{ir.OLE, types.TINT16}:   ssa.OpLeq16,
	opAndType{ir.OLE, types.TUINT16}:  ssa.OpLeq16U,
	opAndType{ir.OLE, types.TINT32}:   ssa.OpLeq32,
	opAndType{ir.OLE, types.TUINT32}:  ssa.OpLeq32U,
	opAndType{ir.OLE, types.TINT64}:   ssa.OpLeq64,
	opAndType{ir.OLE, types.TUINT64}:  ssa.OpLeq64U,
	opAndType{ir.OLE, types.TFLOAT64}: ssa.OpLeq64F,
	opAndType{ir.OLE, types.TFLOAT32}: ssa.OpLeq32F,
}

func (s *state) concreteEtype(t *types.Type) types.Kind {
	e := t.Kind()
	switch e {
	default:
		return e
	case types.TINT:
		if s.config.PtrSize == 8 {
			return types.TINT64
		}
		return types.TINT32
	case types.TUINT:
		if s.config.PtrSize == 8 {
			return types.TUINT64
		}
		return types.TUINT32
	case types.TUINTPTR:
		if s.config.PtrSize == 8 {
			return types.TUINT64
		}
		return types.TUINT32
	}
}

func (s *state) ssaOp(op ir.Op, t *types.Type) ssa.Op {
	etype := s.concreteEtype(t)
	x, ok := opToSSA[opAndType{op, etype}]
	if !ok {
		s.Fatalf("unhandled binary op %v %s", op, etype)
	}
	return x
}

type opAndTwoTypes struct {
	op     ir.Op
	etype1 types.Kind
	etype2 types.Kind
}

type twoTypes struct {
	etype1 types.Kind
	etype2 types.Kind
}

type twoOpsAndType struct {
	op1              ssa.Op
	op2              ssa.Op
	intermediateType types.Kind
}

var fpConvOpToSSA = map[twoTypes]twoOpsAndType{

	twoTypes{types.TINT8, types.TFLOAT32}:  twoOpsAndType{ssa.OpSignExt8to32, ssa.OpCvt32to32F, types.TINT32},
	twoTypes{types.TINT16, types.TFLOAT32}: twoOpsAndType{ssa.OpSignExt16to32, ssa.OpCvt32to32F, types.TINT32},
	twoTypes{types.TINT32, types.TFLOAT32}: twoOpsAndType{ssa.OpCopy, ssa.OpCvt32to32F, types.TINT32},
	twoTypes{types.TINT64, types.TFLOAT32}: twoOpsAndType{ssa.OpCopy, ssa.OpCvt64to32F, types.TINT64},

	twoTypes{types.TINT8, types.TFLOAT64}:  twoOpsAndType{ssa.OpSignExt8to32, ssa.OpCvt32to64F, types.TINT32},
	twoTypes{types.TINT16, types.TFLOAT64}: twoOpsAndType{ssa.OpSignExt16to32, ssa.OpCvt32to64F, types.TINT32},
	twoTypes{types.TINT32, types.TFLOAT64}: twoOpsAndType{ssa.OpCopy, ssa.OpCvt32to64F, types.TINT32},
	twoTypes{types.TINT64, types.TFLOAT64}: twoOpsAndType{ssa.OpCopy, ssa.OpCvt64to64F, types.TINT64},

	twoTypes{types.TFLOAT32, types.TINT8}:  twoOpsAndType{ssa.OpCvt32Fto32, ssa.OpTrunc32to8, types.TINT32},
	twoTypes{types.TFLOAT32, types.TINT16}: twoOpsAndType{ssa.OpCvt32Fto32, ssa.OpTrunc32to16, types.TINT32},
	twoTypes{types.TFLOAT32, types.TINT32}: twoOpsAndType{ssa.OpCvt32Fto32, ssa.OpCopy, types.TINT32},
	twoTypes{types.TFLOAT32, types.TINT64}: twoOpsAndType{ssa.OpCvt32Fto64, ssa.OpCopy, types.TINT64},

	twoTypes{types.TFLOAT64, types.TINT8}:  twoOpsAndType{ssa.OpCvt64Fto32, ssa.OpTrunc32to8, types.TINT32},
	twoTypes{types.TFLOAT64, types.TINT16}: twoOpsAndType{ssa.OpCvt64Fto32, ssa.OpTrunc32to16, types.TINT32},
	twoTypes{types.TFLOAT64, types.TINT32}: twoOpsAndType{ssa.OpCvt64Fto32, ssa.OpCopy, types.TINT32},
	twoTypes{types.TFLOAT64, types.TINT64}: twoOpsAndType{ssa.OpCvt64Fto64, ssa.OpCopy, types.TINT64},
	// unsigned
	twoTypes{types.TUINT8, types.TFLOAT32}:  twoOpsAndType{ssa.OpZeroExt8to32, ssa.OpCvt32to32F, types.TINT32},
	twoTypes{types.TUINT16, types.TFLOAT32}: twoOpsAndType{ssa.OpZeroExt16to32, ssa.OpCvt32to32F, types.TINT32},
	twoTypes{types.TUINT32, types.TFLOAT32}: twoOpsAndType{ssa.OpZeroExt32to64, ssa.OpCvt64to32F, types.TINT64}, // go wide to dodge unsigned
	twoTypes{types.TUINT64, types.TFLOAT32}: twoOpsAndType{ssa.OpCopy, ssa.OpInvalid, types.TUINT64},            // Cvt64Uto32F, branchy code expansion instead

	twoTypes{types.TUINT8, types.TFLOAT64}:  twoOpsAndType{ssa.OpZeroExt8to32, ssa.OpCvt32to64F, types.TINT32},
	twoTypes{types.TUINT16, types.TFLOAT64}: twoOpsAndType{ssa.OpZeroExt16to32, ssa.OpCvt32to64F, types.TINT32},
	twoTypes{types.TUINT32, types.TFLOAT64}: twoOpsAndType{ssa.OpZeroExt32to64, ssa.OpCvt64to64F, types.TINT64}, // go wide to dodge unsigned
	twoTypes{types.TUINT64, types.TFLOAT64}: twoOpsAndType{ssa.OpCopy, ssa.OpInvalid, types.TUINT64},            // Cvt64Uto64F, branchy code expansion instead

	twoTypes{types.TFLOAT32, types.TUINT8}:  twoOpsAndType{ssa.OpCvt32Fto32, ssa.OpTrunc32to8, types.TINT32},
	twoTypes{types.TFLOAT32, types.TUINT16}: twoOpsAndType{ssa.OpCvt32Fto32, ssa.OpTrunc32to16, types.TINT32},
	twoTypes{types.TFLOAT32, types.TUINT32}: twoOpsAndType{ssa.OpCvt32Fto64, ssa.OpTrunc64to32, types.TINT64}, // go wide to dodge unsigned
	twoTypes{types.TFLOAT32, types.TUINT64}: twoOpsAndType{ssa.OpInvalid, ssa.OpCopy, types.TUINT64},          // Cvt32Fto64U, branchy code expansion instead

	twoTypes{types.TFLOAT64, types.TUINT8}:  twoOpsAndType{ssa.OpCvt64Fto32, ssa.OpTrunc32to8, types.TINT32},
	twoTypes{types.TFLOAT64, types.TUINT16}: twoOpsAndType{ssa.OpCvt64Fto32, ssa.OpTrunc32to16, types.TINT32},
	twoTypes{types.TFLOAT64, types.TUINT32}: twoOpsAndType{ssa.OpCvt64Fto64, ssa.OpTrunc64to32, types.TINT64}, // go wide to dodge unsigned
	twoTypes{types.TFLOAT64, types.TUINT64}: twoOpsAndType{ssa.OpInvalid, ssa.OpCopy, types.TUINT64},          // Cvt64Fto64U, branchy code expansion instead

	// float
	twoTypes{types.TFLOAT64, types.TFLOAT32}: twoOpsAndType{ssa.OpCvt64Fto32F, ssa.OpCopy, types.TFLOAT32},
	twoTypes{types.TFLOAT64, types.TFLOAT64}: twoOpsAndType{ssa.OpRound64F, ssa.OpCopy, types.TFLOAT64},
	twoTypes{types.TFLOAT32, types.TFLOAT32}: twoOpsAndType{ssa.OpRound32F, ssa.OpCopy, types.TFLOAT32},
	twoTypes{types.TFLOAT32, types.TFLOAT64}: twoOpsAndType{ssa.OpCvt32Fto64F, ssa.OpCopy, types.TFLOAT64},
}

// this map is used only for 32-bit arch, and only includes the difference
// on 32-bit arch, don't use int64<->float conversion for uint32
var fpConvOpToSSA32 = map[twoTypes]twoOpsAndType{
	twoTypes{types.TUINT32, types.TFLOAT32}: twoOpsAndType{ssa.OpCopy, ssa.OpCvt32Uto32F, types.TUINT32},
	twoTypes{types.TUINT32, types.TFLOAT64}: twoOpsAndType{ssa.OpCopy, ssa.OpCvt32Uto64F, types.TUINT32},
	twoTypes{types.TFLOAT32, types.TUINT32}: twoOpsAndType{ssa.OpCvt32Fto32U, ssa.OpCopy, types.TUINT32},
	twoTypes{types.TFLOAT64, types.TUINT32}: twoOpsAndType{ssa.OpCvt64Fto32U, ssa.OpCopy, types.TUINT32},
}

// uint64<->float conversions, only on machines that have instructions for that
var uint64fpConvOpToSSA = map[twoTypes]twoOpsAndType{
	twoTypes{types.TUINT64, types.TFLOAT32}: twoOpsAndType{ssa.OpCopy, ssa.OpCvt64Uto32F, types.TUINT64},
	twoTypes{types.TUINT64, types.TFLOAT64}: twoOpsAndType{ssa.OpCopy, ssa.OpCvt64Uto64F, types.TUINT64},
	twoTypes{types.TFLOAT32, types.TUINT64}: twoOpsAndType{ssa.OpCvt32Fto64U, ssa.OpCopy, types.TUINT64},
	twoTypes{types.TFLOAT64, types.TUINT64}: twoOpsAndType{ssa.OpCvt64Fto64U, ssa.OpCopy, types.TUINT64},
}

var shiftOpToSSA = map[opAndTwoTypes]ssa.Op{
	opAndTwoTypes{ir.OLSH, types.TINT8, types.TUINT8}:   ssa.OpLsh8x8,
	opAndTwoTypes{ir.OLSH, types.TUINT8, types.TUINT8}:  ssa.OpLsh8x8,
	opAndTwoTypes{ir.OLSH, types.TINT8, types.TUINT16}:  ssa.OpLsh8x16,
	opAndTwoTypes{ir.OLSH, types.TUINT8, types.TUINT16}: ssa.OpLsh8x16,
	opAndTwoTypes{ir.OLSH, types.TINT8, types.TUINT32}:  ssa.OpLsh8x32,
	opAndTwoTypes{ir.OLSH, types.TUINT8, types.TUINT32}: ssa.OpLsh8x32,
	opAndTwoTypes{ir.OLSH, types.TINT8, types.TUINT64}:  ssa.OpLsh8x64,
	opAndTwoTypes{ir.OLSH, types.TUINT8, types.TUINT64}: ssa.OpLsh8x64,

	opAndTwoTypes{ir.OLSH, types.TINT16, types.TUINT8}:   ssa.OpLsh16x8,
	opAndTwoTypes{ir.OLSH, types.TUINT16, types.TUINT8}:  ssa.OpLsh16x8,
	opAndTwoTypes{ir.OLSH, types.TINT16, types.TUINT16}:  ssa.OpLsh16x16,
	opAndTwoTypes{ir.OLSH, types.TUINT16, types.TUINT16}: ssa.OpLsh16x16,
	opAndTwoTypes{ir.OLSH, types.TINT16, types.TUINT32}:  ssa.OpLsh16x32,
	opAndTwoTypes{ir.OLSH, types.TUINT16, types.TUINT32}: ssa.OpLsh16x32,
	opAndTwoTypes{ir.OLSH, types.TINT16, types.TUINT64}:  ssa.OpLsh16x64,
	opAndTwoTypes{ir.OLSH, types.TUINT16, types.TUINT64}: ssa.OpLsh16x64,

	opAndTwoTypes{ir.OLSH, types.TINT32, types.TUINT8}:   ssa.OpLsh32x8,
	opAndTwoTypes{ir.OLSH, types.TUINT32, types.TUINT8}:  ssa.OpLsh32x8,
	opAndTwoTypes{ir.OLSH, types.TINT32, types.TUINT16}:  ssa.OpLsh32x16,
	opAndTwoTypes{ir.OLSH, types.TUINT32, types.TUINT16}: ssa.OpLsh32x16,
	opAndTwoTypes{ir.OLSH, types.TINT32, types.TUINT32}:  ssa.OpLsh32x32,
	opAndTwoTypes{ir.OLSH, types.TUINT32, types.TUINT32}: ssa.OpLsh32x32,
	opAndTwoTypes{ir.OLSH, types.TINT32, types.TUINT64}:  ssa.OpLsh32x64,
	opAndTwoTypes{ir.OLSH, types.TUINT32, types.TUINT64}: ssa.OpLsh32x64,

	opAndTwoTypes{ir.OLSH, types.TINT64, types.TUINT8}:   ssa.OpLsh64x8,
	opAndTwoTypes{ir.OLSH, types.TUINT64, types.TUINT8}:  ssa.OpLsh64x8,
	opAndTwoTypes{ir.OLSH, types.TINT64, types.TUINT16}:  ssa.OpLsh64x16,
	opAndTwoTypes{ir.OLSH, types.TUINT64, types.TUINT16}: ssa.OpLsh64x16,
	opAndTwoTypes{ir.OLSH, types.TINT64, types.TUINT32}:  ssa.OpLsh64x32,
	opAndTwoTypes{ir.OLSH, types.TUINT64, types.TUINT32}: ssa.OpLsh64x32,
	opAndTwoTypes{ir.OLSH, types.TINT64, types.TUINT64}:  ssa.OpLsh64x64,
	opAndTwoTypes{ir.OLSH, types.TUINT64, types.TUINT64}: ssa.OpLsh64x64,

	opAndTwoTypes{ir.ORSH, types.TINT8, types.TUINT8}:   ssa.OpRsh8x8,
	opAndTwoTypes{ir.ORSH, types.TUINT8, types.TUINT8}:  ssa.OpRsh8Ux8,
	opAndTwoTypes{ir.ORSH, types.TINT8, types.TUINT16}:  ssa.OpRsh8x16,
	opAndTwoTypes{ir.ORSH, types.TUINT8, types.TUINT16}: ssa.OpRsh8Ux16,
	opAndTwoTypes{ir.ORSH, types.TINT8, types.TUINT32}:  ssa.OpRsh8x32,
	opAndTwoTypes{ir.ORSH, types.TUINT8, types.TUINT32}: ssa.OpRsh8Ux32,
	opAndTwoTypes{ir.ORSH, types.TINT8, types.TUINT64}:  ssa.OpRsh8x64,
	opAndTwoTypes{ir.ORSH, types.TUINT8, types.TUINT64}: ssa.OpRsh8Ux64,

	opAndTwoTypes{ir.ORSH, types.TINT16, types.TUINT8}:   ssa.OpRsh16x8,
	opAndTwoTypes{ir.ORSH, types.TUINT16, types.TUINT8}:  ssa.OpRsh16Ux8,
	opAndTwoTypes{ir.ORSH, types.TINT16, types.TUINT16}:  ssa.OpRsh16x16,
	opAndTwoTypes{ir.ORSH, types.TUINT16, types.TUINT16}: ssa.OpRsh16Ux16,
	opAndTwoTypes{ir.ORSH, types.TINT16, types.TUINT32}:  ssa.OpRsh16x32,
	opAndTwoTypes{ir.ORSH, types.TUINT16, types.TUINT32}: ssa.OpRsh16Ux32,
	opAndTwoTypes{ir.ORSH, types.TINT16, types.TUINT64}:  ssa.OpRsh16x64,
	opAndTwoTypes{ir.ORSH, types.TUINT16, types.TUINT64}: ssa.OpRsh16Ux64,

	opAndTwoTypes{ir.ORSH, types.TINT32, types.TUINT8}:   ssa.OpRsh32x8,
	opAndTwoTypes{ir.ORSH, types.TUINT32, types.TUINT8}:  ssa.OpRsh32Ux8,
	opAndTwoTypes{ir.ORSH, types.TINT32, types.TUINT16}:  ssa.OpRsh32x16,
	opAndTwoTypes{ir.ORSH, types.TUINT32, types.TUINT16}: ssa.OpRsh32Ux16,
	opAndTwoTypes{ir.ORSH, types.TINT32, types.TUINT32}:  ssa.OpRsh32x32,
	opAndTwoTypes{ir.ORSH, types.TUINT32, types.TUINT32}: ssa.OpRsh32Ux32,
	opAndTwoTypes{ir.ORSH, types.TINT32, types.TUINT64}:  ssa.OpRsh32x64,
	opAndTwoTypes{ir.ORSH, types.TUINT32, types.TUINT64}: ssa.OpRsh32Ux64,

	opAndTwoTypes{ir.ORSH, types.TINT64, types.TUINT8}:   ssa.OpRsh64x8,
	opAndTwoTypes{ir.ORSH, types.TUINT64, types.TUINT8}:  ssa.OpRsh64Ux8,
	opAndTwoTypes{ir.ORSH, types.TINT64, types.TUINT16}:  ssa.OpRsh64x16,
	opAndTwoTypes{ir.ORSH, types.TUINT64, types.TUINT16}: ssa.OpRsh64Ux16,
	opAndTwoTypes{ir.ORSH, types.TINT64, types.TUINT32}:  ssa.OpRsh64x32,
	opAndTwoTypes{ir.ORSH, types.TUINT64, types.TUINT32}: ssa.OpRsh64Ux32,
	opAndTwoTypes{ir.ORSH, types.TINT64, types.TUINT64}:  ssa.OpRsh64x64,
	opAndTwoTypes{ir.ORSH, types.TUINT64, types.TUINT64}: ssa.OpRsh64Ux64,
}

func (s *state) ssaShiftOp(op ir.Op, t *types.Type, u *types.Type) ssa.Op {
	etype1 := s.concreteEtype(t)
	etype2 := s.concreteEtype(u)
	x, ok := shiftOpToSSA[opAndTwoTypes{op, etype1, etype2}]
	if !ok {
		s.Fatalf("unhandled shift op %v etype=%s/%s", op, etype1, etype2)
	}
	return x
}

// expr converts the expression n to ssa, adds it to s and returns the ssa result.
func (s *state) expr(n ir.Node) *ssa.Value {
	if ir.HasUniquePos(n) {
		// ONAMEs and named OLITERALs have the line number
		// of the decl, not the use. See issue 14742.
		s.pushLine(n.Pos())
		defer s.popLine()
	}

	s.stmtList(n.Init())
	switch n.Op() {
	case ir.OBYTES2STRTMP:
		n := n.(*ir.ConvExpr)
		slice := s.expr(n.X)
		ptr := s.newValue1(ssa.OpSlicePtr, s.f.Config.Types.BytePtr, slice)
		len := s.newValue1(ssa.OpSliceLen, types.Types[types.TINT], slice)
		return s.newValue2(ssa.OpStringMake, n.Type(), ptr, len)
	case ir.OSTR2BYTESTMP:
		n := n.(*ir.ConvExpr)
		str := s.expr(n.X)
		ptr := s.newValue1(ssa.OpStringPtr, s.f.Config.Types.BytePtr, str)
		len := s.newValue1(ssa.OpStringLen, types.Types[types.TINT], str)
		return s.newValue3(ssa.OpSliceMake, n.Type(), ptr, len, len)
	case ir.OCFUNC:
		n := n.(*ir.UnaryExpr)
		aux := n.X.(*ir.Name).Linksym()
		// OCFUNC is used to build function values, which must
		// always reference ABIInternal entry points.
		if aux.ABI() != obj.ABIInternal {
			s.Fatalf("expected ABIInternal: %v", aux.ABI())
		}
		return s.entryNewValue1A(ssa.OpAddr, n.Type(), aux, s.sb)
	case ir.ONAME:
		n := n.(*ir.Name)
		if n.Class == ir.PFUNC {
			// "value" of a function is the address of the function's closure
			sym := staticdata.FuncLinksym(n)
			return s.entryNewValue1A(ssa.OpAddr, types.NewPtr(n.Type()), sym, s.sb)
		}
		if s.canSSA(n) {
			return s.variable(n, n.Type())
		}
		return s.load(n.Type(), s.addr(n))
	case ir.OLINKSYMOFFSET:
		n := n.(*ir.LinksymOffsetExpr)
		return s.load(n.Type(), s.addr(n))
	case ir.ONIL:
		n := n.(*ir.NilExpr)
		t := n.Type()
		switch {
		case t.IsSlice():
			return s.constSlice(t)
		case t.IsInterface():
			return s.constInterface(t)
		default:
			return s.constNil(t)
		}
	case ir.OLITERAL:
		switch u := n.Val(); u.Kind() {
		case constant.Int:
			i := ir.IntVal(n.Type(), u)
			switch n.Type().Size() {
			case 1:
				return s.constInt8(n.Type(), int8(i))
			case 2:
				return s.constInt16(n.Type(), int16(i))
			case 4:
				return s.constInt32(n.Type(), int32(i))
			case 8:
				return s.constInt64(n.Type(), i)
			default:
				s.Fatalf("bad integer size %d", n.Type().Size())
				return nil
			}
		case constant.String:
			i := constant.StringVal(u)
			if i == "" {
				return s.constEmptyString(n.Type())
			}
			return s.entryNewValue0A(ssa.OpConstString, n.Type(), ssa.StringToAux(i))
		case constant.Bool:
			return s.constBool(constant.BoolVal(u))
		case constant.Float:
			f, _ := constant.Float64Val(u)
			switch n.Type().Size() {
			case 4:
				return s.constFloat32(n.Type(), f)
			case 8:
				return s.constFloat64(n.Type(), f)
			default:
				s.Fatalf("bad float size %d", n.Type().Size())
				return nil
			}
		case constant.Complex:
			re, _ := constant.Float64Val(constant.Real(u))
			im, _ := constant.Float64Val(constant.Imag(u))
			switch n.Type().Size() {
			case 8:
				pt := types.Types[types.TFLOAT32]
				return s.newValue2(ssa.OpComplexMake, n.Type(),
					s.constFloat32(pt, re),
					s.constFloat32(pt, im))
			case 16:
				pt := types.Types[types.TFLOAT64]
				return s.newValue2(ssa.OpComplexMake, n.Type(),
					s.constFloat64(pt, re),
					s.constFloat64(pt, im))
			default:
				s.Fatalf("bad complex size %d", n.Type().Size())
				return nil
			}
		default:
			s.Fatalf("unhandled OLITERAL %v", u.Kind())
			return nil
		}
	case ir.OCONVNOP:
		n := n.(*ir.ConvExpr)
		to := n.Type()
		from := n.X.Type()

		// Assume everything will work out, so set up our return value.
		// Anything interesting that happens from here is a fatal.
		x := s.expr(n.X)
		if to == from {
			return x
		}

		// Special case for not confusing GC and liveness.
		// We don't want pointers accidentally classified
		// as not-pointers or vice-versa because of copy
		// elision.
		if to.IsPtrShaped() != from.IsPtrShaped() {
			return s.newValue2(ssa.OpConvert, to, x, s.mem())
		}

		v := s.newValue1(ssa.OpCopy, to, x) // ensure that v has the right type

		// CONVNOP closure
		if to.Kind() == types.TFUNC && from.IsPtrShaped() {
			return v
		}

		// named <--> unnamed type or typed <--> untyped const
		if from.Kind() == to.Kind() {
			return v
		}

		// unsafe.Pointer <--> *T
		if to.IsUnsafePtr() && from.IsPtrShaped() || from.IsUnsafePtr() && to.IsPtrShaped() {
			return v
		}

		// map <--> *hmap
		if to.Kind() == types.TMAP && from.IsPtr() &&
			to.MapType().Hmap == from.Elem() {
			return v
		}

		types.CalcSize(from)
		types.CalcSize(to)
		if from.Width != to.Width {
			s.Fatalf("CONVNOP width mismatch %v (%d) -> %v (%d)\n", from, from.Width, to, to.Width)
			return nil
		}
		if etypesign(from.Kind()) != etypesign(to.Kind()) {
			s.Fatalf("CONVNOP sign mismatch %v (%s) -> %v (%s)\n", from, from.Kind(), to, to.Kind())
			return nil
		}

		if base.Flag.Cfg.Instrumenting {
			// These appear to be fine, but they fail the
			// integer constraint below, so okay them here.
			// Sample non-integer conversion: map[string]string -> *uint8
			return v
		}

		if etypesign(from.Kind()) == 0 {
			s.Fatalf("CONVNOP unrecognized non-integer %v -> %v\n", from, to)
			return nil
		}

		// integer, same width, same sign
		return v

	case ir.OCONV:
		n := n.(*ir.ConvExpr)
		x := s.expr(n.X)
		ft := n.X.Type() // from type
		tt := n.Type()   // to type
		if ft.IsBoolean() && tt.IsKind(types.TUINT8) {
			// Bool -> uint8 is generated internally when indexing into runtime.staticbyte.
			return s.newValue1(ssa.OpCopy, n.Type(), x)
		}
		if ft.IsInteger() && tt.IsInteger() {
			var op ssa.Op
			if tt.Size() == ft.Size() {
				op = ssa.OpCopy
			} else if tt.Size() < ft.Size() {
				// truncation
				switch 10*ft.Size() + tt.Size() {
				case 21:
					op = ssa.OpTrunc16to8
				case 41:
					op = ssa.OpTrunc32to8
				case 42:
					op = ssa.OpTrunc32to16
				case 81:
					op = ssa.OpTrunc64to8
				case 82:
					op = ssa.OpTrunc64to16
				case 84:
					op = ssa.OpTrunc64to32
				default:
					s.Fatalf("weird integer truncation %v -> %v", ft, tt)
				}
			} else if ft.IsSigned() {
				// sign extension
				switch 10*ft.Size() + tt.Size() {
				case 12:
					op = ssa.OpSignExt8to16
				case 14:
					op = ssa.OpSignExt8to32
				case 18:
					op = ssa.OpSignExt8to64
				case 24:
					op = ssa.OpSignExt16to32
				case 28:
					op = ssa.OpSignExt16to64
				case 48:
					op = ssa.OpSignExt32to64
				default:
					s.Fatalf("bad integer sign extension %v -> %v", ft, tt)
				}
			} else {
				// zero extension
				switch 10*ft.Size() + tt.Size() {
				case 12:
					op = ssa.OpZeroExt8to16
				case 14:
					op = ssa.OpZeroExt8to32
				case 18:
					op = ssa.OpZeroExt8to64
				case 24:
					op = ssa.OpZeroExt16to32
				case 28:
					op = ssa.OpZeroExt16to64
				case 48:
					op = ssa.OpZeroExt32to64
				default:
					s.Fatalf("weird integer sign extension %v -> %v", ft, tt)
				}
			}
			return s.newValue1(op, n.Type(), x)
		}

		if ft.IsFloat() || tt.IsFloat() {
			conv, ok := fpConvOpToSSA[twoTypes{s.concreteEtype(ft), s.concreteEtype(tt)}]
			if s.config.RegSize == 4 && Arch.LinkArch.Family != sys.MIPS && !s.softFloat {
				if conv1, ok1 := fpConvOpToSSA32[twoTypes{s.concreteEtype(ft), s.concreteEtype(tt)}]; ok1 {
					conv = conv1
				}
			}
			if Arch.LinkArch.Family == sys.ARM64 || Arch.LinkArch.Family == sys.Wasm || Arch.LinkArch.Family == sys.S390X || s.softFloat {
				if conv1, ok1 := uint64fpConvOpToSSA[twoTypes{s.concreteEtype(ft), s.concreteEtype(tt)}]; ok1 {
					conv = conv1
				}
			}

			if Arch.LinkArch.Family == sys.MIPS && !s.softFloat {
				if ft.Size() == 4 && ft.IsInteger() && !ft.IsSigned() {
					// tt is float32 or float64, and ft is also unsigned
					if tt.Size() == 4 {
						return s.uint32Tofloat32(n, x, ft, tt)
					}
					if tt.Size() == 8 {
						return s.uint32Tofloat64(n, x, ft, tt)
					}
				} else if tt.Size() == 4 && tt.IsInteger() && !tt.IsSigned() {
					// ft is float32 or float64, and tt is unsigned integer
					if ft.Size() == 4 {
						return s.float32ToUint32(n, x, ft, tt)
					}
					if ft.Size() == 8 {
						return s.float64ToUint32(n, x, ft, tt)
					}
				}
			}

			if !ok {
				s.Fatalf("weird float conversion %v -> %v", ft, tt)
			}
			op1, op2, it := conv.op1, conv.op2, conv.intermediateType

			if op1 != ssa.OpInvalid && op2 != ssa.OpInvalid {
				// normal case, not tripping over unsigned 64
				if op1 == ssa.OpCopy {
					if op2 == ssa.OpCopy {
						return x
					}
					return s.newValueOrSfCall1(op2, n.Type(), x)
				}
				if op2 == ssa.OpCopy {
					return s.newValueOrSfCall1(op1, n.Type(), x)
				}
				return s.newValueOrSfCall1(op2, n.Type(), s.newValueOrSfCall1(op1, types.Types[it], x))
			}
			// Tricky 64-bit unsigned cases.
			if ft.IsInteger() {
				// tt is float32 or float64, and ft is also unsigned
				if tt.Size() == 4 {
					return s.uint64Tofloat32(n, x, ft, tt)
				}
				if tt.Size() == 8 {
					return s.uint64Tofloat64(n, x, ft, tt)
				}
				s.Fatalf("weird unsigned integer to float conversion %v -> %v", ft, tt)
			}
			// ft is float32 or float64, and tt is unsigned integer
			if ft.Size() == 4 {
				return s.float32ToUint64(n, x, ft, tt)
			}
			if ft.Size() == 8 {
				return s.float64ToUint64(n, x, ft, tt)
			}
			s.Fatalf("weird float to unsigned integer conversion %v -> %v", ft, tt)
			return nil
		}

		if ft.IsComplex() && tt.IsComplex() {
			var op ssa.Op
			if ft.Size() == tt.Size() {
				switch ft.Size() {
				case 8:
					op = ssa.OpRound32F
				case 16:
					op = ssa.OpRound64F
				default:
					s.Fatalf("weird complex conversion %v -> %v", ft, tt)
				}
			} else if ft.Size() == 8 && tt.Size() == 16 {
				op = ssa.OpCvt32Fto64F
			} else if ft.Size() == 16 && tt.Size() == 8 {
				op = ssa.OpCvt64Fto32F
			} else {
				s.Fatalf("weird complex conversion %v -> %v", ft, tt)
			}
			ftp := types.FloatForComplex(ft)
			ttp := types.FloatForComplex(tt)
			return s.newValue2(ssa.OpComplexMake, tt,
				s.newValueOrSfCall1(op, ttp, s.newValue1(ssa.OpComplexReal, ftp, x)),
				s.newValueOrSfCall1(op, ttp, s.newValue1(ssa.OpComplexImag, ftp, x)))
		}

		s.Fatalf("unhandled OCONV %s -> %s", n.X.Type().Kind(), n.Type().Kind())
		return nil

	case ir.ODOTTYPE:
		n := n.(*ir.TypeAssertExpr)
		res, _ := s.dottype(n, false)
		return res

	// binary ops
	case ir.OLT, ir.OEQ, ir.ONE, ir.OLE, ir.OGE, ir.OGT:
		n := n.(*ir.BinaryExpr)
		a := s.expr(n.X)
		b := s.expr(n.Y)
		if n.X.Type().IsComplex() {
			pt := types.FloatForComplex(n.X.Type())
			op := s.ssaOp(ir.OEQ, pt)
			r := s.newValueOrSfCall2(op, types.Types[types.TBOOL], s.newValue1(ssa.OpComplexReal, pt, a), s.newValue1(ssa.OpComplexReal, pt, b))
			i := s.newValueOrSfCall2(op, types.Types[types.TBOOL], s.newValue1(ssa.OpComplexImag, pt, a), s.newValue1(ssa.OpComplexImag, pt, b))
			c := s.newValue2(ssa.OpAndB, types.Types[types.TBOOL], r, i)
			switch n.Op() {
			case ir.OEQ:
				return c
			case ir.ONE:
				return s.newValue1(ssa.OpNot, types.Types[types.TBOOL], c)
			default:
				s.Fatalf("ordered complex compare %v", n.Op())
			}
		}

		// Convert OGE and OGT into OLE and OLT.
		op := n.Op()
		switch op {
		case ir.OGE:
			op, a, b = ir.OLE, b, a
		case ir.OGT:
			op, a, b = ir.OLT, b, a
		}
		if n.X.Type().IsFloat() {
			// float comparison
			return s.newValueOrSfCall2(s.ssaOp(op, n.X.Type()), types.Types[types.TBOOL], a, b)
		}
		// integer comparison
		return s.newValue2(s.ssaOp(op, n.X.Type()), types.Types[types.TBOOL], a, b)
	case ir.OMUL:
		n := n.(*ir.BinaryExpr)
		a := s.expr(n.X)
		b := s.expr(n.Y)
		if n.Type().IsComplex() {
			mulop := ssa.OpMul64F
			addop := ssa.OpAdd64F
			subop := ssa.OpSub64F
			pt := types.FloatForComplex(n.Type()) // Could be Float32 or Float64
			wt := types.Types[types.TFLOAT64]     // Compute in Float64 to minimize cancellation error

			areal := s.newValue1(ssa.OpComplexReal, pt, a)
			breal := s.newValue1(ssa.OpComplexReal, pt, b)
			aimag := s.newValue1(ssa.OpComplexImag, pt, a)
			bimag := s.newValue1(ssa.OpComplexImag, pt, b)

			if pt != wt { // Widen for calculation
				areal = s.newValueOrSfCall1(ssa.OpCvt32Fto64F, wt, areal)
				breal = s.newValueOrSfCall1(ssa.OpCvt32Fto64F, wt, breal)
				aimag = s.newValueOrSfCall1(ssa.OpCvt32Fto64F, wt, aimag)
				bimag = s.newValueOrSfCall1(ssa.OpCvt32Fto64F, wt, bimag)
			}

			xreal := s.newValueOrSfCall2(subop, wt, s.newValueOrSfCall2(mulop, wt, areal, breal), s.newValueOrSfCall2(mulop, wt, aimag, bimag))
			ximag := s.newValueOrSfCall2(addop, wt, s.newValueOrSfCall2(mulop, wt, areal, bimag), s.newValueOrSfCall2(mulop, wt, aimag, breal))

			if pt != wt { // Narrow to store back
				xreal = s.newValueOrSfCall1(ssa.OpCvt64Fto32F, pt, xreal)
				ximag = s.newValueOrSfCall1(ssa.OpCvt64Fto32F, pt, ximag)
			}

			return s.newValue2(ssa.OpComplexMake, n.Type(), xreal, ximag)
		}

		if n.Type().IsFloat() {
			return s.newValueOrSfCall2(s.ssaOp(n.Op(), n.Type()), a.Type, a, b)
		}

		return s.newValue2(s.ssaOp(n.Op(), n.Type()), a.Type, a, b)

	case ir.ODIV:
		n := n.(*ir.BinaryExpr)
		a := s.expr(n.X)
		b := s.expr(n.Y)
		if n.Type().IsComplex() {
			// TODO this is not executed because the front-end substitutes a runtime call.
			// That probably ought to change; with modest optimization the widen/narrow
			// conversions could all be elided in larger expression trees.
			mulop := ssa.OpMul64F
			addop := ssa.OpAdd64F
			subop := ssa.OpSub64F
			divop := ssa.OpDiv64F
			pt := types.FloatForComplex(n.Type()) // Could be Float32 or Float64
			wt := types.Types[types.TFLOAT64]     // Compute in Float64 to minimize cancellation error

			areal := s.newValue1(ssa.OpComplexReal, pt, a)
			breal := s.newValue1(ssa.OpComplexReal, pt, b)
			aimag := s.newValue1(ssa.OpComplexImag, pt, a)
			bimag := s.newValue1(ssa.OpComplexImag, pt, b)

			if pt != wt { // Widen for calculation
				areal = s.newValueOrSfCall1(ssa.OpCvt32Fto64F, wt, areal)
				breal = s.newValueOrSfCall1(ssa.OpCvt32Fto64F, wt, breal)
				aimag = s.newValueOrSfCall1(ssa.OpCvt32Fto64F, wt, aimag)
				bimag = s.newValueOrSfCall1(ssa.OpCvt32Fto64F, wt, bimag)
			}

			denom := s.newValueOrSfCall2(addop, wt, s.newValueOrSfCall2(mulop, wt, breal, breal), s.newValueOrSfCall2(mulop, wt, bimag, bimag))
			xreal := s.newValueOrSfCall2(addop, wt, s.newValueOrSfCall2(mulop, wt, areal, breal), s.newValueOrSfCall2(mulop, wt, aimag, bimag))
			ximag := s.newValueOrSfCall2(subop, wt, s.newValueOrSfCall2(mulop, wt, aimag, breal), s.newValueOrSfCall2(mulop, wt, areal, bimag))

			// TODO not sure if this is best done in wide precision or narrow
			// Double-rounding might be an issue.
			// Note that the pre-SSA implementation does the entire calculation
			// in wide format, so wide is compatible.
			xreal = s.newValueOrSfCall2(divop, wt, xreal, denom)
			ximag = s.newValueOrSfCall2(divop, wt, ximag, denom)

			if pt != wt { // Narrow to store back
				xreal = s.newValueOrSfCall1(ssa.OpCvt64Fto32F, pt, xreal)
				ximag = s.newValueOrSfCall1(ssa.OpCvt64Fto32F, pt, ximag)
			}
			return s.newValue2(ssa.OpComplexMake, n.Type(), xreal, ximag)
		}
		if n.Type().IsFloat() {
			return s.newValueOrSfCall2(s.ssaOp(n.Op(), n.Type()), a.Type, a, b)
		}
		return s.intDivide(n, a, b)
	case ir.OMOD:
		n := n.(*ir.BinaryExpr)
		a := s.expr(n.X)
		b := s.expr(n.Y)
		return s.intDivide(n, a, b)
	case ir.OADD, ir.OSUB:
		n := n.(*ir.BinaryExpr)
		a := s.expr(n.X)
		b := s.expr(n.Y)
		if n.Type().IsComplex() {
			pt := types.FloatForComplex(n.Type())
			op := s.ssaOp(n.Op(), pt)
			return s.newValue2(ssa.OpComplexMake, n.Type(),
				s.newValueOrSfCall2(op, pt, s.newValue1(ssa.OpComplexReal, pt, a), s.newValue1(ssa.OpComplexReal, pt, b)),
				s.newValueOrSfCall2(op, pt, s.newValue1(ssa.OpComplexImag, pt, a), s.newValue1(ssa.OpComplexImag, pt, b)))
		}
		if n.Type().IsFloat() {
			return s.newValueOrSfCall2(s.ssaOp(n.Op(), n.Type()), a.Type, a, b)
		}
		return s.newValue2(s.ssaOp(n.Op(), n.Type()), a.Type, a, b)
	case ir.OAND, ir.OOR, ir.OXOR:
		n := n.(*ir.BinaryExpr)
		a := s.expr(n.X)
		b := s.expr(n.Y)
		return s.newValue2(s.ssaOp(n.Op(), n.Type()), a.Type, a, b)
	case ir.OANDNOT:
		n := n.(*ir.BinaryExpr)
		a := s.expr(n.X)
		b := s.expr(n.Y)
		b = s.newValue1(s.ssaOp(ir.OBITNOT, b.Type), b.Type, b)
		return s.newValue2(s.ssaOp(ir.OAND, n.Type()), a.Type, a, b)
	case ir.OLSH, ir.ORSH:
		n := n.(*ir.BinaryExpr)
		a := s.expr(n.X)
		b := s.expr(n.Y)
		bt := b.Type
		if bt.IsSigned() {
			cmp := s.newValue2(s.ssaOp(ir.OLE, bt), types.Types[types.TBOOL], s.zeroVal(bt), b)
			s.check(cmp, ir.Syms.Panicshift)
			bt = bt.ToUnsigned()
		}
		return s.newValue2(s.ssaShiftOp(n.Op(), n.Type(), bt), a.Type, a, b)
	case ir.OANDAND, ir.OOROR:
		// To implement OANDAND (and OOROR), we introduce a
		// new temporary variable to hold the result. The
		// variable is associated with the OANDAND node in the
		// s.vars table (normally variables are only
		// associated with ONAME nodes). We convert
		//     A && B
		// to
		//     var = A
		//     if var {
		//         var = B
		//     }
		// Using var in the subsequent block introduces the
		// necessary phi variable.
		n := n.(*ir.LogicalExpr)
		el := s.expr(n.X)
		s.vars[n] = el

		b := s.endBlock()
		b.Kind = ssa.BlockIf
		b.SetControl(el)
		// In theory, we should set b.Likely here based on context.
		// However, gc only gives us likeliness hints
		// in a single place, for plain OIF statements,
		// and passing around context is finnicky, so don't bother for now.

		bRight := s.f.NewBlock(ssa.BlockPlain)
		bResult := s.f.NewBlock(ssa.BlockPlain)
		if n.Op() == ir.OANDAND {
			b.AddEdgeTo(bRight)
			b.AddEdgeTo(bResult)
		} else if n.Op() == ir.OOROR {
			b.AddEdgeTo(bResult)
			b.AddEdgeTo(bRight)
		}

		s.startBlock(bRight)
		er := s.expr(n.Y)
		s.vars[n] = er

		b = s.endBlock()
		b.AddEdgeTo(bResult)

		s.startBlock(bResult)
		return s.variable(n, types.Types[types.TBOOL])
	case ir.OCOMPLEX:
		n := n.(*ir.BinaryExpr)
		r := s.expr(n.X)
		i := s.expr(n.Y)
		return s.newValue2(ssa.OpComplexMake, n.Type(), r, i)

	// unary ops
	case ir.ONEG:
		n := n.(*ir.UnaryExpr)
		a := s.expr(n.X)
		if n.Type().IsComplex() {
			tp := types.FloatForComplex(n.Type())
			negop := s.ssaOp(n.Op(), tp)
			return s.newValue2(ssa.OpComplexMake, n.Type(),
				s.newValue1(negop, tp, s.newValue1(ssa.OpComplexReal, tp, a)),
				s.newValue1(negop, tp, s.newValue1(ssa.OpComplexImag, tp, a)))
		}
		return s.newValue1(s.ssaOp(n.Op(), n.Type()), a.Type, a)
	case ir.ONOT, ir.OBITNOT:
		n := n.(*ir.UnaryExpr)
		a := s.expr(n.X)
		return s.newValue1(s.ssaOp(n.Op(), n.Type()), a.Type, a)
	case ir.OIMAG, ir.OREAL:
		n := n.(*ir.UnaryExpr)
		a := s.expr(n.X)
		return s.newValue1(s.ssaOp(n.Op(), n.X.Type()), n.Type(), a)
	case ir.OPLUS:
		n := n.(*ir.UnaryExpr)
		return s.expr(n.X)

	case ir.OADDR:
		n := n.(*ir.AddrExpr)
		return s.addr(n.X)

	case ir.ORESULT:
		n := n.(*ir.ResultExpr)
		if s.prevCall == nil || s.prevCall.Op != ssa.OpStaticLECall && s.prevCall.Op != ssa.OpInterLECall && s.prevCall.Op != ssa.OpClosureLECall {
			panic("Expected to see a previous call")
		}
		which := n.Index
		if which == -1 {
			panic(fmt.Errorf("ORESULT %v does not match call %s", n, s.prevCall))
		}
		return s.resultOfCall(s.prevCall, which, n.Type())

	case ir.ODEREF:
		n := n.(*ir.StarExpr)
		p := s.exprPtr(n.X, n.Bounded(), n.Pos())
		return s.load(n.Type(), p)

	case ir.ODOT:
		n := n.(*ir.SelectorExpr)
		if n.X.Op() == ir.OSTRUCTLIT {
			// All literals with nonzero fields have already been
			// rewritten during walk. Any that remain are just T{}
			// or equivalents. Use the zero value.
			if !ir.IsZero(n.X) {
				s.Fatalf("literal with nonzero value in SSA: %v", n.X)
			}
			return s.zeroVal(n.Type())
		}
		// If n is addressable and can't be represented in
		// SSA, then load just the selected field. This
		// prevents false memory dependencies in race/msan
		// instrumentation.
		if ir.IsAddressable(n) && !s.canSSA(n) {
			p := s.addr(n)
			return s.load(n.Type(), p)
		}
		v := s.expr(n.X)
		return s.newValue1I(ssa.OpStructSelect, n.Type(), int64(fieldIdx(n)), v)

	case ir.ODOTPTR:
		n := n.(*ir.SelectorExpr)
		p := s.exprPtr(n.X, n.Bounded(), n.Pos())
		p = s.newValue1I(ssa.OpOffPtr, types.NewPtr(n.Type()), n.Offset(), p)
		return s.load(n.Type(), p)

	case ir.OINDEX:
		n := n.(*ir.IndexExpr)
		switch {
		case n.X.Type().IsString():
			if n.Bounded() && ir.IsConst(n.X, constant.String) && ir.IsConst(n.Index, constant.Int) {
				// Replace "abc"[1] with 'b'.
				// Delayed until now because "abc"[1] is not an ideal constant.
				// See test/fixedbugs/issue11370.go.
				return s.newValue0I(ssa.OpConst8, types.Types[types.TUINT8], int64(int8(ir.StringVal(n.X)[ir.Int64Val(n.Index)])))
			}
			a := s.expr(n.X)
			i := s.expr(n.Index)
			len := s.newValue1(ssa.OpStringLen, types.Types[types.TINT], a)
			i = s.boundsCheck(i, len, ssa.BoundsIndex, n.Bounded())
			ptrtyp := s.f.Config.Types.BytePtr
			ptr := s.newValue1(ssa.OpStringPtr, ptrtyp, a)
			if ir.IsConst(n.Index, constant.Int) {
				ptr = s.newValue1I(ssa.OpOffPtr, ptrtyp, ir.Int64Val(n.Index), ptr)
			} else {
				ptr = s.newValue2(ssa.OpAddPtr, ptrtyp, ptr, i)
			}
			return s.load(types.Types[types.TUINT8], ptr)
		case n.X.Type().IsSlice():
			p := s.addr(n)
			return s.load(n.X.Type().Elem(), p)
		case n.X.Type().IsArray():
			if TypeOK(n.X.Type()) {
				// SSA can handle arrays of length at most 1.
				bound := n.X.Type().NumElem()
				a := s.expr(n.X)
				i := s.expr(n.Index)
				if bound == 0 {
					// Bounds check will never succeed.  Might as well
					// use constants for the bounds check.
					z := s.constInt(types.Types[types.TINT], 0)
					s.boundsCheck(z, z, ssa.BoundsIndex, false)
					// The return value won't be live, return junk.
					return s.newValue0(ssa.OpUnknown, n.Type())
				}
				len := s.constInt(types.Types[types.TINT], bound)
				s.boundsCheck(i, len, ssa.BoundsIndex, n.Bounded()) // checks i == 0
				return s.newValue1I(ssa.OpArraySelect, n.Type(), 0, a)
			}
			p := s.addr(n)
			return s.load(n.X.Type().Elem(), p)
		default:
			s.Fatalf("bad type for index %v", n.X.Type())
			return nil
		}

	case ir.OLEN, ir.OCAP:
		n := n.(*ir.UnaryExpr)
		switch {
		case n.X.Type().IsSlice():
			op := ssa.OpSliceLen
			if n.Op() == ir.OCAP {
				op = ssa.OpSliceCap
			}
			return s.newValue1(op, types.Types[types.TINT], s.expr(n.X))
		case n.X.Type().IsString(): // string; not reachable for OCAP
			return s.newValue1(ssa.OpStringLen, types.Types[types.TINT], s.expr(n.X))
		case n.X.Type().IsMap(), n.X.Type().IsChan():
			return s.referenceTypeBuiltin(n, s.expr(n.X))
		default: // array
			return s.constInt(types.Types[types.TINT], n.X.Type().NumElem())
		}

	case ir.OSPTR:
		n := n.(*ir.UnaryExpr)
		a := s.expr(n.X)
		if n.X.Type().IsSlice() {
			return s.newValue1(ssa.OpSlicePtr, n.Type(), a)
		} else {
			return s.newValue1(ssa.OpStringPtr, n.Type(), a)
		}

	case ir.OITAB:
		n := n.(*ir.UnaryExpr)
		a := s.expr(n.X)
		return s.newValue1(ssa.OpITab, n.Type(), a)

	case ir.OIDATA:
		n := n.(*ir.UnaryExpr)
		a := s.expr(n.X)
		return s.newValue1(ssa.OpIData, n.Type(), a)

	case ir.OEFACE:
		n := n.(*ir.BinaryExpr)
		tab := s.expr(n.X)
		data := s.expr(n.Y)
		return s.newValue2(ssa.OpIMake, n.Type(), tab, data)

	case ir.OSLICEHEADER:
		n := n.(*ir.SliceHeaderExpr)
		p := s.expr(n.Ptr)
		l := s.expr(n.Len)
		c := s.expr(n.Cap)
		return s.newValue3(ssa.OpSliceMake, n.Type(), p, l, c)

	case ir.OSLICE, ir.OSLICEARR, ir.OSLICE3, ir.OSLICE3ARR:
		n := n.(*ir.SliceExpr)
		v := s.expr(n.X)
		var i, j, k *ssa.Value
		if n.Low != nil {
			i = s.expr(n.Low)
		}
		if n.High != nil {
			j = s.expr(n.High)
		}
		if n.Max != nil {
			k = s.expr(n.Max)
		}
		p, l, c := s.slice(v, i, j, k, n.Bounded())
		return s.newValue3(ssa.OpSliceMake, n.Type(), p, l, c)

	case ir.OSLICESTR:
		n := n.(*ir.SliceExpr)
		v := s.expr(n.X)
		var i, j *ssa.Value
		if n.Low != nil {
			i = s.expr(n.Low)
		}
		if n.High != nil {
			j = s.expr(n.High)
		}
		p, l, _ := s.slice(v, i, j, nil, n.Bounded())
		return s.newValue2(ssa.OpStringMake, n.Type(), p, l)

	case ir.OSLICE2ARRPTR:
		// if arrlen > slice.len {
		//   panic(...)
		// }
		// slice.ptr
		n := n.(*ir.ConvExpr)
		v := s.expr(n.X)
		arrlen := s.constInt(types.Types[types.TINT], n.Type().Elem().NumElem())
		cap := s.newValue1(ssa.OpSliceLen, types.Types[types.TINT], v)
		s.boundsCheck(arrlen, cap, ssa.BoundsConvert, false)
		return s.newValue1(ssa.OpSlicePtrUnchecked, n.Type(), v)

	case ir.OCALLFUNC:
		n := n.(*ir.CallExpr)
		if ir.IsIntrinsicCall(n) {
			return s.intrinsicCall(n)
		}
		fallthrough

	case ir.OCALLINTER, ir.OCALLMETH:
		n := n.(*ir.CallExpr)
		return s.callResult(n, callNormal)

	case ir.OGETG:
		n := n.(*ir.CallExpr)
		return s.newValue1(ssa.OpGetG, n.Type(), s.mem())

	case ir.OAPPEND:
		return s.append(n.(*ir.CallExpr), false)

	case ir.OSTRUCTLIT, ir.OARRAYLIT:
		// All literals with nonzero fields have already been
		// rewritten during walk. Any that remain are just T{}
		// or equivalents. Use the zero value.
		n := n.(*ir.CompLitExpr)
		if !ir.IsZero(n) {
			s.Fatalf("literal with nonzero value in SSA: %v", n)
		}
		return s.zeroVal(n.Type())

	case ir.ONEW:
		n := n.(*ir.UnaryExpr)
		return s.newObject(n.Type().Elem())

	case ir.OUNSAFEADD:
		n := n.(*ir.BinaryExpr)
		ptr := s.expr(n.X)
		len := s.expr(n.Y)
		return s.newValue2(ssa.OpAddPtr, n.Type(), ptr, len)

	default:
		s.Fatalf("unhandled expr %v", n.Op())
		return nil
	}
}

func (s *state) resultOfCall(c *ssa.Value, which int64, t *types.Type) *ssa.Value {
	aux := c.Aux.(*ssa.AuxCall)
	pa := aux.ParamAssignmentForResult(which)
	// TODO(register args) determine if in-memory TypeOK is better loaded early from SelectNAddr or later when SelectN is expanded.
	// SelectN is better for pattern-matching and possible call-aware analysis we might want to do in the future.
	if len(pa.Registers) == 0 && !TypeOK(t) {
		addr := s.newValue1I(ssa.OpSelectNAddr, types.NewPtr(t), which, c)
		return s.rawLoad(t, addr)
	}
	return s.newValue1I(ssa.OpSelectN, t, which, c)
}

func (s *state) resultAddrOfCall(c *ssa.Value, which int64, t *types.Type) *ssa.Value {
	aux := c.Aux.(*ssa.AuxCall)
	pa := aux.ParamAssignmentForResult(which)
	if len(pa.Registers) == 0 {
		return s.newValue1I(ssa.OpSelectNAddr, types.NewPtr(t), which, c)
	}
	_, addr := s.temp(c.Pos, t)
	rval := s.newValue1I(ssa.OpSelectN, t, which, c)
	s.vars[memVar] = s.newValue3Apos(ssa.OpStore, types.TypeMem, t, addr, rval, s.mem(), false)
	return addr
}

// append converts an OAPPEND node to SSA.
// If inplace is false, it converts the OAPPEND expression n to an ssa.Value,
// adds it to s, and returns the Value.
// If inplace is true, it writes the result of the OAPPEND expression n
// back to the slice being appended to, and returns nil.
// inplace MUST be set to false if the slice can be SSA'd.
func (s *state) append(n *ir.CallExpr, inplace bool) *ssa.Value {
	// If inplace is false, process as expression "append(s, e1, e2, e3)":
	//
	// ptr, len, cap := s
	// newlen := len + 3
	// if newlen > cap {
	//     ptr, len, cap = growslice(s, newlen)
	//     newlen = len + 3 // recalculate to avoid a spill
	// }
	// // with write barriers, if needed:
	// *(ptr+len) = e1
	// *(ptr+len+1) = e2
	// *(ptr+len+2) = e3
	// return makeslice(ptr, newlen, cap)
	//
	//
	// If inplace is true, process as statement "s = append(s, e1, e2, e3)":
	//
	// a := &s
	// ptr, len, cap := s
	// newlen := len + 3
	// if uint(newlen) > uint(cap) {
	//    newptr, len, newcap = growslice(ptr, len, cap, newlen)
	//    vardef(a)       // if necessary, advise liveness we are writing a new a
	//    *a.cap = newcap // write before ptr to avoid a spill
	//    *a.ptr = newptr // with write barrier
	// }
	// newlen = len + 3 // recalculate to avoid a spill
	// *a.len = newlen
	// // with write barriers, if needed:
	// *(ptr+len) = e1
	// *(ptr+len+1) = e2
	// *(ptr+len+2) = e3

	et := n.Type().Elem()
	pt := types.NewPtr(et)

	// Evaluate slice
	sn := n.Args[0] // the slice node is the first in the list

	var slice, addr *ssa.Value
	if inplace {
		addr = s.addr(sn)
		slice = s.load(n.Type(), addr)
	} else {
		slice = s.expr(sn)
	}

	// Allocate new blocks
	grow := s.f.NewBlock(ssa.BlockPlain)
	assign := s.f.NewBlock(ssa.BlockPlain)

	// Decide if we need to grow
	nargs := int64(len(n.Args) - 1)
	p := s.newValue1(ssa.OpSlicePtr, pt, slice)
	l := s.newValue1(ssa.OpSliceLen, types.Types[types.TINT], slice)
	c := s.newValue1(ssa.OpSliceCap, types.Types[types.TINT], slice)
	nl := s.newValue2(s.ssaOp(ir.OADD, types.Types[types.TINT]), types.Types[types.TINT], l, s.constInt(types.Types[types.TINT], nargs))

	cmp := s.newValue2(s.ssaOp(ir.OLT, types.Types[types.TUINT]), types.Types[types.TBOOL], c, nl)
	s.vars[ptrVar] = p

	if !inplace {
		s.vars[newlenVar] = nl
		s.vars[capVar] = c
	} else {
		s.vars[lenVar] = l
	}

	b := s.endBlock()
	b.Kind = ssa.BlockIf
	b.Likely = ssa.BranchUnlikely
	b.SetControl(cmp)
	b.AddEdgeTo(grow)
	b.AddEdgeTo(assign)

	// Call growslice
	s.startBlock(grow)
	taddr := s.expr(n.X)
	r := s.rtcall(ir.Syms.Growslice, true, []*types.Type{pt, types.Types[types.TINT], types.Types[types.TINT]}, taddr, p, l, c, nl)

	if inplace {
		if sn.Op() == ir.ONAME {
			sn := sn.(*ir.Name)
			if sn.Class != ir.PEXTERN {
				// Tell liveness we're about to build a new slice
				s.vars[memVar] = s.newValue1A(ssa.OpVarDef, types.TypeMem, sn, s.mem())
			}
		}
		capaddr := s.newValue1I(ssa.OpOffPtr, s.f.Config.Types.IntPtr, types.SliceCapOffset, addr)
		s.store(types.Types[types.TINT], capaddr, r[2])
		s.store(pt, addr, r[0])
		// load the value we just stored to avoid having to spill it
		s.vars[ptrVar] = s.load(pt, addr)
		s.vars[lenVar] = r[1] // avoid a spill in the fast path
	} else {
		s.vars[ptrVar] = r[0]
		s.vars[newlenVar] = s.newValue2(s.ssaOp(ir.OADD, types.Types[types.TINT]), types.Types[types.TINT], r[1], s.constInt(types.Types[types.TINT], nargs))
		s.vars[capVar] = r[2]
	}

	b = s.endBlock()
	b.AddEdgeTo(assign)

	// assign new elements to slots
	s.startBlock(assign)

	if inplace {
		l = s.variable(lenVar, types.Types[types.TINT]) // generates phi for len
		nl = s.newValue2(s.ssaOp(ir.OADD, types.Types[types.TINT]), types.Types[types.TINT], l, s.constInt(types.Types[types.TINT], nargs))
		lenaddr := s.newValue1I(ssa.OpOffPtr, s.f.Config.Types.IntPtr, types.SliceLenOffset, addr)
		s.store(types.Types[types.TINT], lenaddr, nl)
	}

	// Evaluate args
	type argRec struct {
		// if store is true, we're appending the value v.  If false, we're appending the
		// value at *v.
		v     *ssa.Value
		store bool
	}
	args := make([]argRec, 0, nargs)
	for _, n := range n.Args[1:] {
		if TypeOK(n.Type()) {
			args = append(args, argRec{v: s.expr(n), store: true})
		} else {
			v := s.addr(n)
			args = append(args, argRec{v: v})
		}
	}

	p = s.variable(ptrVar, pt) // generates phi for ptr
	if !inplace {
		nl = s.variable(newlenVar, types.Types[types.TINT]) // generates phi for nl
		c = s.variable(capVar, types.Types[types.TINT])     // generates phi for cap
	}
	p2 := s.newValue2(ssa.OpPtrIndex, pt, p, l)
	for i, arg := range args {
		addr := s.newValue2(ssa.OpPtrIndex, pt, p2, s.constInt(types.Types[types.TINT], int64(i)))
		if arg.store {
			s.storeType(et, addr, arg.v, 0, true)
		} else {
			s.move(et, addr, arg.v)
		}
	}

	delete(s.vars, ptrVar)
	if inplace {
		delete(s.vars, lenVar)
		return nil
	}
	delete(s.vars, newlenVar)
	delete(s.vars, capVar)
	// make result
	return s.newValue3(ssa.OpSliceMake, n.Type(), p, nl, c)
}

// condBranch evaluates the boolean expression cond and branches to yes
// if cond is true and no if cond is false.
// This function is intended to handle && and || better than just calling
// s.expr(cond) and branching on the result.
func (s *state) condBranch(cond ir.Node, yes, no *ssa.Block, likely int8) {
	switch cond.Op() {
	case ir.OANDAND:
		cond := cond.(*ir.LogicalExpr)
		mid := s.f.NewBlock(ssa.BlockPlain)
		s.stmtList(cond.Init())
		s.condBranch(cond.X, mid, no, max8(likely, 0))
		s.startBlock(mid)
		s.condBranch(cond.Y, yes, no, likely)
		return
		// Note: if likely==1, then both recursive calls pass 1.
		// If likely==-1, then we don't have enough information to decide
		// whether the first branch is likely or not. So we pass 0 for
		// the likeliness of the first branch.
		// TODO: have the frontend give us branch prediction hints for
		// OANDAND and OOROR nodes (if it ever has such info).
	case ir.OOROR:
		cond := cond.(*ir.LogicalExpr)
		mid := s.f.NewBlock(ssa.BlockPlain)
		s.stmtList(cond.Init())
		s.condBranch(cond.X, yes, mid, min8(likely, 0))
		s.startBlock(mid)
		s.condBranch(cond.Y, yes, no, likely)
		return
		// Note: if likely==-1, then both recursive calls pass -1.
		// If likely==1, then we don't have enough info to decide
		// the likelihood of the first branch.
	case ir.ONOT:
		cond := cond.(*ir.UnaryExpr)
		s.stmtList(cond.Init())
		s.condBranch(cond.X, no, yes, -likely)
		return
	case ir.OCONVNOP:
		cond := cond.(*ir.ConvExpr)
		s.stmtList(cond.Init())
		s.condBranch(cond.X, yes, no, likely)
		return
	}
	c := s.expr(cond)
	b := s.endBlock()
	b.Kind = ssa.BlockIf
	b.SetControl(c)
	b.Likely = ssa.BranchPrediction(likely) // gc and ssa both use -1/0/+1 for likeliness
	b.AddEdgeTo(yes)
	b.AddEdgeTo(no)
}

type skipMask uint8

const (
	skipPtr skipMask = 1 << iota
	skipLen
	skipCap
)

// assign does left = right.
// Right has already been evaluated to ssa, left has not.
// If deref is true, then we do left = *right instead (and right has already been nil-checked).
// If deref is true and right == nil, just do left = 0.
// skip indicates assignments (at the top level) that can be avoided.
func (s *state) assign(left ir.Node, right *ssa.Value, deref bool, skip skipMask) {
	if left.Op() == ir.ONAME && ir.IsBlank(left) {
		return
	}
	t := left.Type()
	types.CalcSize(t)
	if s.canSSA(left) {
		if deref {
			s.Fatalf("can SSA LHS %v but not RHS %s", left, right)
		}
		if left.Op() == ir.ODOT {
			// We're assigning to a field of an ssa-able value.
			// We need to build a new structure with the new value for the
			// field we're assigning and the old values for the other fields.
			// For instance:
			//   type T struct {a, b, c int}
			//   var T x
			//   x.b = 5
			// For the x.b = 5 assignment we want to generate x = T{x.a, 5, x.c}

			// Grab information about the structure type.
			left := left.(*ir.SelectorExpr)
			t := left.X.Type()
			nf := t.NumFields()
			idx := fieldIdx(left)

			// Grab old value of structure.
			old := s.expr(left.X)

			// Make new structure.
			new := s.newValue0(ssa.StructMakeOp(t.NumFields()), t)

			// Add fields as args.
			for i := 0; i < nf; i++ {
				if i == idx {
					new.AddArg(right)
				} else {
					new.AddArg(s.newValue1I(ssa.OpStructSelect, t.FieldType(i), int64(i), old))
				}
			}

			// Recursively assign the new value we've made to the base of the dot op.
			s.assign(left.X, new, false, 0)
			// TODO: do we need to update named values here?
			return
		}
		if left.Op() == ir.OINDEX && left.(*ir.IndexExpr).X.Type().IsArray() {
			left := left.(*ir.IndexExpr)
			s.pushLine(left.Pos())
			defer s.popLine()
			// We're assigning to an element of an ssa-able array.
			// a[i] = v
			t := left.X.Type()
			n := t.NumElem()

			i := s.expr(left.Index) // index
			if n == 0 {
				// The bounds check must fail.  Might as well
				// ignore the actual index and just use zeros.
				z := s.constInt(types.Types[types.TINT], 0)
				s.boundsCheck(z, z, ssa.BoundsIndex, false)
				return
			}
			if n != 1 {
				s.Fatalf("assigning to non-1-length array")
			}
			// Rewrite to a = [1]{v}
			len := s.constInt(types.Types[types.TINT], 1)
			s.boundsCheck(i, len, ssa.BoundsIndex, false) // checks i == 0
			v := s.newValue1(ssa.OpArrayMake1, t, right)
			s.assign(left.X, v, false, 0)
			return
		}
		left := left.(*ir.Name)
		// Update variable assignment.
		s.vars[left] = right
		s.addNamedValue(left, right)
		return
	}

	// If this assignment clobbers an entire local variable, then emit
	// OpVarDef so liveness analysis knows the variable is redefined.
	if base, ok := clobberBase(left).(*ir.Name); ok && base.OnStack() && skip == 0 {
		s.vars[memVar] = s.newValue1Apos(ssa.OpVarDef, types.TypeMem, base, s.mem(), !ir.IsAutoTmp(base))
	}

	// Left is not ssa-able. Compute its address.
	addr := s.addr(left)
	if ir.IsReflectHeaderDataField(left) {
		// Package unsafe's documentation says storing pointers into
		// reflect.SliceHeader and reflect.StringHeader's Data fields
		// is valid, even though they have type uintptr (#19168).
		// Mark it pointer type to signal the writebarrier pass to
		// insert a write barrier.
		t = types.Types[types.TUNSAFEPTR]
	}
	if deref {
		// Treat as a mem->mem move.
		if right == nil {
			s.zero(t, addr)
		} else {
			s.move(t, addr, right)
		}
		return
	}
	// Treat as a store.
	s.storeType(t, addr, right, skip, !ir.IsAutoTmp(left))
}

// zeroVal returns the zero value for type t.
func (s *state) zeroVal(t *types.Type) *ssa.Value {
	switch {
	case t.IsInteger():
		switch t.Size() {
		case 1:
			return s.constInt8(t, 0)
		case 2:
			return s.constInt16(t, 0)
		case 4:
			return s.constInt32(t, 0)
		case 8:
			return s.constInt64(t, 0)
		default:
			s.Fatalf("bad sized integer type %v", t)
		}
	case t.IsFloat():
		switch t.Size() {
		case 4:
			return s.constFloat32(t, 0)
		case 8:
			return s.constFloat64(t, 0)
		default:
			s.Fatalf("bad sized float type %v", t)
		}
	case t.IsComplex():
		switch t.Size() {
		case 8:
			z := s.constFloat32(types.Types[types.TFLOAT32], 0)
			return s.entryNewValue2(ssa.OpComplexMake, t, z, z)
		case 16:
			z := s.constFloat64(types.Types[types.TFLOAT64], 0)
			return s.entryNewValue2(ssa.OpComplexMake, t, z, z)
		default:
			s.Fatalf("bad sized complex type %v", t)
		}

	case t.IsString():
		return s.constEmptyString(t)
	case t.IsPtrShaped():
		return s.constNil(t)
	case t.IsBoolean():
		return s.constBool(false)
	case t.IsInterface():
		return s.constInterface(t)
	case t.IsSlice():
		return s.constSlice(t)
	case t.IsStruct():
		n := t.NumFields()
		v := s.entryNewValue0(ssa.StructMakeOp(t.NumFields()), t)
		for i := 0; i < n; i++ {
			v.AddArg(s.zeroVal(t.FieldType(i)))
		}
		return v
	case t.IsArray():
		switch t.NumElem() {
		case 0:
			return s.entryNewValue0(ssa.OpArrayMake0, t)
		case 1:
			return s.entryNewValue1(ssa.OpArrayMake1, t, s.zeroVal(t.Elem()))
		}
	}
	s.Fatalf("zero for type %v not implemented", t)
	return nil
}

type callKind int8

const (
	callNormal callKind = iota
	callDefer
	callDeferStack
	callGo
)

type sfRtCallDef struct {
	rtfn  *obj.LSym
	rtype types.Kind
}

var softFloatOps map[ssa.Op]sfRtCallDef

func softfloatInit() {
	// Some of these operations get transformed by sfcall.
	softFloatOps = map[ssa.Op]sfRtCallDef{
		ssa.OpAdd32F: sfRtCallDef{typecheck.LookupRuntimeFunc("fadd32"), types.TFLOAT32},
		ssa.OpAdd64F: sfRtCallDef{typecheck.LookupRuntimeFunc("fadd64"), types.TFLOAT64},
		ssa.OpSub32F: sfRtCallDef{typecheck.LookupRuntimeFunc("fadd32"), types.TFLOAT32},
		ssa.OpSub64F: sfRtCallDef{typecheck.LookupRuntimeFunc("fadd64"), types.TFLOAT64},
		ssa.OpMul32F: sfRtCallDef{typecheck.LookupRuntimeFunc("fmul32"), types.TFLOAT32},
		ssa.OpMul64F: sfRtCallDef{typecheck.LookupRuntimeFunc("fmul64"), types.TFLOAT64},
		ssa.OpDiv32F: sfRtCallDef{typecheck.LookupRuntimeFunc("fdiv32"), types.TFLOAT32},
		ssa.OpDiv64F: sfRtCallDef{typecheck.LookupRuntimeFunc("fdiv64"), types.TFLOAT64},

		ssa.OpEq64F:   sfRtCallDef{typecheck.LookupRuntimeFunc("feq64"), types.TBOOL},
		ssa.OpEq32F:   sfRtCallDef{typecheck.LookupRuntimeFunc("feq32"), types.TBOOL},
		ssa.OpNeq64F:  sfRtCallDef{typecheck.LookupRuntimeFunc("feq64"), types.TBOOL},
		ssa.OpNeq32F:  sfRtCallDef{typecheck.LookupRuntimeFunc("feq32"), types.TBOOL},
		ssa.OpLess64F: sfRtCallDef{typecheck.LookupRuntimeFunc("fgt64"), types.TBOOL},
		ssa.OpLess32F: sfRtCallDef{typecheck.LookupRuntimeFunc("fgt32"), types.TBOOL},
		ssa.OpLeq64F:  sfRtCallDef{typecheck.LookupRuntimeFunc("fge64"), types.TBOOL},
		ssa.OpLeq32F:  sfRtCallDef{typecheck.LookupRuntimeFunc("fge32"), types.TBOOL},

		ssa.OpCvt32to32F:  sfRtCallDef{typecheck.LookupRuntimeFunc("fint32to32"), types.TFLOAT32},
		ssa.OpCvt32Fto32:  sfRtCallDef{typecheck.LookupRuntimeFunc("f32toint32"), types.TINT32},
		ssa.OpCvt64to32F:  sfRtCallDef{typecheck.LookupRuntimeFunc("fint64to32"), types.TFLOAT32},
		ssa.OpCvt32Fto64:  sfRtCallDef{typecheck.LookupRuntimeFunc("f32toint64"), types.TINT64},
		ssa.OpCvt64Uto32F: sfRtCallDef{typecheck.LookupRuntimeFunc("fuint64to32"), types.TFLOAT32},
		ssa.OpCvt32Fto64U: sfRtCallDef{typecheck.LookupRuntimeFunc("f32touint64"), types.TUINT64},
		ssa.OpCvt32to64F:  sfRtCallDef{typecheck.LookupRuntimeFunc("fint32to64"), types.TFLOAT64},
		ssa.OpCvt64Fto32:  sfRtCallDef{typecheck.LookupRuntimeFunc("f64toint32"), types.TINT32},
		ssa.OpCvt64to64F:  sfRtCallDef{typecheck.LookupRuntimeFunc("fint64to64"), types.TFLOAT64},
		ssa.OpCvt64Fto64:  sfRtCallDef{typecheck.LookupRuntimeFunc("f64toint64"), types.TINT64},
		ssa.OpCvt64Uto64F: sfRtCallDef{typecheck.LookupRuntimeFunc("fuint64to64"), types.TFLOAT64},
		ssa.OpCvt64Fto64U: sfRtCallDef{typecheck.LookupRuntimeFunc("f64touint64"), types.TUINT64},
		ssa.OpCvt32Fto64F: sfRtCallDef{typecheck.LookupRuntimeFunc("f32to64"), types.TFLOAT64},
		ssa.OpCvt64Fto32F: sfRtCallDef{typecheck.LookupRuntimeFunc("f64to32"), types.TFLOAT32},
	}
}

// TODO: do not emit sfcall if operation can be optimized to constant in later
// opt phase
func (s *state) sfcall(op ssa.Op, args ...*ssa.Value) (*ssa.Value, bool) {
	if callDef, ok := softFloatOps[op]; ok {
		switch op {
		case ssa.OpLess32F,
			ssa.OpLess64F,
			ssa.OpLeq32F,
			ssa.OpLeq64F:
			args[0], args[1] = args[1], args[0]
		case ssa.OpSub32F,
			ssa.OpSub64F:
			args[1] = s.newValue1(s.ssaOp(ir.ONEG, types.Types[callDef.rtype]), args[1].Type, args[1])
		}

		result := s.rtcall(callDef.rtfn, true, []*types.Type{types.Types[callDef.rtype]}, args...)[0]
		if op == ssa.OpNeq32F || op == ssa.OpNeq64F {
			result = s.newValue1(ssa.OpNot, result.Type, result)
		}
		return result, true
	}
	return nil, false
}

var intrinsics map[intrinsicKey]intrinsicBuilder

// An intrinsicBuilder converts a call node n into an ssa value that
// implements that call as an intrinsic. args is a list of arguments to the func.
type intrinsicBuilder func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value

type intrinsicKey struct {
	arch *sys.Arch
	pkg  string
	fn   string
}

func InitTables() {
	intrinsics = map[intrinsicKey]intrinsicBuilder{}

	var all []*sys.Arch
	var p4 []*sys.Arch
	var p8 []*sys.Arch
	var lwatomics []*sys.Arch
	for _, a := range &sys.Archs {
		all = append(all, a)
		if a.PtrSize == 4 {
			p4 = append(p4, a)
		} else {
			p8 = append(p8, a)
		}
		if a.Family != sys.PPC64 {
			lwatomics = append(lwatomics, a)
		}
	}

	// add adds the intrinsic b for pkg.fn for the given list of architectures.
	add := func(pkg, fn string, b intrinsicBuilder, archs ...*sys.Arch) {
		for _, a := range archs {
			intrinsics[intrinsicKey{a, pkg, fn}] = b
		}
	}
	// addF does the same as add but operates on architecture families.
	addF := func(pkg, fn string, b intrinsicBuilder, archFamilies ...sys.ArchFamily) {
		m := 0
		for _, f := range archFamilies {
			if f >= 32 {
				panic("too many architecture families")
			}
			m |= 1 << uint(f)
		}
		for _, a := range all {
			if m>>uint(a.Family)&1 != 0 {
				intrinsics[intrinsicKey{a, pkg, fn}] = b
			}
		}
	}
	// alias defines pkg.fn = pkg2.fn2 for all architectures in archs for which pkg2.fn2 exists.
	alias := func(pkg, fn, pkg2, fn2 string, archs ...*sys.Arch) {
		aliased := false
		for _, a := range archs {
			if b, ok := intrinsics[intrinsicKey{a, pkg2, fn2}]; ok {
				intrinsics[intrinsicKey{a, pkg, fn}] = b
				aliased = true
			}
		}
		if !aliased {
			panic(fmt.Sprintf("attempted to alias undefined intrinsic: %s.%s", pkg, fn))
		}
	}

	/******** runtime ********/
	if !base.Flag.Cfg.Instrumenting {
		add("runtime", "slicebytetostringtmp",
			func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
				// Compiler frontend optimizations emit OBYTES2STRTMP nodes
				// for the backend instead of slicebytetostringtmp calls
				// when not instrumenting.
				return s.newValue2(ssa.OpStringMake, n.Type(), args[0], args[1])
			},
			all...)
	}
	addF("runtime/internal/math", "MulUintptr",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			if s.config.PtrSize == 4 {
				return s.newValue2(ssa.OpMul32uover, types.NewTuple(types.Types[types.TUINT], types.Types[types.TUINT]), args[0], args[1])
			}
			return s.newValue2(ssa.OpMul64uover, types.NewTuple(types.Types[types.TUINT], types.Types[types.TUINT]), args[0], args[1])
		},
		sys.AMD64, sys.I386, sys.MIPS64)
	add("runtime", "KeepAlive",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			data := s.newValue1(ssa.OpIData, s.f.Config.Types.BytePtr, args[0])
			s.vars[memVar] = s.newValue2(ssa.OpKeepAlive, types.TypeMem, data, s.mem())
			return nil
		},
		all...)
	add("runtime", "getclosureptr",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue0(ssa.OpGetClosurePtr, s.f.Config.Types.Uintptr)
		},
		all...)

	add("runtime", "getcallerpc",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue0(ssa.OpGetCallerPC, s.f.Config.Types.Uintptr)
		},
		all...)

	add("runtime", "getcallersp",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue0(ssa.OpGetCallerSP, s.f.Config.Types.Uintptr)
		},
		all...)

	/******** runtime/internal/sys ********/
	addF("runtime/internal/sys", "Ctz32",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue1(ssa.OpCtz32, types.Types[types.TINT], args[0])
		},
		sys.AMD64, sys.ARM64, sys.ARM, sys.S390X, sys.MIPS, sys.PPC64)
	addF("runtime/internal/sys", "Ctz64",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue1(ssa.OpCtz64, types.Types[types.TINT], args[0])
		},
		sys.AMD64, sys.ARM64, sys.ARM, sys.S390X, sys.MIPS, sys.PPC64)
	addF("runtime/internal/sys", "Bswap32",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue1(ssa.OpBswap32, types.Types[types.TUINT32], args[0])
		},
		sys.AMD64, sys.ARM64, sys.ARM, sys.S390X)
	addF("runtime/internal/sys", "Bswap64",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue1(ssa.OpBswap64, types.Types[types.TUINT64], args[0])
		},
		sys.AMD64, sys.ARM64, sys.ARM, sys.S390X)

	/******** runtime/internal/atomic ********/
	addF("runtime/internal/atomic", "Load",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			v := s.newValue2(ssa.OpAtomicLoad32, types.NewTuple(types.Types[types.TUINT32], types.TypeMem), args[0], s.mem())
			s.vars[memVar] = s.newValue1(ssa.OpSelect1, types.TypeMem, v)
			return s.newValue1(ssa.OpSelect0, types.Types[types.TUINT32], v)
		},
		sys.AMD64, sys.ARM64, sys.MIPS, sys.MIPS64, sys.PPC64, sys.RISCV64, sys.S390X)
	addF("runtime/internal/atomic", "Load8",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			v := s.newValue2(ssa.OpAtomicLoad8, types.NewTuple(types.Types[types.TUINT8], types.TypeMem), args[0], s.mem())
			s.vars[memVar] = s.newValue1(ssa.OpSelect1, types.TypeMem, v)
			return s.newValue1(ssa.OpSelect0, types.Types[types.TUINT8], v)
		},
		sys.AMD64, sys.ARM64, sys.MIPS, sys.MIPS64, sys.PPC64, sys.RISCV64, sys.S390X)
	addF("runtime/internal/atomic", "Load64",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			v := s.newValue2(ssa.OpAtomicLoad64, types.NewTuple(types.Types[types.TUINT64], types.TypeMem), args[0], s.mem())
			s.vars[memVar] = s.newValue1(ssa.OpSelect1, types.TypeMem, v)
			return s.newValue1(ssa.OpSelect0, types.Types[types.TUINT64], v)
		},
		sys.AMD64, sys.ARM64, sys.MIPS64, sys.PPC64, sys.RISCV64, sys.S390X)
	addF("runtime/internal/atomic", "LoadAcq",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			v := s.newValue2(ssa.OpAtomicLoadAcq32, types.NewTuple(types.Types[types.TUINT32], types.TypeMem), args[0], s.mem())
			s.vars[memVar] = s.newValue1(ssa.OpSelect1, types.TypeMem, v)
			return s.newValue1(ssa.OpSelect0, types.Types[types.TUINT32], v)
		},
		sys.PPC64, sys.S390X)
	addF("runtime/internal/atomic", "LoadAcq64",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			v := s.newValue2(ssa.OpAtomicLoadAcq64, types.NewTuple(types.Types[types.TUINT64], types.TypeMem), args[0], s.mem())
			s.vars[memVar] = s.newValue1(ssa.OpSelect1, types.TypeMem, v)
			return s.newValue1(ssa.OpSelect0, types.Types[types.TUINT64], v)
		},
		sys.PPC64)
	addF("runtime/internal/atomic", "Loadp",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			v := s.newValue2(ssa.OpAtomicLoadPtr, types.NewTuple(s.f.Config.Types.BytePtr, types.TypeMem), args[0], s.mem())
			s.vars[memVar] = s.newValue1(ssa.OpSelect1, types.TypeMem, v)
			return s.newValue1(ssa.OpSelect0, s.f.Config.Types.BytePtr, v)
		},
		sys.AMD64, sys.ARM64, sys.MIPS, sys.MIPS64, sys.PPC64, sys.RISCV64, sys.S390X)

	addF("runtime/internal/atomic", "Store",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			s.vars[memVar] = s.newValue3(ssa.OpAtomicStore32, types.TypeMem, args[0], args[1], s.mem())
			return nil
		},
		sys.AMD64, sys.ARM64, sys.MIPS, sys.MIPS64, sys.PPC64, sys.RISCV64, sys.S390X)
	addF("runtime/internal/atomic", "Store8",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			s.vars[memVar] = s.newValue3(ssa.OpAtomicStore8, types.TypeMem, args[0], args[1], s.mem())
			return nil
		},
		sys.AMD64, sys.ARM64, sys.MIPS, sys.MIPS64, sys.PPC64, sys.RISCV64, sys.S390X)
	addF("runtime/internal/atomic", "Store64",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			s.vars[memVar] = s.newValue3(ssa.OpAtomicStore64, types.TypeMem, args[0], args[1], s.mem())
			return nil
		},
		sys.AMD64, sys.ARM64, sys.MIPS64, sys.PPC64, sys.RISCV64, sys.S390X)
	addF("runtime/internal/atomic", "StorepNoWB",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			s.vars[memVar] = s.newValue3(ssa.OpAtomicStorePtrNoWB, types.TypeMem, args[0], args[1], s.mem())
			return nil
		},
		sys.AMD64, sys.ARM64, sys.MIPS, sys.MIPS64, sys.RISCV64, sys.S390X)
	addF("runtime/internal/atomic", "StoreRel",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			s.vars[memVar] = s.newValue3(ssa.OpAtomicStoreRel32, types.TypeMem, args[0], args[1], s.mem())
			return nil
		},
		sys.PPC64, sys.S390X)
	addF("runtime/internal/atomic", "StoreRel64",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			s.vars[memVar] = s.newValue3(ssa.OpAtomicStoreRel64, types.TypeMem, args[0], args[1], s.mem())
			return nil
		},
		sys.PPC64)

	addF("runtime/internal/atomic", "Xchg",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			v := s.newValue3(ssa.OpAtomicExchange32, types.NewTuple(types.Types[types.TUINT32], types.TypeMem), args[0], args[1], s.mem())
			s.vars[memVar] = s.newValue1(ssa.OpSelect1, types.TypeMem, v)
			return s.newValue1(ssa.OpSelect0, types.Types[types.TUINT32], v)
		},
		sys.AMD64, sys.MIPS, sys.MIPS64, sys.PPC64, sys.RISCV64, sys.S390X)
	addF("runtime/internal/atomic", "Xchg64",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			v := s.newValue3(ssa.OpAtomicExchange64, types.NewTuple(types.Types[types.TUINT64], types.TypeMem), args[0], args[1], s.mem())
			s.vars[memVar] = s.newValue1(ssa.OpSelect1, types.TypeMem, v)
			return s.newValue1(ssa.OpSelect0, types.Types[types.TUINT64], v)
		},
		sys.AMD64, sys.MIPS64, sys.PPC64, sys.RISCV64, sys.S390X)

	type atomicOpEmitter func(s *state, n *ir.CallExpr, args []*ssa.Value, op ssa.Op, typ types.Kind)

	makeAtomicGuardedIntrinsicARM64 := func(op0, op1 ssa.Op, typ, rtyp types.Kind, emit atomicOpEmitter) intrinsicBuilder {

		return func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			// Target Atomic feature is identified by dynamic detection
			addr := s.entryNewValue1A(ssa.OpAddr, types.Types[types.TBOOL].PtrTo(), ir.Syms.ARM64HasATOMICS, s.sb)
			v := s.load(types.Types[types.TBOOL], addr)
			b := s.endBlock()
			b.Kind = ssa.BlockIf
			b.SetControl(v)
			bTrue := s.f.NewBlock(ssa.BlockPlain)
			bFalse := s.f.NewBlock(ssa.BlockPlain)
			bEnd := s.f.NewBlock(ssa.BlockPlain)
			b.AddEdgeTo(bTrue)
			b.AddEdgeTo(bFalse)
			b.Likely = ssa.BranchLikely

			// We have atomic instructions - use it directly.
			s.startBlock(bTrue)
			emit(s, n, args, op1, typ)
			s.endBlock().AddEdgeTo(bEnd)

			// Use original instruction sequence.
			s.startBlock(bFalse)
			emit(s, n, args, op0, typ)
			s.endBlock().AddEdgeTo(bEnd)

			// Merge results.
			s.startBlock(bEnd)
			if rtyp == types.TNIL {
				return nil
			} else {
				return s.variable(n, types.Types[rtyp])
			}
		}
	}

	atomicXchgXaddEmitterARM64 := func(s *state, n *ir.CallExpr, args []*ssa.Value, op ssa.Op, typ types.Kind) {
		v := s.newValue3(op, types.NewTuple(types.Types[typ], types.TypeMem), args[0], args[1], s.mem())
		s.vars[memVar] = s.newValue1(ssa.OpSelect1, types.TypeMem, v)
		s.vars[n] = s.newValue1(ssa.OpSelect0, types.Types[typ], v)
	}
	addF("runtime/internal/atomic", "Xchg",
		makeAtomicGuardedIntrinsicARM64(ssa.OpAtomicExchange32, ssa.OpAtomicExchange32Variant, types.TUINT32, types.TUINT32, atomicXchgXaddEmitterARM64),
		sys.ARM64)
	addF("runtime/internal/atomic", "Xchg64",
		makeAtomicGuardedIntrinsicARM64(ssa.OpAtomicExchange64, ssa.OpAtomicExchange64Variant, types.TUINT64, types.TUINT64, atomicXchgXaddEmitterARM64),
		sys.ARM64)

	addF("runtime/internal/atomic", "Xadd",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			v := s.newValue3(ssa.OpAtomicAdd32, types.NewTuple(types.Types[types.TUINT32], types.TypeMem), args[0], args[1], s.mem())
			s.vars[memVar] = s.newValue1(ssa.OpSelect1, types.TypeMem, v)
			return s.newValue1(ssa.OpSelect0, types.Types[types.TUINT32], v)
		},
		sys.AMD64, sys.MIPS, sys.MIPS64, sys.PPC64, sys.RISCV64, sys.S390X)
	addF("runtime/internal/atomic", "Xadd64",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			v := s.newValue3(ssa.OpAtomicAdd64, types.NewTuple(types.Types[types.TUINT64], types.TypeMem), args[0], args[1], s.mem())
			s.vars[memVar] = s.newValue1(ssa.OpSelect1, types.TypeMem, v)
			return s.newValue1(ssa.OpSelect0, types.Types[types.TUINT64], v)
		},
		sys.AMD64, sys.MIPS64, sys.PPC64, sys.RISCV64, sys.S390X)

	addF("runtime/internal/atomic", "Xadd",
		makeAtomicGuardedIntrinsicARM64(ssa.OpAtomicAdd32, ssa.OpAtomicAdd32Variant, types.TUINT32, types.TUINT32, atomicXchgXaddEmitterARM64),
		sys.ARM64)
	addF("runtime/internal/atomic", "Xadd64",
		makeAtomicGuardedIntrinsicARM64(ssa.OpAtomicAdd64, ssa.OpAtomicAdd64Variant, types.TUINT64, types.TUINT64, atomicXchgXaddEmitterARM64),
		sys.ARM64)

	addF("runtime/internal/atomic", "Cas",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			v := s.newValue4(ssa.OpAtomicCompareAndSwap32, types.NewTuple(types.Types[types.TBOOL], types.TypeMem), args[0], args[1], args[2], s.mem())
			s.vars[memVar] = s.newValue1(ssa.OpSelect1, types.TypeMem, v)
			return s.newValue1(ssa.OpSelect0, types.Types[types.TBOOL], v)
		},
		sys.AMD64, sys.MIPS, sys.MIPS64, sys.PPC64, sys.RISCV64, sys.S390X)
	addF("runtime/internal/atomic", "Cas64",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			v := s.newValue4(ssa.OpAtomicCompareAndSwap64, types.NewTuple(types.Types[types.TBOOL], types.TypeMem), args[0], args[1], args[2], s.mem())
			s.vars[memVar] = s.newValue1(ssa.OpSelect1, types.TypeMem, v)
			return s.newValue1(ssa.OpSelect0, types.Types[types.TBOOL], v)
		},
		sys.AMD64, sys.MIPS64, sys.PPC64, sys.RISCV64, sys.S390X)
	addF("runtime/internal/atomic", "CasRel",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			v := s.newValue4(ssa.OpAtomicCompareAndSwap32, types.NewTuple(types.Types[types.TBOOL], types.TypeMem), args[0], args[1], args[2], s.mem())
			s.vars[memVar] = s.newValue1(ssa.OpSelect1, types.TypeMem, v)
			return s.newValue1(ssa.OpSelect0, types.Types[types.TBOOL], v)
		},
		sys.PPC64)

	atomicCasEmitterARM64 := func(s *state, n *ir.CallExpr, args []*ssa.Value, op ssa.Op, typ types.Kind) {
		v := s.newValue4(op, types.NewTuple(types.Types[types.TBOOL], types.TypeMem), args[0], args[1], args[2], s.mem())
		s.vars[memVar] = s.newValue1(ssa.OpSelect1, types.TypeMem, v)
		s.vars[n] = s.newValue1(ssa.OpSelect0, types.Types[typ], v)
	}

	addF("runtime/internal/atomic", "Cas",
		makeAtomicGuardedIntrinsicARM64(ssa.OpAtomicCompareAndSwap32, ssa.OpAtomicCompareAndSwap32Variant, types.TUINT32, types.TBOOL, atomicCasEmitterARM64),
		sys.ARM64)
	addF("runtime/internal/atomic", "Cas64",
		makeAtomicGuardedIntrinsicARM64(ssa.OpAtomicCompareAndSwap64, ssa.OpAtomicCompareAndSwap64Variant, types.TUINT64, types.TBOOL, atomicCasEmitterARM64),
		sys.ARM64)

	addF("runtime/internal/atomic", "And8",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			s.vars[memVar] = s.newValue3(ssa.OpAtomicAnd8, types.TypeMem, args[0], args[1], s.mem())
			return nil
		},
		sys.AMD64, sys.MIPS, sys.PPC64, sys.RISCV64, sys.S390X)
	addF("runtime/internal/atomic", "And",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			s.vars[memVar] = s.newValue3(ssa.OpAtomicAnd32, types.TypeMem, args[0], args[1], s.mem())
			return nil
		},
		sys.AMD64, sys.MIPS, sys.PPC64, sys.RISCV64, sys.S390X)
	addF("runtime/internal/atomic", "Or8",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			s.vars[memVar] = s.newValue3(ssa.OpAtomicOr8, types.TypeMem, args[0], args[1], s.mem())
			return nil
		},
		sys.AMD64, sys.ARM64, sys.MIPS, sys.PPC64, sys.RISCV64, sys.S390X)
	addF("runtime/internal/atomic", "Or",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			s.vars[memVar] = s.newValue3(ssa.OpAtomicOr32, types.TypeMem, args[0], args[1], s.mem())
			return nil
		},
		sys.AMD64, sys.MIPS, sys.PPC64, sys.RISCV64, sys.S390X)

	atomicAndOrEmitterARM64 := func(s *state, n *ir.CallExpr, args []*ssa.Value, op ssa.Op, typ types.Kind) {
		s.vars[memVar] = s.newValue3(op, types.TypeMem, args[0], args[1], s.mem())
	}

	addF("runtime/internal/atomic", "And8",
		makeAtomicGuardedIntrinsicARM64(ssa.OpAtomicAnd8, ssa.OpAtomicAnd8Variant, types.TNIL, types.TNIL, atomicAndOrEmitterARM64),
		sys.ARM64)
	addF("runtime/internal/atomic", "And",
		makeAtomicGuardedIntrinsicARM64(ssa.OpAtomicAnd32, ssa.OpAtomicAnd32Variant, types.TNIL, types.TNIL, atomicAndOrEmitterARM64),
		sys.ARM64)
	addF("runtime/internal/atomic", "Or8",
		makeAtomicGuardedIntrinsicARM64(ssa.OpAtomicOr8, ssa.OpAtomicOr8Variant, types.TNIL, types.TNIL, atomicAndOrEmitterARM64),
		sys.ARM64)
	addF("runtime/internal/atomic", "Or",
		makeAtomicGuardedIntrinsicARM64(ssa.OpAtomicOr32, ssa.OpAtomicOr32Variant, types.TNIL, types.TNIL, atomicAndOrEmitterARM64),
		sys.ARM64)

	// Aliases for atomic load operations
	alias("runtime/internal/atomic", "Loadint32", "runtime/internal/atomic", "Load", all...)
	alias("runtime/internal/atomic", "Loadint64", "runtime/internal/atomic", "Load64", all...)
	alias("runtime/internal/atomic", "Loaduintptr", "runtime/internal/atomic", "Load", p4...)
	alias("runtime/internal/atomic", "Loaduintptr", "runtime/internal/atomic", "Load64", p8...)
	alias("runtime/internal/atomic", "Loaduint", "runtime/internal/atomic", "Load", p4...)
	alias("runtime/internal/atomic", "Loaduint", "runtime/internal/atomic", "Load64", p8...)
	alias("runtime/internal/atomic", "LoadAcq", "runtime/internal/atomic", "Load", lwatomics...)
	alias("runtime/internal/atomic", "LoadAcq64", "runtime/internal/atomic", "Load64", lwatomics...)
	alias("runtime/internal/atomic", "LoadAcquintptr", "runtime/internal/atomic", "LoadAcq", p4...)
	alias("sync", "runtime_LoadAcquintptr", "runtime/internal/atomic", "LoadAcq", p4...) // linknamed
	alias("runtime/internal/atomic", "LoadAcquintptr", "runtime/internal/atomic", "LoadAcq64", p8...)
	alias("sync", "runtime_LoadAcquintptr", "runtime/internal/atomic", "LoadAcq64", p8...) // linknamed

	// Aliases for atomic store operations
	alias("runtime/internal/atomic", "Storeint32", "runtime/internal/atomic", "Store", all...)
	alias("runtime/internal/atomic", "Storeint64", "runtime/internal/atomic", "Store64", all...)
	alias("runtime/internal/atomic", "Storeuintptr", "runtime/internal/atomic", "Store", p4...)
	alias("runtime/internal/atomic", "Storeuintptr", "runtime/internal/atomic", "Store64", p8...)
	alias("runtime/internal/atomic", "StoreRel", "runtime/internal/atomic", "Store", lwatomics...)
	alias("runtime/internal/atomic", "StoreRel64", "runtime/internal/atomic", "Store64", lwatomics...)
	alias("runtime/internal/atomic", "StoreReluintptr", "runtime/internal/atomic", "StoreRel", p4...)
	alias("sync", "runtime_StoreReluintptr", "runtime/internal/atomic", "StoreRel", p4...) // linknamed
	alias("runtime/internal/atomic", "StoreReluintptr", "runtime/internal/atomic", "StoreRel64", p8...)
	alias("sync", "runtime_StoreReluintptr", "runtime/internal/atomic", "StoreRel64", p8...) // linknamed

	// Aliases for atomic swap operations
	alias("runtime/internal/atomic", "Xchgint32", "runtime/internal/atomic", "Xchg", all...)
	alias("runtime/internal/atomic", "Xchgint64", "runtime/internal/atomic", "Xchg64", all...)
	alias("runtime/internal/atomic", "Xchguintptr", "runtime/internal/atomic", "Xchg", p4...)
	alias("runtime/internal/atomic", "Xchguintptr", "runtime/internal/atomic", "Xchg64", p8...)

	// Aliases for atomic add operations
	alias("runtime/internal/atomic", "Xaddint32", "runtime/internal/atomic", "Xadd", all...)
	alias("runtime/internal/atomic", "Xaddint64", "runtime/internal/atomic", "Xadd64", all...)
	alias("runtime/internal/atomic", "Xadduintptr", "runtime/internal/atomic", "Xadd", p4...)
	alias("runtime/internal/atomic", "Xadduintptr", "runtime/internal/atomic", "Xadd64", p8...)

	// Aliases for atomic CAS operations
	alias("runtime/internal/atomic", "Casint32", "runtime/internal/atomic", "Cas", all...)
	alias("runtime/internal/atomic", "Casint64", "runtime/internal/atomic", "Cas64", all...)
	alias("runtime/internal/atomic", "Casuintptr", "runtime/internal/atomic", "Cas", p4...)
	alias("runtime/internal/atomic", "Casuintptr", "runtime/internal/atomic", "Cas64", p8...)
	alias("runtime/internal/atomic", "Casp1", "runtime/internal/atomic", "Cas", p4...)
	alias("runtime/internal/atomic", "Casp1", "runtime/internal/atomic", "Cas64", p8...)
	alias("runtime/internal/atomic", "CasRel", "runtime/internal/atomic", "Cas", lwatomics...)

	/******** math ********/
	addF("math", "Sqrt",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue1(ssa.OpSqrt, types.Types[types.TFLOAT64], args[0])
		},
		sys.I386, sys.AMD64, sys.ARM, sys.ARM64, sys.MIPS, sys.MIPS64, sys.PPC64, sys.RISCV64, sys.S390X, sys.Wasm)
	addF("math", "Trunc",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue1(ssa.OpTrunc, types.Types[types.TFLOAT64], args[0])
		},
		sys.ARM64, sys.PPC64, sys.S390X, sys.Wasm)
	addF("math", "Ceil",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue1(ssa.OpCeil, types.Types[types.TFLOAT64], args[0])
		},
		sys.ARM64, sys.PPC64, sys.S390X, sys.Wasm)
	addF("math", "Floor",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue1(ssa.OpFloor, types.Types[types.TFLOAT64], args[0])
		},
		sys.ARM64, sys.PPC64, sys.S390X, sys.Wasm)
	addF("math", "Round",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue1(ssa.OpRound, types.Types[types.TFLOAT64], args[0])
		},
		sys.ARM64, sys.PPC64, sys.S390X)
	addF("math", "RoundToEven",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue1(ssa.OpRoundToEven, types.Types[types.TFLOAT64], args[0])
		},
		sys.ARM64, sys.S390X, sys.Wasm)
	addF("math", "Abs",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue1(ssa.OpAbs, types.Types[types.TFLOAT64], args[0])
		},
		sys.ARM64, sys.ARM, sys.PPC64, sys.Wasm)
	addF("math", "Copysign",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue2(ssa.OpCopysign, types.Types[types.TFLOAT64], args[0], args[1])
		},
		sys.PPC64, sys.Wasm)
	addF("math", "FMA",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue3(ssa.OpFMA, types.Types[types.TFLOAT64], args[0], args[1], args[2])
		},
		sys.ARM64, sys.PPC64, sys.S390X)
	addF("math", "FMA",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			if !s.config.UseFMA {
				s.vars[n] = s.callResult(n, callNormal) // types.Types[TFLOAT64]
				return s.variable(n, types.Types[types.TFLOAT64])
			}
			v := s.entryNewValue0A(ssa.OpHasCPUFeature, types.Types[types.TBOOL], ir.Syms.X86HasFMA)
			b := s.endBlock()
			b.Kind = ssa.BlockIf
			b.SetControl(v)
			bTrue := s.f.NewBlock(ssa.BlockPlain)
			bFalse := s.f.NewBlock(ssa.BlockPlain)
			bEnd := s.f.NewBlock(ssa.BlockPlain)
			b.AddEdgeTo(bTrue)
			b.AddEdgeTo(bFalse)
			b.Likely = ssa.BranchLikely // >= haswell cpus are common

			// We have the intrinsic - use it directly.
			s.startBlock(bTrue)
			s.vars[n] = s.newValue3(ssa.OpFMA, types.Types[types.TFLOAT64], args[0], args[1], args[2])
			s.endBlock().AddEdgeTo(bEnd)

			// Call the pure Go version.
			s.startBlock(bFalse)
			s.vars[n] = s.callResult(n, callNormal) // types.Types[TFLOAT64]
			s.endBlock().AddEdgeTo(bEnd)

			// Merge results.
			s.startBlock(bEnd)
			return s.variable(n, types.Types[types.TFLOAT64])
		},
		sys.AMD64)
	addF("math", "FMA",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			if !s.config.UseFMA {
				s.vars[n] = s.callResult(n, callNormal) // types.Types[TFLOAT64]
				return s.variable(n, types.Types[types.TFLOAT64])
			}
			addr := s.entryNewValue1A(ssa.OpAddr, types.Types[types.TBOOL].PtrTo(), ir.Syms.ARMHasVFPv4, s.sb)
			v := s.load(types.Types[types.TBOOL], addr)
			b := s.endBlock()
			b.Kind = ssa.BlockIf
			b.SetControl(v)
			bTrue := s.f.NewBlock(ssa.BlockPlain)
			bFalse := s.f.NewBlock(ssa.BlockPlain)
			bEnd := s.f.NewBlock(ssa.BlockPlain)
			b.AddEdgeTo(bTrue)
			b.AddEdgeTo(bFalse)
			b.Likely = ssa.BranchLikely

			// We have the intrinsic - use it directly.
			s.startBlock(bTrue)
			s.vars[n] = s.newValue3(ssa.OpFMA, types.Types[types.TFLOAT64], args[0], args[1], args[2])
			s.endBlock().AddEdgeTo(bEnd)

			// Call the pure Go version.
			s.startBlock(bFalse)
			s.vars[n] = s.callResult(n, callNormal) // types.Types[TFLOAT64]
			s.endBlock().AddEdgeTo(bEnd)

			// Merge results.
			s.startBlock(bEnd)
			return s.variable(n, types.Types[types.TFLOAT64])
		},
		sys.ARM)

	makeRoundAMD64 := func(op ssa.Op) func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
		return func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			v := s.entryNewValue0A(ssa.OpHasCPUFeature, types.Types[types.TBOOL], ir.Syms.X86HasSSE41)
			b := s.endBlock()
			b.Kind = ssa.BlockIf
			b.SetControl(v)
			bTrue := s.f.NewBlock(ssa.BlockPlain)
			bFalse := s.f.NewBlock(ssa.BlockPlain)
			bEnd := s.f.NewBlock(ssa.BlockPlain)
			b.AddEdgeTo(bTrue)
			b.AddEdgeTo(bFalse)
			b.Likely = ssa.BranchLikely // most machines have sse4.1 nowadays

			// We have the intrinsic - use it directly.
			s.startBlock(bTrue)
			s.vars[n] = s.newValue1(op, types.Types[types.TFLOAT64], args[0])
			s.endBlock().AddEdgeTo(bEnd)

			// Call the pure Go version.
			s.startBlock(bFalse)
			s.vars[n] = s.callResult(n, callNormal) // types.Types[TFLOAT64]
			s.endBlock().AddEdgeTo(bEnd)

			// Merge results.
			s.startBlock(bEnd)
			return s.variable(n, types.Types[types.TFLOAT64])
		}
	}
	addF("math", "RoundToEven",
		makeRoundAMD64(ssa.OpRoundToEven),
		sys.AMD64)
	addF("math", "Floor",
		makeRoundAMD64(ssa.OpFloor),
		sys.AMD64)
	addF("math", "Ceil",
		makeRoundAMD64(ssa.OpCeil),
		sys.AMD64)
	addF("math", "Trunc",
		makeRoundAMD64(ssa.OpTrunc),
		sys.AMD64)

	/******** math/bits ********/
	addF("math/bits", "TrailingZeros64",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue1(ssa.OpCtz64, types.Types[types.TINT], args[0])
		},
		sys.AMD64, sys.ARM64, sys.ARM, sys.S390X, sys.MIPS, sys.PPC64, sys.Wasm)
	addF("math/bits", "TrailingZeros32",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue1(ssa.OpCtz32, types.Types[types.TINT], args[0])
		},
		sys.AMD64, sys.ARM64, sys.ARM, sys.S390X, sys.MIPS, sys.PPC64, sys.Wasm)
	addF("math/bits", "TrailingZeros16",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			x := s.newValue1(ssa.OpZeroExt16to32, types.Types[types.TUINT32], args[0])
			c := s.constInt32(types.Types[types.TUINT32], 1<<16)
			y := s.newValue2(ssa.OpOr32, types.Types[types.TUINT32], x, c)
			return s.newValue1(ssa.OpCtz32, types.Types[types.TINT], y)
		},
		sys.MIPS)
	addF("math/bits", "TrailingZeros16",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue1(ssa.OpCtz16, types.Types[types.TINT], args[0])
		},
		sys.AMD64, sys.I386, sys.ARM, sys.ARM64, sys.Wasm)
	addF("math/bits", "TrailingZeros16",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			x := s.newValue1(ssa.OpZeroExt16to64, types.Types[types.TUINT64], args[0])
			c := s.constInt64(types.Types[types.TUINT64], 1<<16)
			y := s.newValue2(ssa.OpOr64, types.Types[types.TUINT64], x, c)
			return s.newValue1(ssa.OpCtz64, types.Types[types.TINT], y)
		},
		sys.S390X, sys.PPC64)
	addF("math/bits", "TrailingZeros8",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			x := s.newValue1(ssa.OpZeroExt8to32, types.Types[types.TUINT32], args[0])
			c := s.constInt32(types.Types[types.TUINT32], 1<<8)
			y := s.newValue2(ssa.OpOr32, types.Types[types.TUINT32], x, c)
			return s.newValue1(ssa.OpCtz32, types.Types[types.TINT], y)
		},
		sys.MIPS)
	addF("math/bits", "TrailingZeros8",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue1(ssa.OpCtz8, types.Types[types.TINT], args[0])
		},
		sys.AMD64, sys.ARM, sys.ARM64, sys.Wasm)
	addF("math/bits", "TrailingZeros8",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			x := s.newValue1(ssa.OpZeroExt8to64, types.Types[types.TUINT64], args[0])
			c := s.constInt64(types.Types[types.TUINT64], 1<<8)
			y := s.newValue2(ssa.OpOr64, types.Types[types.TUINT64], x, c)
			return s.newValue1(ssa.OpCtz64, types.Types[types.TINT], y)
		},
		sys.S390X)
	alias("math/bits", "ReverseBytes64", "runtime/internal/sys", "Bswap64", all...)
	alias("math/bits", "ReverseBytes32", "runtime/internal/sys", "Bswap32", all...)
	// ReverseBytes inlines correctly, no need to intrinsify it.
	// ReverseBytes16 lowers to a rotate, no need for anything special here.
	addF("math/bits", "Len64",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue1(ssa.OpBitLen64, types.Types[types.TINT], args[0])
		},
		sys.AMD64, sys.ARM64, sys.ARM, sys.S390X, sys.MIPS, sys.PPC64, sys.Wasm)
	addF("math/bits", "Len32",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue1(ssa.OpBitLen32, types.Types[types.TINT], args[0])
		},
		sys.AMD64, sys.ARM64)
	addF("math/bits", "Len32",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			if s.config.PtrSize == 4 {
				return s.newValue1(ssa.OpBitLen32, types.Types[types.TINT], args[0])
			}
			x := s.newValue1(ssa.OpZeroExt32to64, types.Types[types.TUINT64], args[0])
			return s.newValue1(ssa.OpBitLen64, types.Types[types.TINT], x)
		},
		sys.ARM, sys.S390X, sys.MIPS, sys.PPC64, sys.Wasm)
	addF("math/bits", "Len16",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			if s.config.PtrSize == 4 {
				x := s.newValue1(ssa.OpZeroExt16to32, types.Types[types.TUINT32], args[0])
				return s.newValue1(ssa.OpBitLen32, types.Types[types.TINT], x)
			}
			x := s.newValue1(ssa.OpZeroExt16to64, types.Types[types.TUINT64], args[0])
			return s.newValue1(ssa.OpBitLen64, types.Types[types.TINT], x)
		},
		sys.ARM64, sys.ARM, sys.S390X, sys.MIPS, sys.PPC64, sys.Wasm)
	addF("math/bits", "Len16",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue1(ssa.OpBitLen16, types.Types[types.TINT], args[0])
		},
		sys.AMD64)
	addF("math/bits", "Len8",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			if s.config.PtrSize == 4 {
				x := s.newValue1(ssa.OpZeroExt8to32, types.Types[types.TUINT32], args[0])
				return s.newValue1(ssa.OpBitLen32, types.Types[types.TINT], x)
			}
			x := s.newValue1(ssa.OpZeroExt8to64, types.Types[types.TUINT64], args[0])
			return s.newValue1(ssa.OpBitLen64, types.Types[types.TINT], x)
		},
		sys.ARM64, sys.ARM, sys.S390X, sys.MIPS, sys.PPC64, sys.Wasm)
	addF("math/bits", "Len8",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue1(ssa.OpBitLen8, types.Types[types.TINT], args[0])
		},
		sys.AMD64)
	addF("math/bits", "Len",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			if s.config.PtrSize == 4 {
				return s.newValue1(ssa.OpBitLen32, types.Types[types.TINT], args[0])
			}
			return s.newValue1(ssa.OpBitLen64, types.Types[types.TINT], args[0])
		},
		sys.AMD64, sys.ARM64, sys.ARM, sys.S390X, sys.MIPS, sys.PPC64, sys.Wasm)
	// LeadingZeros is handled because it trivially calls Len.
	addF("math/bits", "Reverse64",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue1(ssa.OpBitRev64, types.Types[types.TINT], args[0])
		},
		sys.ARM64)
	addF("math/bits", "Reverse32",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue1(ssa.OpBitRev32, types.Types[types.TINT], args[0])
		},
		sys.ARM64)
	addF("math/bits", "Reverse16",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue1(ssa.OpBitRev16, types.Types[types.TINT], args[0])
		},
		sys.ARM64)
	addF("math/bits", "Reverse8",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue1(ssa.OpBitRev8, types.Types[types.TINT], args[0])
		},
		sys.ARM64)
	addF("math/bits", "Reverse",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			if s.config.PtrSize == 4 {
				return s.newValue1(ssa.OpBitRev32, types.Types[types.TINT], args[0])
			}
			return s.newValue1(ssa.OpBitRev64, types.Types[types.TINT], args[0])
		},
		sys.ARM64)
	addF("math/bits", "RotateLeft8",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue2(ssa.OpRotateLeft8, types.Types[types.TUINT8], args[0], args[1])
		},
		sys.AMD64)
	addF("math/bits", "RotateLeft16",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue2(ssa.OpRotateLeft16, types.Types[types.TUINT16], args[0], args[1])
		},
		sys.AMD64)
	addF("math/bits", "RotateLeft32",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue2(ssa.OpRotateLeft32, types.Types[types.TUINT32], args[0], args[1])
		},
		sys.AMD64, sys.ARM, sys.ARM64, sys.S390X, sys.PPC64, sys.Wasm)
	addF("math/bits", "RotateLeft64",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue2(ssa.OpRotateLeft64, types.Types[types.TUINT64], args[0], args[1])
		},
		sys.AMD64, sys.ARM64, sys.S390X, sys.PPC64, sys.Wasm)
	alias("math/bits", "RotateLeft", "math/bits", "RotateLeft64", p8...)

	makeOnesCountAMD64 := func(op64 ssa.Op, op32 ssa.Op) func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
		return func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			v := s.entryNewValue0A(ssa.OpHasCPUFeature, types.Types[types.TBOOL], ir.Syms.X86HasPOPCNT)
			b := s.endBlock()
			b.Kind = ssa.BlockIf
			b.SetControl(v)
			bTrue := s.f.NewBlock(ssa.BlockPlain)
			bFalse := s.f.NewBlock(ssa.BlockPlain)
			bEnd := s.f.NewBlock(ssa.BlockPlain)
			b.AddEdgeTo(bTrue)
			b.AddEdgeTo(bFalse)
			b.Likely = ssa.BranchLikely // most machines have popcnt nowadays

			// We have the intrinsic - use it directly.
			s.startBlock(bTrue)
			op := op64
			if s.config.PtrSize == 4 {
				op = op32
			}
			s.vars[n] = s.newValue1(op, types.Types[types.TINT], args[0])
			s.endBlock().AddEdgeTo(bEnd)

			// Call the pure Go version.
			s.startBlock(bFalse)
			s.vars[n] = s.callResult(n, callNormal) // types.Types[TINT]
			s.endBlock().AddEdgeTo(bEnd)

			// Merge results.
			s.startBlock(bEnd)
			return s.variable(n, types.Types[types.TINT])
		}
	}
	addF("math/bits", "OnesCount64",
		makeOnesCountAMD64(ssa.OpPopCount64, ssa.OpPopCount64),
		sys.AMD64)
	addF("math/bits", "OnesCount64",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue1(ssa.OpPopCount64, types.Types[types.TINT], args[0])
		},
		sys.PPC64, sys.ARM64, sys.S390X, sys.Wasm)
	addF("math/bits", "OnesCount32",
		makeOnesCountAMD64(ssa.OpPopCount32, ssa.OpPopCount32),
		sys.AMD64)
	addF("math/bits", "OnesCount32",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue1(ssa.OpPopCount32, types.Types[types.TINT], args[0])
		},
		sys.PPC64, sys.ARM64, sys.S390X, sys.Wasm)
	addF("math/bits", "OnesCount16",
		makeOnesCountAMD64(ssa.OpPopCount16, ssa.OpPopCount16),
		sys.AMD64)
	addF("math/bits", "OnesCount16",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue1(ssa.OpPopCount16, types.Types[types.TINT], args[0])
		},
		sys.ARM64, sys.S390X, sys.PPC64, sys.Wasm)
	addF("math/bits", "OnesCount8",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue1(ssa.OpPopCount8, types.Types[types.TINT], args[0])
		},
		sys.S390X, sys.PPC64, sys.Wasm)
	addF("math/bits", "OnesCount",
		makeOnesCountAMD64(ssa.OpPopCount64, ssa.OpPopCount32),
		sys.AMD64)
	addF("math/bits", "Mul64",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue2(ssa.OpMul64uhilo, types.NewTuple(types.Types[types.TUINT64], types.Types[types.TUINT64]), args[0], args[1])
		},
		sys.AMD64, sys.ARM64, sys.PPC64, sys.S390X, sys.MIPS64)
	alias("math/bits", "Mul", "math/bits", "Mul64", sys.ArchAMD64, sys.ArchARM64, sys.ArchPPC64, sys.ArchPPC64LE, sys.ArchS390X, sys.ArchMIPS64, sys.ArchMIPS64LE)
	alias("runtime/internal/math", "Mul64", "math/bits", "Mul64", sys.ArchAMD64, sys.ArchARM64, sys.ArchPPC64, sys.ArchPPC64LE, sys.ArchS390X, sys.ArchMIPS64, sys.ArchMIPS64LE)
	addF("math/bits", "Add64",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue3(ssa.OpAdd64carry, types.NewTuple(types.Types[types.TUINT64], types.Types[types.TUINT64]), args[0], args[1], args[2])
		},
		sys.AMD64, sys.ARM64, sys.PPC64, sys.S390X)
	alias("math/bits", "Add", "math/bits", "Add64", sys.ArchAMD64, sys.ArchARM64, sys.ArchPPC64, sys.ArchPPC64LE, sys.ArchS390X)
	addF("math/bits", "Sub64",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue3(ssa.OpSub64borrow, types.NewTuple(types.Types[types.TUINT64], types.Types[types.TUINT64]), args[0], args[1], args[2])
		},
		sys.AMD64, sys.ARM64, sys.S390X)
	alias("math/bits", "Sub", "math/bits", "Sub64", sys.ArchAMD64, sys.ArchARM64, sys.ArchS390X)
	addF("math/bits", "Div64",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			// check for divide-by-zero/overflow and panic with appropriate message
			cmpZero := s.newValue2(s.ssaOp(ir.ONE, types.Types[types.TUINT64]), types.Types[types.TBOOL], args[2], s.zeroVal(types.Types[types.TUINT64]))
			s.check(cmpZero, ir.Syms.Panicdivide)
			cmpOverflow := s.newValue2(s.ssaOp(ir.OLT, types.Types[types.TUINT64]), types.Types[types.TBOOL], args[0], args[2])
			s.check(cmpOverflow, ir.Syms.Panicoverflow)
			return s.newValue3(ssa.OpDiv128u, types.NewTuple(types.Types[types.TUINT64], types.Types[types.TUINT64]), args[0], args[1], args[2])
		},
		sys.AMD64)
	alias("math/bits", "Div", "math/bits", "Div64", sys.ArchAMD64)

	alias("runtime/internal/sys", "Ctz8", "math/bits", "TrailingZeros8", all...)
	alias("runtime/internal/sys", "TrailingZeros8", "math/bits", "TrailingZeros8", all...)
	alias("runtime/internal/sys", "TrailingZeros64", "math/bits", "TrailingZeros64", all...)
	alias("runtime/internal/sys", "Len8", "math/bits", "Len8", all...)
	alias("runtime/internal/sys", "Len64", "math/bits", "Len64", all...)
	alias("runtime/internal/sys", "OnesCount64", "math/bits", "OnesCount64", all...)

	/******** sync/atomic ********/

	// Note: these are disabled by flag_race in findIntrinsic below.
	alias("sync/atomic", "LoadInt32", "runtime/internal/atomic", "Load", all...)
	alias("sync/atomic", "LoadInt64", "runtime/internal/atomic", "Load64", all...)
	alias("sync/atomic", "LoadPointer", "runtime/internal/atomic", "Loadp", all...)
	alias("sync/atomic", "LoadUint32", "runtime/internal/atomic", "Load", all...)
	alias("sync/atomic", "LoadUint64", "runtime/internal/atomic", "Load64", all...)
	alias("sync/atomic", "LoadUintptr", "runtime/internal/atomic", "Load", p4...)
	alias("sync/atomic", "LoadUintptr", "runtime/internal/atomic", "Load64", p8...)

	alias("sync/atomic", "StoreInt32", "runtime/internal/atomic", "Store", all...)
	alias("sync/atomic", "StoreInt64", "runtime/internal/atomic", "Store64", all...)
	// Note: not StorePointer, that needs a write barrier.  Same below for {CompareAnd}Swap.
	alias("sync/atomic", "StoreUint32", "runtime/internal/atomic", "Store", all...)
	alias("sync/atomic", "StoreUint64", "runtime/internal/atomic", "Store64", all...)
	alias("sync/atomic", "StoreUintptr", "runtime/internal/atomic", "Store", p4...)
	alias("sync/atomic", "StoreUintptr", "runtime/internal/atomic", "Store64", p8...)

	alias("sync/atomic", "SwapInt32", "runtime/internal/atomic", "Xchg", all...)
	alias("sync/atomic", "SwapInt64", "runtime/internal/atomic", "Xchg64", all...)
	alias("sync/atomic", "SwapUint32", "runtime/internal/atomic", "Xchg", all...)
	alias("sync/atomic", "SwapUint64", "runtime/internal/atomic", "Xchg64", all...)
	alias("sync/atomic", "SwapUintptr", "runtime/internal/atomic", "Xchg", p4...)
	alias("sync/atomic", "SwapUintptr", "runtime/internal/atomic", "Xchg64", p8...)

	alias("sync/atomic", "CompareAndSwapInt32", "runtime/internal/atomic", "Cas", all...)
	alias("sync/atomic", "CompareAndSwapInt64", "runtime/internal/atomic", "Cas64", all...)
	alias("sync/atomic", "CompareAndSwapUint32", "runtime/internal/atomic", "Cas", all...)
	alias("sync/atomic", "CompareAndSwapUint64", "runtime/internal/atomic", "Cas64", all...)
	alias("sync/atomic", "CompareAndSwapUintptr", "runtime/internal/atomic", "Cas", p4...)
	alias("sync/atomic", "CompareAndSwapUintptr", "runtime/internal/atomic", "Cas64", p8...)

	alias("sync/atomic", "AddInt32", "runtime/internal/atomic", "Xadd", all...)
	alias("sync/atomic", "AddInt64", "runtime/internal/atomic", "Xadd64", all...)
	alias("sync/atomic", "AddUint32", "runtime/internal/atomic", "Xadd", all...)
	alias("sync/atomic", "AddUint64", "runtime/internal/atomic", "Xadd64", all...)
	alias("sync/atomic", "AddUintptr", "runtime/internal/atomic", "Xadd", p4...)
	alias("sync/atomic", "AddUintptr", "runtime/internal/atomic", "Xadd64", p8...)

	/******** math/big ********/
	add("math/big", "mulWW",
		func(s *state, n *ir.CallExpr, args []*ssa.Value) *ssa.Value {
			return s.newValue2(ssa.OpMul64uhilo, types.NewTuple(types.Types[types.TUINT64], types.Types[types.TUINT64]), args[0], args[1])
		},
		sys.ArchAMD64, sys.ArchARM64, sys.ArchPPC64LE, sys.ArchPPC64, sys.ArchS390X)
}

// findIntrinsic returns a function which builds the SSA equivalent of the
// function identified by the symbol sym.  If sym is not an intrinsic call, returns nil.
func findIntrinsic(sym *types.Sym) intrinsicBuilder {
	if sym == nil || sym.Pkg == nil {
		return nil
	}
	pkg := sym.Pkg.Path
	if sym.Pkg == types.LocalPkg {
		pkg = base.Ctxt.Pkgpath
	}
	if sym.Pkg == ir.Pkgs.Runtime {
		pkg = "runtime"
	}
	if base.Flag.Race && pkg == "sync/atomic" {
		// The race detector needs to be able to intercept these calls.
		// We can't intrinsify them.
		return nil
	}
	// Skip intrinsifying math functions (which may contain hard-float
	// instructions) when soft-float
	if Arch.SoftFloat && pkg == "math" {
		return nil
	}

	fn := sym.Name
	if ssa.IntrinsicsDisable {
		if pkg == "runtime" && (fn == "getcallerpc" || fn == "getcallersp" || fn == "getclosureptr") {
			// These runtime functions don't have definitions, must be intrinsics.
		} else {
			return nil
		}
	}
	return intrinsics[intrinsicKey{Arch.LinkArch.Arch, pkg, fn}]
}

func IsIntrinsicCall(n *ir.CallExpr) bool {
	if n == nil {
		return false
	}
	name, ok := n.X.(*ir.Name)
	if !ok {
		return false
	}
	return findIntrinsic(name.Sym()) != nil
}

// intrinsicCall converts a call to a recognized intrinsic function into the intrinsic SSA operation.
func (s *state) intrinsicCall(n *ir.CallExpr) *ssa.Value {
	v := findIntrinsic(n.X.Sym())(s, n, s.intrinsicArgs(n))
	if ssa.IntrinsicsDebug > 0 {
		x := v
		if x == nil {
			x = s.mem()
		}
		if x.Op == ssa.OpSelect0 || x.Op == ssa.OpSelect1 {
			x = x.Args[0]
		}
		base.WarnfAt(n.Pos(), "intrinsic substitution for %v with %s", n.X.Sym().Name, x.LongString())
	}
	return v
}

// intrinsicArgs extracts args from n, evaluates them to SSA values, and returns them.
func (s *state) intrinsicArgs(n *ir.CallExpr) []*ssa.Value {
	args := make([]*ssa.Value, len(n.Args))
	for i, n := range n.Args {
		args[i] = s.expr(n)
	}
	return args
}

// openDeferRecord adds code to evaluate and store the args for an open-code defer
// call, and records info about the defer, so we can generate proper code on the
// exit paths. n is the sub-node of the defer node that is the actual function
// call. We will also record funcdata information on where the args are stored
// (as well as the deferBits variable), and this will enable us to run the proper
// defer calls during panics.
func (s *state) openDeferRecord(n *ir.CallExpr) {
	var args []*ssa.Value
	var argNodes []*ir.Name

	if buildcfg.Experiment.RegabiDefer && (len(n.Args) != 0 || n.Op() == ir.OCALLINTER || n.X.Type().NumResults() != 0) {
		s.Fatalf("defer call with arguments or results: %v", n)
	}

	opendefer := &openDeferInfo{
		n: n,
	}
	fn := n.X
	if n.Op() == ir.OCALLFUNC {
		// We must always store the function value in a stack slot for the
		// runtime panic code to use. But in the defer exit code, we will
		// call the function directly if it is a static function.
		closureVal := s.expr(fn)
		closure := s.openDeferSave(nil, fn.Type(), closureVal)
		opendefer.closureNode = closure.Aux.(*ir.Name)
		if !(fn.Op() == ir.ONAME && fn.(*ir.Name).Class == ir.PFUNC) {
			opendefer.closure = closure
		}
	} else if n.Op() == ir.OCALLMETH {
		base.Fatalf("OCALLMETH missed by walkCall")
	} else {
		if fn.Op() != ir.ODOTINTER {
			base.Fatalf("OCALLINTER: n.Left not an ODOTINTER: %v", fn.Op())
		}
		fn := fn.(*ir.SelectorExpr)
		closure, rcvr := s.getClosureAndRcvr(fn)
		opendefer.closure = s.openDeferSave(nil, closure.Type, closure)
		// Important to get the receiver type correct, so it is recognized
		// as a pointer for GC purposes.
		opendefer.rcvr = s.openDeferSave(nil, fn.Type().Recv().Type, rcvr)
		opendefer.closureNode = opendefer.closure.Aux.(*ir.Name)
		opendefer.rcvrNode = opendefer.rcvr.Aux.(*ir.Name)
	}
	for _, argn := range n.Args {
		var v *ssa.Value
		if TypeOK(argn.Type()) {
			v = s.openDeferSave(nil, argn.Type(), s.expr(argn))
		} else {
			v = s.openDeferSave(argn, argn.Type(), nil)
		}
		args = append(args, v)
		argNodes = append(argNodes, v.Aux.(*ir.Name))
	}
	opendefer.argVals = args
	opendefer.argNodes = argNodes
	index := len(s.openDefers)
	s.openDefers = append(s.openDefers, opendefer)

	// Update deferBits only after evaluation and storage to stack of
	// args/receiver/interface is successful.
	bitvalue := s.constInt8(types.Types[types.TUINT8], 1<<uint(index))
	newDeferBits := s.newValue2(ssa.OpOr8, types.Types[types.TUINT8], s.variable(deferBitsVar, types.Types[types.TUINT8]), bitvalue)
	s.vars[deferBitsVar] = newDeferBits
	s.store(types.Types[types.TUINT8], s.deferBitsAddr, newDeferBits)
}

// openDeferSave generates SSA nodes to store a value (with type t) for an
// open-coded defer at an explicit autotmp location on the stack, so it can be
// reloaded and used for the appropriate call on exit. If type t is SSAable, then
// val must be non-nil (and n should be nil) and val is the value to be stored. If
// type t is non-SSAable, then n must be non-nil (and val should be nil) and n is
// evaluated (via s.addr() below) to get the value that is to be stored. The
// function returns an SSA value representing a pointer to the autotmp location.
func (s *state) openDeferSave(n ir.Node, t *types.Type, val *ssa.Value) *ssa.Value {
	canSSA := TypeOK(t)
	var pos src.XPos
	if canSSA {
		pos = val.Pos
	} else {
		pos = n.Pos()
	}
	argTemp := typecheck.TempAt(pos.WithNotStmt(), s.curfn, t)
	argTemp.SetOpenDeferSlot(true)
	var addrArgTemp *ssa.Value
	// Use OpVarLive to make sure stack slots for the args, etc. are not
	// removed by dead-store elimination
	if s.curBlock.ID != s.f.Entry.ID {
		// Force the argtmp storing this defer function/receiver/arg to be
		// declared in the entry block, so that it will be live for the
		// defer exit code (which will actually access it only if the
		// associated defer call has been activated).
		s.defvars[s.f.Entry.ID][memVar] = s.f.Entry.NewValue1A(src.NoXPos, ssa.OpVarDef, types.TypeMem, argTemp, s.defvars[s.f.Entry.ID][memVar])
		s.defvars[s.f.Entry.ID][memVar] = s.f.Entry.NewValue1A(src.NoXPos, ssa.OpVarLive, types.TypeMem, argTemp, s.defvars[s.f.Entry.ID][memVar])
		addrArgTemp = s.f.Entry.NewValue2A(src.NoXPos, ssa.OpLocalAddr, types.NewPtr(argTemp.Type()), argTemp, s.sp, s.defvars[s.f.Entry.ID][memVar])
	} else {
		// Special case if we're still in the entry block. We can't use
		// the above code, since s.defvars[s.f.Entry.ID] isn't defined
		// until we end the entry block with s.endBlock().
		s.vars[memVar] = s.newValue1Apos(ssa.OpVarDef, types.TypeMem, argTemp, s.mem(), false)
		s.vars[memVar] = s.newValue1Apos(ssa.OpVarLive, types.TypeMem, argTemp, s.mem(), false)
		addrArgTemp = s.newValue2Apos(ssa.OpLocalAddr, types.NewPtr(argTemp.Type()), argTemp, s.sp, s.mem(), false)
	}
	if t.HasPointers() {
		// Since we may use this argTemp during exit depending on the
		// deferBits, we must define it unconditionally on entry.
		// Therefore, we must make sure it is zeroed out in the entry
		// block if it contains pointers, else GC may wrongly follow an
		// uninitialized pointer value.
		argTemp.SetNeedzero(true)
	}
	if !canSSA {
		a := s.addr(n)
		s.move(t, addrArgTemp, a)
		return addrArgTemp
	}
	// We are storing to the stack, hence we can avoid the full checks in
	// storeType() (no write barrier) and do a simple store().
	s.store(t, addrArgTemp, val)
	return addrArgTemp
}

// openDeferExit generates SSA for processing all the open coded defers at exit.
// The code involves loading deferBits, and checking each of the bits to see if
// the corresponding defer statement was executed. For each bit that is turned
// on, the associated defer call is made.
func (s *state) openDeferExit() {
	deferExit := s.f.NewBlock(ssa.BlockPlain)
	s.endBlock().AddEdgeTo(deferExit)
	s.startBlock(deferExit)
	s.lastDeferExit = deferExit
	s.lastDeferCount = len(s.openDefers)
	zeroval := s.constInt8(types.Types[types.TUINT8], 0)
	// Test for and run defers in reverse order
	for i := len(s.openDefers) - 1; i >= 0; i-- {
		r := s.openDefers[i]
		bCond := s.f.NewBlock(ssa.BlockPlain)
		bEnd := s.f.NewBlock(ssa.BlockPlain)

		deferBits := s.variable(deferBitsVar, types.Types[types.TUINT8])
		// Generate code to check if the bit associated with the current
		// defer is set.
		bitval := s.constInt8(types.Types[types.TUINT8], 1<<uint(i))
		andval := s.newValue2(ssa.OpAnd8, types.Types[types.TUINT8], deferBits, bitval)
		eqVal := s.newValue2(ssa.OpEq8, types.Types[types.TBOOL], andval, zeroval)
		b := s.endBlock()
		b.Kind = ssa.BlockIf
		b.SetControl(eqVal)
		b.AddEdgeTo(bEnd)
		b.AddEdgeTo(bCond)
		bCond.AddEdgeTo(bEnd)
		s.startBlock(bCond)

		// Clear this bit in deferBits and force store back to stack, so
		// we will not try to re-run this defer call if this defer call panics.
		nbitval := s.newValue1(ssa.OpCom8, types.Types[types.TUINT8], bitval)
		maskedval := s.newValue2(ssa.OpAnd8, types.Types[types.TUINT8], deferBits, nbitval)
		s.store(types.Types[types.TUINT8], s.deferBitsAddr, maskedval)
		// Use this value for following tests, so we keep previous
		// bits cleared.
		s.vars[deferBitsVar] = maskedval

		// Generate code to call the function call of the defer, using the
		// closure/receiver/args that were stored in argtmps at the point
		// of the defer statement.
		fn := r.n.X
		stksize := fn.Type().ArgWidth()
		var ACArgs []*types.Type
		var ACResults []*types.Type
		var callArgs []*ssa.Value
		if r.rcvr != nil {
			// rcvr in case of OCALLINTER
			v := s.load(r.rcvr.Type.Elem(), r.rcvr)
			ACArgs = append(ACArgs, types.Types[types.TUINTPTR])
			callArgs = append(callArgs, v)
		}
		for j, argAddrVal := range r.argVals {
			f := getParam(r.n, j)
			ACArgs = append(ACArgs, f.Type)
			var a *ssa.Value
			if !TypeOK(f.Type) {
				a = s.newValue2(ssa.OpDereference, f.Type, argAddrVal, s.mem())
			} else {
				a = s.load(f.Type, argAddrVal)
			}
			callArgs = append(callArgs, a)
		}
		var call *ssa.Value
		if r.closure != nil {
			v := s.load(r.closure.Type.Elem(), r.closure)
			s.maybeNilCheckClosure(v, callDefer)
			codeptr := s.rawLoad(types.Types[types.TUINTPTR], v)
			aux := ssa.ClosureAuxCall(s.f.ABIDefault.ABIAnalyzeTypes(nil, ACArgs, ACResults))
			call = s.newValue2A(ssa.OpClosureLECall, aux.LateExpansionResultType(), aux, codeptr, v)
		} else {
			aux := ssa.StaticAuxCall(fn.(*ir.Name).Linksym(), s.f.ABIDefault.ABIAnalyzeTypes(nil, ACArgs, ACResults))
			call = s.newValue0A(ssa.OpStaticLECall, aux.LateExpansionResultType(), aux)
		}
		callArgs = append(callArgs, s.mem())
		call.AddArgs(callArgs...)
		call.AuxInt = stksize
		s.vars[memVar] = s.newValue1I(ssa.OpSelectN, types.TypeMem, int64(len(ACResults)), call)
		// Make sure that the stack slots with pointers are kept live
		// through the call (which is a pre-emption point). Also, we will
		// use the first call of the last defer exit to compute liveness
		// for the deferreturn, so we want all stack slots to be live.
		if r.closureNode != nil {
			s.vars[memVar] = s.newValue1Apos(ssa.OpVarLive, types.TypeMem, r.closureNode, s.mem(), false)
		}
		if r.rcvrNode != nil {
			if r.rcvrNode.Type().HasPointers() {
				s.vars[memVar] = s.newValue1Apos(ssa.OpVarLive, types.TypeMem, r.rcvrNode, s.mem(), false)
			}
		}
		for _, argNode := range r.argNodes {
			if argNode.Type().HasPointers() {
				s.vars[memVar] = s.newValue1Apos(ssa.OpVarLive, types.TypeMem, argNode, s.mem(), false)
			}
		}

		s.endBlock()
		s.startBlock(bEnd)
	}
}

func (s *state) callResult(n *ir.CallExpr, k callKind) *ssa.Value {
	return s.call(n, k, false)
}

func (s *state) callAddr(n *ir.CallExpr, k callKind) *ssa.Value {
	return s.call(n, k, true)
}

// Calls the function n using the specified call type.
// Returns the address of the return value (or nil if none).
func (s *state) call(n *ir.CallExpr, k callKind, returnResultAddr bool) *ssa.Value {
	s.prevCall = nil
	var callee *ir.Name    // target function (if static)
	var closure *ssa.Value // ptr to closure to run (if dynamic)
	var codeptr *ssa.Value // ptr to target code (if dynamic)
	var rcvr *ssa.Value    // receiver to set
	fn := n.X
	var ACArgs []*types.Type    // AuxCall args
	var ACResults []*types.Type // AuxCall results
	var callArgs []*ssa.Value   // For late-expansion, the args themselves (not stored, args to the call instead).

	callABI := s.f.ABIDefault

	if !buildcfg.Experiment.RegabiArgs {
		var magicFnNameSym *types.Sym
		if fn.Name() != nil {
			magicFnNameSym = fn.Name().Sym()
			ss := magicFnNameSym.Name
			if strings.HasSuffix(ss, magicNameDotSuffix) {
				callABI = s.f.ABI1
			}
		}
		if magicFnNameSym == nil && n.Op() == ir.OCALLINTER {
			magicFnNameSym = fn.(*ir.SelectorExpr).Sym()
			ss := magicFnNameSym.Name
			if strings.HasSuffix(ss, magicNameDotSuffix[1:]) {
				callABI = s.f.ABI1
			}
		}
	}

	if buildcfg.Experiment.RegabiDefer && k != callNormal && (len(n.Args) != 0 || n.Op() == ir.OCALLINTER || n.X.Type().NumResults() != 0) {
		s.Fatalf("go/defer call with arguments: %v", n)
	}

	switch n.Op() {
	case ir.OCALLFUNC:
		if k == callNormal && fn.Op() == ir.ONAME && fn.(*ir.Name).Class == ir.PFUNC {
			fn := fn.(*ir.Name)
			callee = fn
			if buildcfg.Experiment.RegabiArgs {
				// This is a static call, so it may be
				// a direct call to a non-ABIInternal
				// function. fn.Func may be nil for
				// some compiler-generated functions,
				// but those are all ABIInternal.
				if fn.Func != nil {
					callABI = abiForFunc(fn.Func, s.f.ABI0, s.f.ABI1)
				}
			} else {
				// TODO(register args) remove after register abi is working
				inRegistersImported := fn.Pragma()&ir.RegisterParams != 0
				inRegistersSamePackage := fn.Func != nil && fn.Func.Pragma&ir.RegisterParams != 0
				if inRegistersImported || inRegistersSamePackage {
					callABI = s.f.ABI1
				}
			}
			break
		}
		closure = s.expr(fn)
		if k != callDefer && k != callDeferStack {
			// Deferred nil function needs to panic when the function is invoked,
			// not the point of defer statement.
			s.maybeNilCheckClosure(closure, k)
		}
	case ir.OCALLMETH:
		base.Fatalf("OCALLMETH missed by walkCall")
	case ir.OCALLINTER:
		if fn.Op() != ir.ODOTINTER {
			s.Fatalf("OCALLINTER: n.Left not an ODOTINTER: %v", fn.Op())
		}
		fn := fn.(*ir.SelectorExpr)
		var iclosure *ssa.Value
		iclosure, rcvr = s.getClosureAndRcvr(fn)
		if k == callNormal {
			codeptr = s.load(types.Types[types.TUINTPTR], iclosure)
		} else {
			closure = iclosure
		}
	}

	if !buildcfg.Experiment.RegabiArgs {
		if regAbiForFuncType(n.X.Type().FuncType()) {
			// Magic last type in input args to call
			callABI = s.f.ABI1
		}
	}

	params := callABI.ABIAnalyze(n.X.Type(), false /* Do not set (register) nNames from caller side -- can cause races. */)
	types.CalcSize(fn.Type())
	stksize := params.ArgWidth() // includes receiver, args, and results

	res := n.X.Type().Results()
	if k == callNormal {
		for _, p := range params.OutParams() {
			ACResults = append(ACResults, p.Type)
		}
	}

	var call *ssa.Value
	if k == callDeferStack {
		// Make a defer struct d on the stack.
		t := deferstruct(stksize)
		d := typecheck.TempAt(n.Pos(), s.curfn, t)

		s.vars[memVar] = s.newValue1A(ssa.OpVarDef, types.TypeMem, d, s.mem())
		addr := s.addr(d)

		// Must match reflect.go:deferstruct and src/runtime/runtime2.go:_defer.
		// 0: siz
		s.store(types.Types[types.TUINT32],
			s.newValue1I(ssa.OpOffPtr, types.Types[types.TUINT32].PtrTo(), t.FieldOff(0), addr),
			s.constInt32(types.Types[types.TUINT32], int32(stksize)))
		// 1: started, set in deferprocStack
		// 2: heap, set in deferprocStack
		// 3: openDefer
		// 4: sp, set in deferprocStack
		// 5: pc, set in deferprocStack
		// 6: fn
		s.store(closure.Type,
			s.newValue1I(ssa.OpOffPtr, closure.Type.PtrTo(), t.FieldOff(6), addr),
			closure)
		// 7: panic, set in deferprocStack
		// 8: link, set in deferprocStack
		// 9: framepc
		// 10: varp
		// 11: fd

		// Then, store all the arguments of the defer call.
		ft := fn.Type()
		off := t.FieldOff(12) // TODO register args: be sure this isn't a hardcoded param stack offset.
		args := n.Args
		i0 := 0

		// Set receiver (for interface calls). Always a pointer.
		if rcvr != nil {
			p := s.newValue1I(ssa.OpOffPtr, ft.Recv().Type.PtrTo(), off, addr)
			s.store(types.Types[types.TUINTPTR], p, rcvr)
			i0 = 1
		}
		// Set receiver (for method calls).
		if n.Op() == ir.OCALLMETH {
			base.Fatalf("OCALLMETH missed by walkCall")
		}
		// Set other args.
		// This code is only used when RegabiDefer is not enabled, and arguments are always
		// passed on stack.
		for i, f := range ft.Params().Fields().Slice() {
			s.storeArgWithBase(args[0], f.Type, addr, off+params.InParam(i+i0).FrameOffset(params))
			args = args[1:]
		}

		// Call runtime.deferprocStack with pointer to _defer record.
		ACArgs = append(ACArgs, types.Types[types.TUINTPTR])
		aux := ssa.StaticAuxCall(ir.Syms.DeferprocStack, s.f.ABIDefault.ABIAnalyzeTypes(nil, ACArgs, ACResults))
		callArgs = append(callArgs, addr, s.mem())
		call = s.newValue0A(ssa.OpStaticLECall, aux.LateExpansionResultType(), aux)
		call.AddArgs(callArgs...)
		if stksize < int64(types.PtrSize) {
			// We need room for both the call to deferprocStack and the call to
			// the deferred function.
			stksize = int64(types.PtrSize)
		}
		call.AuxInt = stksize
	} else {
		// Store arguments to stack, including defer/go arguments and receiver for method calls.
		// These are written in SP-offset order.
		argStart := base.Ctxt.FixedFrameSize()
		// Defer/go args.
		if k != callNormal {
			// Write argsize and closure (args to newproc/deferproc).
			argsize := s.constInt32(types.Types[types.TUINT32], int32(stksize))
			ACArgs = append(ACArgs, types.Types[types.TUINT32]) // not argExtra
			callArgs = append(callArgs, argsize)
			ACArgs = append(ACArgs, types.Types[types.TUINTPTR])
			callArgs = append(callArgs, closure)
			stksize += 2 * int64(types.PtrSize)
			argStart += 2 * int64(types.PtrSize)
		}

		// Set receiver (for interface calls).
		if rcvr != nil {
			callArgs = append(callArgs, rcvr)
		}

		// Write args.
		t := n.X.Type()
		args := n.Args
		if n.Op() == ir.OCALLMETH {
			base.Fatalf("OCALLMETH missed by walkCall")
		}

		for _, p := range params.InParams() { // includes receiver for interface calls
			ACArgs = append(ACArgs, p.Type)
		}
		for i, n := range args {
			callArgs = append(callArgs, s.putArg(n, t.Params().Field(i).Type))
		}

		callArgs = append(callArgs, s.mem())

		// call target
		switch {
		case k == callDefer:
			aux := ssa.StaticAuxCall(ir.Syms.Deferproc, s.f.ABIDefault.ABIAnalyzeTypes(nil, ACArgs, ACResults)) // TODO paramResultInfo for DeferProc
			call = s.newValue0A(ssa.OpStaticLECall, aux.LateExpansionResultType(), aux)
		case k == callGo:
			aux := ssa.StaticAuxCall(ir.Syms.Newproc, s.f.ABIDefault.ABIAnalyzeTypes(nil, ACArgs, ACResults))
			call = s.newValue0A(ssa.OpStaticLECall, aux.LateExpansionResultType(), aux) // TODO paramResultInfo for NewProc
		case closure != nil:
			// rawLoad because loading the code pointer from a
			// closure is always safe, but IsSanitizerSafeAddr
			// can't always figure that out currently, and it's
			// critical that we not clobber any arguments already
			// stored onto the stack.
			codeptr = s.rawLoad(types.Types[types.TUINTPTR], closure)
			aux := ssa.ClosureAuxCall(callABI.ABIAnalyzeTypes(nil, ACArgs, ACResults))
			call = s.newValue2A(ssa.OpClosureLECall, aux.LateExpansionResultType(), aux, codeptr, closure)
		case codeptr != nil:
			// Note that the "receiver" parameter is nil because the actual receiver is the first input parameter.
			aux := ssa.InterfaceAuxCall(params)
			call = s.newValue1A(ssa.OpInterLECall, aux.LateExpansionResultType(), aux, codeptr)
		case callee != nil:
			aux := ssa.StaticAuxCall(callTargetLSym(callee), params)
			call = s.newValue0A(ssa.OpStaticLECall, aux.LateExpansionResultType(), aux)
		default:
			s.Fatalf("bad call type %v %v", n.Op(), n)
		}
		call.AddArgs(callArgs...)
		call.AuxInt = stksize // Call operations carry the argsize of the callee along with them
	}
	s.prevCall = call
	s.vars[memVar] = s.newValue1I(ssa.OpSelectN, types.TypeMem, int64(len(ACResults)), call)
	// Insert OVARLIVE nodes
	for _, name := range n.KeepAlive {
		s.stmt(ir.NewUnaryExpr(n.Pos(), ir.OVARLIVE, name))
	}

	// Finish block for defers
	if k == callDefer || k == callDeferStack {
		b := s.endBlock()
		b.Kind = ssa.BlockDefer
		b.SetControl(call)
		bNext := s.f.NewBlock(ssa.BlockPlain)
		b.AddEdgeTo(bNext)
		// Add recover edge to exit code.
		r := s.f.NewBlock(ssa.BlockPlain)
		s.startBlock(r)
		s.exit()
		b.AddEdgeTo(r)
		b.Likely = ssa.BranchLikely
		s.startBlock(bNext)
	}

	if res.NumFields() == 0 || k != callNormal {
		// call has no return value. Continue with the next statement.
		return nil
	}
	fp := res.Field(0)
	if returnResultAddr {
		return s.resultAddrOfCall(call, 0, fp.Type)
	}
	return s.newValue1I(ssa.OpSelectN, fp.Type, 0, call)
}

// maybeNilCheckClosure checks if a nil check of a closure is needed in some
// architecture-dependent situations and, if so, emits the nil check.
func (s *state) maybeNilCheckClosure(closure *ssa.Value, k callKind) {
	if Arch.LinkArch.Family == sys.Wasm || buildcfg.GOOS == "aix" && k != callGo {
		// On AIX, the closure needs to be verified as fn can be nil, except if it's a call go. This needs to be handled by the runtime to have the "go of nil func value" error.
		// TODO(neelance): On other architectures this should be eliminated by the optimization steps
		s.nilCheck(closure)
	}
}

// getClosureAndRcvr returns values for the appropriate closure and receiver of an
// interface call
func (s *state) getClosureAndRcvr(fn *ir.SelectorExpr) (*ssa.Value, *ssa.Value) {
	i := s.expr(fn.X)
	itab := s.newValue1(ssa.OpITab, types.Types[types.TUINTPTR], i)
	s.nilCheck(itab)
	itabidx := fn.Offset() + 2*int64(types.PtrSize) + 8 // offset of fun field in runtime.itab
	closure := s.newValue1I(ssa.OpOffPtr, s.f.Config.Types.UintptrPtr, itabidx, itab)
	rcvr := s.newValue1(ssa.OpIData, s.f.Config.Types.BytePtr, i)
	return closure, rcvr
}

// etypesign returns the signed-ness of e, for integer/pointer etypes.
// -1 means signed, +1 means unsigned, 0 means non-integer/non-pointer.
func etypesign(e types.Kind) int8 {
	switch e {
	case types.TINT8, types.TINT16, types.TINT32, types.TINT64, types.TINT:
		return -1
	case types.TUINT8, types.TUINT16, types.TUINT32, types.TUINT64, types.TUINT, types.TUINTPTR, types.TUNSAFEPTR:
		return +1
	}
	return 0
}

// addr converts the address of the expression n to SSA, adds it to s and returns the SSA result.
// The value that the returned Value represents is guaranteed to be non-nil.
func (s *state) addr(n ir.Node) *ssa.Value {
	if n.Op() != ir.ONAME {
		s.pushLine(n.Pos())
		defer s.popLine()
	}

	if s.canSSA(n) {
		s.Fatalf("addr of canSSA expression: %+v", n)
	}

	t := types.NewPtr(n.Type())
	linksymOffset := func(lsym *obj.LSym, offset int64) *ssa.Value {
		v := s.entryNewValue1A(ssa.OpAddr, t, lsym, s.sb)
		// TODO: Make OpAddr use AuxInt as well as Aux.
		if offset != 0 {
			v = s.entryNewValue1I(ssa.OpOffPtr, v.Type, offset, v)
		}
		return v
	}
	switch n.Op() {
	case ir.OLINKSYMOFFSET:
		no := n.(*ir.LinksymOffsetExpr)
		return linksymOffset(no.Linksym, no.Offset_)
	case ir.ONAME:
		n := n.(*ir.Name)
		if n.Heapaddr != nil {
			return s.expr(n.Heapaddr)
		}
		switch n.Class {
		case ir.PEXTERN:
			// global variable
			return linksymOffset(n.Linksym(), 0)
		case ir.PPARAM:
			// parameter slot
			v := s.decladdrs[n]
			if v != nil {
				return v
			}
			s.Fatalf("addr of undeclared ONAME %v. declared: %v", n, s.decladdrs)
			return nil
		case ir.PAUTO:
			return s.newValue2Apos(ssa.OpLocalAddr, t, n, s.sp, s.mem(), !ir.IsAutoTmp(n))

		case ir.PPARAMOUT: // Same as PAUTO -- cannot generate LEA early.
			// ensure that we reuse symbols for out parameters so
			// that cse works on their addresses
			return s.newValue2Apos(ssa.OpLocalAddr, t, n, s.sp, s.mem(), true)
		default:
			s.Fatalf("variable address class %v not implemented", n.Class)
			return nil
		}
	case ir.ORESULT:
		// load return from callee
		n := n.(*ir.ResultExpr)
		return s.resultAddrOfCall(s.prevCall, n.Index, n.Type())
	case ir.OINDEX:
		n := n.(*ir.IndexExpr)
		if n.X.Type().IsSlice() {
			a := s.expr(n.X)
			i := s.expr(n.Index)
			len := s.newValue1(ssa.OpSliceLen, types.Types[types.TINT], a)
			i = s.boundsCheck(i, len, ssa.BoundsIndex, n.Bounded())
			p := s.newValue1(ssa.OpSlicePtr, t, a)
			return s.newValue2(ssa.OpPtrIndex, t, p, i)
		} else { // array
			a := s.addr(n.X)
			i := s.expr(n.Index)
			len := s.constInt(types.Types[types.TINT], n.X.Type().NumElem())
			i = s.boundsCheck(i, len, ssa.BoundsIndex, n.Bounded())
			return s.newValue2(ssa.OpPtrIndex, types.NewPtr(n.X.Type().Elem()), a, i)
		}
	case ir.ODEREF:
		n := n.(*ir.StarExpr)
		return s.exprPtr(n.X, n.Bounded(), n.Pos())
	case ir.ODOT:
		n := n.(*ir.SelectorExpr)
		p := s.addr(n.X)
		return s.newValue1I(ssa.OpOffPtr, t, n.Offset(), p)
	case ir.ODOTPTR:
		n := n.(*ir.SelectorExpr)
		p := s.exprPtr(n.X, n.Bounded(), n.Pos())
		return s.newValue1I(ssa.OpOffPtr, t, n.Offset(), p)
	case ir.OCONVNOP:
		n := n.(*ir.ConvExpr)
		if n.Type() == n.X.Type() {
			return s.addr(n.X)
		}
		addr := s.addr(n.X)
		return s.newValue1(ssa.OpCopy, t, addr) // ensure that addr has the right type
	case ir.OCALLFUNC, ir.OCALLINTER:
		n := n.(*ir.CallExpr)
		return s.callAddr(n, callNormal)
	case ir.ODOTTYPE:
		n := n.(*ir.TypeAssertExpr)
		v, _ := s.dottype(n, false)
		if v.Op != ssa.OpLoad {
			s.Fatalf("dottype of non-load")
		}
		if v.Args[1] != s.mem() {
			s.Fatalf("memory no longer live from dottype load")
		}
		return v.Args[0]
	default:
		s.Fatalf("unhandled addr %v", n.Op())
		return nil
	}
}

// canSSA reports whether n is SSA-able.
// n must be an ONAME (or an ODOT sequence with an ONAME base).
func (s *state) canSSA(n ir.Node) bool {
	if base.Flag.N != 0 {
		return false
	}
	for {
		nn := n
		if nn.Op() == ir.ODOT {
			nn := nn.(*ir.SelectorExpr)
			n = nn.X
			continue
		}
		if nn.Op() == ir.OINDEX {
			nn := nn.(*ir.IndexExpr)
			if nn.X.Type().IsArray() {
				n = nn.X
				continue
			}
		}
		break
	}
	if n.Op() != ir.ONAME {
		return false
	}
	return s.canSSAName(n.(*ir.Name)) && TypeOK(n.Type())
}

func (s *state) canSSAName(name *ir.Name) bool {
	if name.Addrtaken() || !name.OnStack() {
		return false
	}
	switch name.Class {
	case ir.PPARAMOUT:
		if s.hasdefer {
			// TODO: handle this case? Named return values must be
			// in memory so that the deferred function can see them.
			// Maybe do: if !strings.HasPrefix(n.String(), "~") { return false }
			// Or maybe not, see issue 18860.  Even unnamed return values
			// must be written back so if a defer recovers, the caller can see them.
			return false
		}
		if s.cgoUnsafeArgs {
			// Cgo effectively takes the address of all result args,
			// but the compiler can't see that.
			return false
		}
	}
	if name.Class == ir.PPARAM && name.Sym() != nil && name.Sym().Name == ".this" {
		// wrappers generated by genwrapper need to update
		// the .this pointer in place.
		// TODO: treat as a PPARAMOUT?
		return false
	}
	return true
	// TODO: try to make more variables SSAable?
}

// TypeOK reports whether variables of type t are SSA-able.
func TypeOK(t *types.Type) bool {
	types.CalcSize(t)
	if t.Width > int64(4*types.PtrSize) {
		// 4*Widthptr is an arbitrary constant. We want it
		// to be at least 3*Widthptr so slices can be registerized.
		// Too big and we'll introduce too much register pressure.
		return false
	}
	switch t.Kind() {
	case types.TARRAY:
		// We can't do larger arrays because dynamic indexing is
		// not supported on SSA variables.
		// TODO: allow if all indexes are constant.
		if t.NumElem() <= 1 {
			return TypeOK(t.Elem())
		}
		return false
	case types.TSTRUCT:
		if t.NumFields() > ssa.MaxStruct {
			return false
		}
		for _, t1 := range t.Fields().Slice() {
			if !TypeOK(t1.Type) {
				return false
			}
		}
		return true
	default:
		return true
	}
}

// exprPtr evaluates n to a pointer and nil-checks it.
func (s *state) exprPtr(n ir.Node, bounded bool, lineno src.XPos) *ssa.Value {
	p := s.expr(n)
	if bounded || n.NonNil() {
		if s.f.Frontend().Debug_checknil() && lineno.Line() > 1 {
			s.f.Warnl(lineno, "removed nil check")
		}
		return p
	}
	s.nilCheck(p)
	return p
}

// nilCheck generates nil pointer checking code.
// Used only for automatically inserted nil checks,
// not for user code like 'x != nil'.
func (s *state) nilCheck(ptr *ssa.Value) {
	if base.Debug.DisableNil != 0 || s.curfn.NilCheckDisabled() {
		return
	}
	s.newValue2(ssa.OpNilCheck, types.TypeVoid, ptr, s.mem())
}

// boundsCheck generates bounds checking code. Checks if 0 <= idx <[=] len, branches to exit if not.
// Starts a new block on return.
// On input, len must be converted to full int width and be nonnegative.
// Returns idx converted to full int width.
// If bounded is true then caller guarantees the index is not out of bounds
// (but boundsCheck will still extend the index to full int width).
func (s *state) boundsCheck(idx, len *ssa.Value, kind ssa.BoundsKind, bounded bool) *ssa.Value {
	idx = s.extendIndex(idx, len, kind, bounded)

	if bounded || base.Flag.B != 0 {
		// If bounded or bounds checking is flag-disabled, then no check necessary,
		// just return the extended index.
		//
		// Here, bounded == true if the compiler generated the index itself,
		// such as in the expansion of a slice initializer. These indexes are
		// compiler-generated, not Go program variables, so they cannot be
		// attacker-controlled, so we can omit Spectre masking as well.
		//
		// Note that we do not want to omit Spectre masking in code like:
		//
		//	if 0 <= i && i < len(x) {
		//		use(x[i])
		//	}
		//
		// Lucky for us, bounded==false for that code.
		// In that case (handled below), we emit a bound check (and Spectre mask)
		// and then the prove pass will remove the bounds check.
		// In theory the prove pass could potentially remove certain
		// Spectre masks, but it's very delicate and probably better
		// to be conservative and leave them all in.
		return idx
	}

	bNext := s.f.NewBlock(ssa.BlockPlain)
	bPanic := s.f.NewBlock(ssa.BlockExit)

	if !idx.Type.IsSigned() {
		switch kind {
		case ssa.BoundsIndex:
			kind = ssa.BoundsIndexU
		case ssa.BoundsSliceAlen:
			kind = ssa.BoundsSliceAlenU
		case ssa.BoundsSliceAcap:
			kind = ssa.BoundsSliceAcapU
		case ssa.BoundsSliceB:
			kind = ssa.BoundsSliceBU
		case ssa.BoundsSlice3Alen:
			kind = ssa.BoundsSlice3AlenU
		case ssa.BoundsSlice3Acap:
			kind = ssa.BoundsSlice3AcapU
		case ssa.BoundsSlice3B:
			kind = ssa.BoundsSlice3BU
		case ssa.BoundsSlice3C:
			kind = ssa.BoundsSlice3CU
		}
	}

	var cmp *ssa.Value
	if kind == ssa.BoundsIndex || kind == ssa.BoundsIndexU {
		cmp = s.newValue2(ssa.OpIsInBounds, types.Types[types.TBOOL], idx, len)
	} else {
		cmp = s.newValue2(ssa.OpIsSliceInBounds, types.Types[types.TBOOL], idx, len)
	}
	b := s.endBlock()
	b.Kind = ssa.BlockIf
	b.SetControl(cmp)
	b.Likely = ssa.BranchLikely
	b.AddEdgeTo(bNext)
	b.AddEdgeTo(bPanic)

	s.startBlock(bPanic)
	if Arch.LinkArch.Family == sys.Wasm {
		// TODO(khr): figure out how to do "register" based calling convention for bounds checks.
		// Should be similar to gcWriteBarrier, but I can't make it work.
		s.rtcall(BoundsCheckFunc[kind], false, nil, idx, len)
	} else {
		mem := s.newValue3I(ssa.OpPanicBounds, types.TypeMem, int64(kind), idx, len, s.mem())
		s.endBlock().SetControl(mem)
	}
	s.startBlock(bNext)

	// In Spectre index mode, apply an appropriate mask to avoid speculative out-of-bounds accesses.
	if base.Flag.Cfg.SpectreIndex {
		op := ssa.OpSpectreIndex
		if kind != ssa.BoundsIndex && kind != ssa.BoundsIndexU {
			op = ssa.OpSpectreSliceIndex
		}
		idx = s.newValue2(op, types.Types[types.TINT], idx, len)
	}

	return idx
}

// If cmp (a bool) is false, panic using the given function.
func (s *state) check(cmp *ssa.Value, fn *obj.LSym) {
	b := s.endBlock()
	b.Kind = ssa.BlockIf
	b.SetControl(cmp)
	b.Likely = ssa.BranchLikely
	bNext := s.f.NewBlock(ssa.BlockPlain)
	line := s.peekPos()
	pos := base.Ctxt.PosTable.Pos(line)
	fl := funcLine{f: fn, base: pos.Base(), line: pos.Line()}
	bPanic := s.panics[fl]
	if bPanic == nil {
		bPanic = s.f.NewBlock(ssa.BlockPlain)
		s.panics[fl] = bPanic
		s.startBlock(bPanic)
		// The panic call takes/returns memory to ensure that the right
		// memory state is observed if the panic happens.
		s.rtcall(fn, false, nil)
	}
	b.AddEdgeTo(bNext)
	b.AddEdgeTo(bPanic)
	s.startBlock(bNext)
}

func (s *state) intDivide(n ir.Node, a, b *ssa.Value) *ssa.Value {
	needcheck := true
	switch b.Op {
	case ssa.OpConst8, ssa.OpConst16, ssa.OpConst32, ssa.OpConst64:
		if b.AuxInt != 0 {
			needcheck = false
		}
	}
	if needcheck {
		// do a size-appropriate check for zero
		cmp := s.newValue2(s.ssaOp(ir.ONE, n.Type()), types.Types[types.TBOOL], b, s.zeroVal(n.Type()))
		s.check(cmp, ir.Syms.Panicdivide)
	}
	return s.newValue2(s.ssaOp(n.Op(), n.Type()), a.Type, a, b)
}

// rtcall issues a call to the given runtime function fn with the listed args.
// Returns a slice of results of the given result types.
// The call is added to the end of the current block.
// If returns is false, the block is marked as an exit block.
func (s *state) rtcall(fn *obj.LSym, returns bool, results []*types.Type, args ...*ssa.Value) []*ssa.Value {
	s.prevCall = nil
	// Write args to the stack
	off := base.Ctxt.FixedFrameSize()
	var callArgs []*ssa.Value
	var callArgTypes []*types.Type

	for _, arg := range args {
		t := arg.Type
		off = types.Rnd(off, t.Alignment())
		size := t.Size()
		callArgs = append(callArgs, arg)
		callArgTypes = append(callArgTypes, t)
		off += size
	}
	off = types.Rnd(off, int64(types.RegSize))

	// Accumulate results types and offsets
	offR := off
	for _, t := range results {
		offR = types.Rnd(offR, t.Alignment())
		offR += t.Size()
	}

	// Issue call
	var call *ssa.Value
	aux := ssa.StaticAuxCall(fn, s.f.ABIDefault.ABIAnalyzeTypes(nil, callArgTypes, results))
	callArgs = append(callArgs, s.mem())
	call = s.newValue0A(ssa.OpStaticLECall, aux.LateExpansionResultType(), aux)
	call.AddArgs(callArgs...)
	s.vars[memVar] = s.newValue1I(ssa.OpSelectN, types.TypeMem, int64(len(results)), call)

	if !returns {
		// Finish block
		b := s.endBlock()
		b.Kind = ssa.BlockExit
		b.SetControl(call)
		call.AuxInt = off - base.Ctxt.FixedFrameSize()
		if len(results) > 0 {
			s.Fatalf("panic call can't have results")
		}
		return nil
	}

	// Load results
	res := make([]*ssa.Value, len(results))
	for i, t := range results {
		off = types.Rnd(off, t.Alignment())
		res[i] = s.resultOfCall(call, int64(i), t)
		off += t.Size()
	}
	off = types.Rnd(off, int64(types.PtrSize))

	// Remember how much callee stack space we needed.
	call.AuxInt = off

	return res
}

// do *left = right for type t.
func (s *state) storeType(t *types.Type, left, right *ssa.Value, skip skipMask, leftIsStmt bool) {
	s.instrument(t, left, instrumentWrite)

	if skip == 0 && (!t.HasPointers() || ssa.IsStackAddr(left)) {
		// Known to not have write barrier. Store the whole type.
		s.vars[memVar] = s.newValue3Apos(ssa.OpStore, types.TypeMem, t, left, right, s.mem(), leftIsStmt)
		return
	}

	// store scalar fields first, so write barrier stores for
	// pointer fields can be grouped together, and scalar values
	// don't need to be live across the write barrier call.
	// TODO: if the writebarrier pass knows how to reorder stores,
	// we can do a single store here as long as skip==0.
	s.storeTypeScalars(t, left, right, skip)
	if skip&skipPtr == 0 && t.HasPointers() {
		s.storeTypePtrs(t, left, right)
	}
}

// do *left = right for all scalar (non-pointer) parts of t.
func (s *state) storeTypeScalars(t *types.Type, left, right *ssa.Value, skip skipMask) {
	switch {
	case t.IsBoolean() || t.IsInteger() || t.IsFloat() || t.IsComplex():
		s.store(t, left, right)
	case t.IsPtrShaped():
		if t.IsPtr() && t.Elem().NotInHeap() {
			s.store(t, left, right) // see issue 42032
		}
		// otherwise, no scalar fields.
	case t.IsString():
		if skip&skipLen != 0 {
			return
		}
		len := s.newValue1(ssa.OpStringLen, types.Types[types.TINT], right)
		lenAddr := s.newValue1I(ssa.OpOffPtr, s.f.Config.Types.IntPtr, s.config.PtrSize, left)
		s.store(types.Types[types.TINT], lenAddr, len)
	case t.IsSlice():
		if skip&skipLen == 0 {
			len := s.newValue1(ssa.OpSliceLen, types.Types[types.TINT], right)
			lenAddr := s.newValue1I(ssa.OpOffPtr, s.f.Config.Types.IntPtr, s.config.PtrSize, left)
			s.store(types.Types[types.TINT], lenAddr, len)
		}
		if skip&skipCap == 0 {
			cap := s.newValue1(ssa.OpSliceCap, types.Types[types.TINT], right)
			capAddr := s.newValue1I(ssa.OpOffPtr, s.f.Config.Types.IntPtr, 2*s.config.PtrSize, left)
			s.store(types.Types[types.TINT], capAddr, cap)
		}
	case t.IsInterface():
		// itab field doesn't need a write barrier (even though it is a pointer).
		itab := s.newValue1(ssa.OpITab, s.f.Config.Types.BytePtr, right)
		s.store(types.Types[types.TUINTPTR], left, itab)
	case t.IsStruct():
		n := t.NumFields()
		for i := 0; i < n; i++ {
			ft := t.FieldType(i)
			addr := s.newValue1I(ssa.OpOffPtr, ft.PtrTo(), t.FieldOff(i), left)
			val := s.newValue1I(ssa.OpStructSelect, ft, int64(i), right)
			s.storeTypeScalars(ft, addr, val, 0)
		}
	case t.IsArray() && t.NumElem() == 0:
		// nothing
	case t.IsArray() && t.NumElem() == 1:
		s.storeTypeScalars(t.Elem(), left, s.newValue1I(ssa.OpArraySelect, t.Elem(), 0, right), 0)
	default:
		s.Fatalf("bad write barrier type %v", t)
	}
}

// do *left = right for all pointer parts of t.
func (s *state) storeTypePtrs(t *types.Type, left, right *ssa.Value) {
	switch {
	case t.IsPtrShaped():
		if t.IsPtr() && t.Elem().NotInHeap() {
			break // see issue 42032
		}
		s.store(t, left, right)
	case t.IsString():
		ptr := s.newValue1(ssa.OpStringPtr, s.f.Config.Types.BytePtr, right)
		s.store(s.f.Config.Types.BytePtr, left, ptr)
	case t.IsSlice():
		elType := types.NewPtr(t.Elem())
		ptr := s.newValue1(ssa.OpSlicePtr, elType, right)
		s.store(elType, left, ptr)
	case t.IsInterface():
		// itab field is treated as a scalar.
		idata := s.newValue1(ssa.OpIData, s.f.Config.Types.BytePtr, right)
		idataAddr := s.newValue1I(ssa.OpOffPtr, s.f.Config.Types.BytePtrPtr, s.config.PtrSize, left)
		s.store(s.f.Config.Types.BytePtr, idataAddr, idata)
	case t.IsStruct():
		n := t.NumFields()
		for i := 0; i < n; i++ {
			ft := t.FieldType(i)
			if !ft.HasPointers() {
				continue
			}
			addr := s.newValue1I(ssa.OpOffPtr, ft.PtrTo(), t.FieldOff(i), left)
			val := s.newValue1I(ssa.OpStructSelect, ft, int64(i), right)
			s.storeTypePtrs(ft, addr, val)
		}
	case t.IsArray() && t.NumElem() == 0:
		// nothing
	case t.IsArray() && t.NumElem() == 1:
		s.storeTypePtrs(t.Elem(), left, s.newValue1I(ssa.OpArraySelect, t.Elem(), 0, right))
	default:
		s.Fatalf("bad write barrier type %v", t)
	}
}

// putArg evaluates n for the purpose of passing it as an argument to a function and returns the value for the call.
func (s *state) putArg(n ir.Node, t *types.Type) *ssa.Value {
	var a *ssa.Value
	if !TypeOK(t) {
		a = s.newValue2(ssa.OpDereference, t, s.addr(n), s.mem())
	} else {
		a = s.expr(n)
	}
	return a
}

func (s *state) storeArgWithBase(n ir.Node, t *types.Type, base *ssa.Value, off int64) {
	pt := types.NewPtr(t)
	var addr *ssa.Value
	if base == s.sp {
		// Use special routine that avoids allocation on duplicate offsets.
		addr = s.constOffPtrSP(pt, off)
	} else {
		addr = s.newValue1I(ssa.OpOffPtr, pt, off, base)
	}

	if !TypeOK(t) {
		a := s.addr(n)
		s.move(t, addr, a)
		return
	}

	a := s.expr(n)
	s.storeType(t, addr, a, 0, false)
}

// slice computes the slice v[i:j:k] and returns ptr, len, and cap of result.
// i,j,k may be nil, in which case they are set to their default value.
// v may be a slice, string or pointer to an array.
func (s *state) slice(v, i, j, k *ssa.Value, bounded bool) (p, l, c *ssa.Value) {
	t := v.Type
	var ptr, len, cap *ssa.Value
	switch {
	case t.IsSlice():
		ptr = s.newValue1(ssa.OpSlicePtr, types.NewPtr(t.Elem()), v)
		len = s.newValue1(ssa.OpSliceLen, types.Types[types.TINT], v)
		cap = s.newValue1(ssa.OpSliceCap, types.Types[types.TINT], v)
	case t.IsString():
		ptr = s.newValue1(ssa.OpStringPtr, types.NewPtr(types.Types[types.TUINT8]), v)
		len = s.newValue1(ssa.OpStringLen, types.Types[types.TINT], v)
		cap = len
	case t.IsPtr():
		if !t.Elem().IsArray() {
			s.Fatalf("bad ptr to array in slice %v\n", t)
		}
		s.nilCheck(v)
		ptr = s.newValue1(ssa.OpCopy, types.NewPtr(t.Elem().Elem()), v)
		len = s.constInt(types.Types[types.TINT], t.Elem().NumElem())
		cap = len
	default:
		s.Fatalf("bad type in slice %v\n", t)
	}

	// Set default values
	if i == nil {
		i = s.constInt(types.Types[types.TINT], 0)
	}
	if j == nil {
		j = len
	}
	three := true
	if k == nil {
		three = false
		k = cap
	}

	// Panic if slice indices are not in bounds.
	// Make sure we check these in reverse order so that we're always
	// comparing against a value known to be nonnegative. See issue 28797.
	if three {
		if k != cap {
			kind := ssa.BoundsSlice3Alen
			if t.IsSlice() {
				kind = ssa.BoundsSlice3Acap
			}
			k = s.boundsCheck(k, cap, kind, bounded)
		}
		if j != k {
			j = s.boundsCheck(j, k, ssa.BoundsSlice3B, bounded)
		}
		i = s.boundsCheck(i, j, ssa.BoundsSlice3C, bounded)
	} else {
		if j != k {
			kind := ssa.BoundsSliceAlen
			if t.IsSlice() {
				kind = ssa.BoundsSliceAcap
			}
			j = s.boundsCheck(j, k, kind, bounded)
		}
		i = s.boundsCheck(i, j, ssa.BoundsSliceB, bounded)
	}

	// Word-sized integer operations.
	subOp := s.ssaOp(ir.OSUB, types.Types[types.TINT])
	mulOp := s.ssaOp(ir.OMUL, types.Types[types.TINT])
	andOp := s.ssaOp(ir.OAND, types.Types[types.TINT])

	// Calculate the length (rlen) and capacity (rcap) of the new slice.
	// For strings the capacity of the result is unimportant. However,
	// we use rcap to test if we've generated a zero-length slice.
	// Use length of strings for that.
	rlen := s.newValue2(subOp, types.Types[types.TINT], j, i)
	rcap := rlen
	if j != k && !t.IsString() {
		rcap = s.newValue2(subOp, types.Types[types.TINT], k, i)
	}

	if (i.Op == ssa.OpConst64 || i.Op == ssa.OpConst32) && i.AuxInt == 0 {
		// No pointer arithmetic necessary.
		return ptr, rlen, rcap
	}

	// Calculate the base pointer (rptr) for the new slice.
	//
	// Generate the following code assuming that indexes are in bounds.
	// The masking is to make sure that we don't generate a slice
	// that points to the next object in memory. We cannot just set
	// the pointer to nil because then we would create a nil slice or
	// string.
	//
	//     rcap = k - i
	//     rlen = j - i
	//     rptr = ptr + (mask(rcap) & (i * stride))
	//
	// Where mask(x) is 0 if x==0 and -1 if x>0 and stride is the width
	// of the element type.
	stride := s.constInt(types.Types[types.TINT], ptr.Type.Elem().Width)

	// The delta is the number of bytes to offset ptr by.
	delta := s.newValue2(mulOp, types.Types[types.TINT], i, stride)

	// If we're slicing to the point where the capacity is zero,
	// zero out the delta.
	mask := s.newValue1(ssa.OpSlicemask, types.Types[types.TINT], rcap)
	delta = s.newValue2(andOp, types.Types[types.TINT], delta, mask)

	// Compute rptr = ptr + delta.
	rptr := s.newValue2(ssa.OpAddPtr, ptr.Type, ptr, delta)

	return rptr, rlen, rcap
}

type u642fcvtTab struct {
	leq, cvt2F, and, rsh, or, add ssa.Op
	one                           func(*state, *types.Type, int64) *ssa.Value
}

var u64_f64 = u642fcvtTab{
	leq:   ssa.OpLeq64,
	cvt2F: ssa.OpCvt64to64F,
	and:   ssa.OpAnd64,
	rsh:   ssa.OpRsh64Ux64,
	or:    ssa.OpOr64,
	add:   ssa.OpAdd64F,
	one:   (*state).constInt64,
}

var u64_f32 = u642fcvtTab{
	leq:   ssa.OpLeq64,
	cvt2F: ssa.OpCvt64to32F,
	and:   ssa.OpAnd64,
	rsh:   ssa.OpRsh64Ux64,
	or:    ssa.OpOr64,
	add:   ssa.OpAdd32F,
	one:   (*state).constInt64,
}

func (s *state) uint64Tofloat64(n ir.Node, x *ssa.Value, ft, tt *types.Type) *ssa.Value {
	return s.uint64Tofloat(&u64_f64, n, x, ft, tt)
}

func (s *state) uint64Tofloat32(n ir.Node, x *ssa.Value, ft, tt *types.Type) *ssa.Value {
	return s.uint64Tofloat(&u64_f32, n, x, ft, tt)
}

func (s *state) uint64Tofloat(cvttab *u642fcvtTab, n ir.Node, x *ssa.Value, ft, tt *types.Type) *ssa.Value {
	// if x >= 0 {
	//    result = (floatY) x
	// } else {
	// 	  y = uintX(x) ; y = x & 1
	// 	  z = uintX(x) ; z = z >> 1
	// 	  z = z >> 1
	// 	  z = z | y
	// 	  result = floatY(z)
	// 	  result = result + result
	// }
	//
	// Code borrowed from old code generator.
	// What's going on: large 64-bit "unsigned" looks like
	// negative number to hardware's integer-to-float
	// conversion. However, because the mantissa is only
	// 63 bits, we don't need the LSB, so instead we do an
	// unsigned right shift (divide by two), convert, and
	// double. However, before we do that, we need to be
	// sure that we do not lose a "1" if that made the
	// difference in the resulting rounding. Therefore, we
	// preserve it, and OR (not ADD) it back in. The case
	// that matters is when the eleven discarded bits are
	// equal to 10000000001; that rounds up, and the 1 cannot
	// be lost else it would round down if the LSB of the
	// candidate mantissa is 0.
	cmp := s.newValue2(cvttab.leq, types.Types[types.TBOOL], s.zeroVal(ft), x)
	b := s.endBlock()
	b.Kind = ssa.BlockIf
	b.SetControl(cmp)
	b.Likely = ssa.BranchLikely

	bThen := s.f.NewBlock(ssa.BlockPlain)
	bElse := s.f.NewBlock(ssa.BlockPlain)
	bAfter := s.f.NewBlock(ssa.BlockPlain)

	b.AddEdgeTo(bThen)
	s.startBlock(bThen)
	a0 := s.newValue1(cvttab.cvt2F, tt, x)
	s.vars[n] = a0
	s.endBlock()
	bThen.AddEdgeTo(bAfter)

	b.AddEdgeTo(bElse)
	s.startBlock(bElse)
	one := cvttab.one(s, ft, 1)
	y := s.newValue2(cvttab.and, ft, x, one)
	z := s.newValue2(cvttab.rsh, ft, x, one)
	z = s.newValue2(cvttab.or, ft, z, y)
	a := s.newValue1(cvttab.cvt2F, tt, z)
	a1 := s.newValue2(cvttab.add, tt, a, a)
	s.vars[n] = a1
	s.endBlock()
	bElse.AddEdgeTo(bAfter)

	s.startBlock(bAfter)
	return s.variable(n, n.Type())
}

type u322fcvtTab struct {
	cvtI2F, cvtF2F ssa.Op
}

var u32_f64 = u322fcvtTab{
	cvtI2F: ssa.OpCvt32to64F,
	cvtF2F: ssa.OpCopy,
}

var u32_f32 = u322fcvtTab{
	cvtI2F: ssa.OpCvt32to32F,
	cvtF2F: ssa.OpCvt64Fto32F,
}

func (s *state) uint32Tofloat64(n ir.Node, x *ssa.Value, ft, tt *types.Type) *ssa.Value {
	return s.uint32Tofloat(&u32_f64, n, x, ft, tt)
}

func (s *state) uint32Tofloat32(n ir.Node, x *ssa.Value, ft, tt *types.Type) *ssa.Value {
	return s.uint32Tofloat(&u32_f32, n, x, ft, tt)
}

func (s *state) uint32Tofloat(cvttab *u322fcvtTab, n ir.Node, x *ssa.Value, ft, tt *types.Type) *ssa.Value {
	// if x >= 0 {
	// 	result = floatY(x)
	// } else {
	// 	result = floatY(float64(x) + (1<<32))
	// }
	cmp := s.newValue2(ssa.OpLeq32, types.Types[types.TBOOL], s.zeroVal(ft), x)
	b := s.endBlock()
	b.Kind = ssa.BlockIf
	b.SetControl(cmp)
	b.Likely = ssa.BranchLikely

	bThen := s.f.NewBlock(ssa.BlockPlain)
	bElse := s.f.NewBlock(ssa.BlockPlain)
	bAfter := s.f.NewBlock(ssa.BlockPlain)

	b.AddEdgeTo(bThen)
	s.startBlock(bThen)
	a0 := s.newValue1(cvttab.cvtI2F, tt, x)
	s.vars[n] = a0
	s.endBlock()
	bThen.AddEdgeTo(bAfter)

	b.AddEdgeTo(bElse)
	s.startBlock(bElse)
	a1 := s.newValue1(ssa.OpCvt32to64F, types.Types[types.TFLOAT64], x)
	twoToThe32 := s.constFloat64(types.Types[types.TFLOAT64], float64(1<<32))
	a2 := s.newValue2(ssa.OpAdd64F, types.Types[types.TFLOAT64], a1, twoToThe32)
	a3 := s.newValue1(cvttab.cvtF2F, tt, a2)

	s.vars[n] = a3
	s.endBlock()
	bElse.AddEdgeTo(bAfter)

	s.startBlock(bAfter)
	return s.variable(n, n.Type())
}

// referenceTypeBuiltin generates code for the len/cap builtins for maps and channels.
func (s *state) referenceTypeBuiltin(n *ir.UnaryExpr, x *ssa.Value) *ssa.Value {
	if !n.X.Type().IsMap() && !n.X.Type().IsChan() {
		s.Fatalf("node must be a map or a channel")
	}
	// if n == nil {
	//   return 0
	// } else {
	//   // len
	//   return *((*int)n)
	//   // cap
	//   return *(((*int)n)+1)
	// }
	lenType := n.Type()
	nilValue := s.constNil(types.Types[types.TUINTPTR])
	cmp := s.newValue2(ssa.OpEqPtr, types.Types[types.TBOOL], x, nilValue)
	b := s.endBlock()
	b.Kind = ssa.BlockIf
	b.SetControl(cmp)
	b.Likely = ssa.BranchUnlikely

	bThen := s.f.NewBlock(ssa.BlockPlain)
	bElse := s.f.NewBlock(ssa.BlockPlain)
	bAfter := s.f.NewBlock(ssa.BlockPlain)

	// length/capacity of a nil map/chan is zero
	b.AddEdgeTo(bThen)
	s.startBlock(bThen)
	s.vars[n] = s.zeroVal(lenType)
	s.endBlock()
	bThen.AddEdgeTo(bAfter)

	b.AddEdgeTo(bElse)
	s.startBlock(bElse)
	switch n.Op() {
	case ir.OLEN:
		// length is stored in the first word for map/chan
		s.vars[n] = s.load(lenType, x)
	case ir.OCAP:
		// capacity is stored in the second word for chan
		sw := s.newValue1I(ssa.OpOffPtr, lenType.PtrTo(), lenType.Width, x)
		s.vars[n] = s.load(lenType, sw)
	default:
		s.Fatalf("op must be OLEN or OCAP")
	}
	s.endBlock()
	bElse.AddEdgeTo(bAfter)

	s.startBlock(bAfter)
	return s.variable(n, lenType)
}

type f2uCvtTab struct {
	ltf, cvt2U, subf, or ssa.Op
	floatValue           func(*state, *types.Type, float64) *ssa.Value
	intValue             func(*state, *types.Type, int64) *ssa.Value
	cutoff               uint64
}

var f32_u64 = f2uCvtTab{
	ltf:        ssa.OpLess32F,
	cvt2U:      ssa.OpCvt32Fto64,
	subf:       ssa.OpSub32F,
	or:         ssa.OpOr64,
	floatValue: (*state).constFloat32,
	intValue:   (*state).constInt64,
	cutoff:     1 << 63,
}

var f64_u64 = f2uCvtTab{
	ltf:        ssa.OpLess64F,
	cvt2U:      ssa.OpCvt64Fto64,
	subf:       ssa.OpSub64F,
	or:         ssa.OpOr64,
	floatValue: (*state).constFloat64,
	intValue:   (*state).constInt64,
	cutoff:     1 << 63,
}

var f32_u32 = f2uCvtTab{
	ltf:        ssa.OpLess32F,
	cvt2U:      ssa.OpCvt32Fto32,
	subf:       ssa.OpSub32F,
	or:         ssa.OpOr32,
	floatValue: (*state).constFloat32,
	intValue:   func(s *state, t *types.Type, v int64) *ssa.Value { return s.constInt32(t, int32(v)) },
	cutoff:     1 << 31,
}

var f64_u32 = f2uCvtTab{
	ltf:        ssa.OpLess64F,
	cvt2U:      ssa.OpCvt64Fto32,
	subf:       ssa.OpSub64F,
	or:         ssa.OpOr32,
	floatValue: (*state).constFloat64,
	intValue:   func(s *state, t *types.Type, v int64) *ssa.Value { return s.constInt32(t, int32(v)) },
	cutoff:     1 << 31,
}

func (s *state) float32ToUint64(n ir.Node, x *ssa.Value, ft, tt *types.Type) *ssa.Value {
	return s.floatToUint(&f32_u64, n, x, ft, tt)
}
func (s *state) float64ToUint64(n ir.Node, x *ssa.Value, ft, tt *types.Type) *ssa.Value {
	return s.floatToUint(&f64_u64, n, x, ft, tt)
}

func (s *state) float32ToUint32(n ir.Node, x *ssa.Value, ft, tt *types.Type) *ssa.Value {
	return s.floatToUint(&f32_u32, n, x, ft, tt)
}

func (s *state) float64ToUint32(n ir.Node, x *ssa.Value, ft, tt *types.Type) *ssa.Value {
	return s.floatToUint(&f64_u32, n, x, ft, tt)
}

func (s *state) floatToUint(cvttab *f2uCvtTab, n ir.Node, x *ssa.Value, ft, tt *types.Type) *ssa.Value {
	// cutoff:=1<<(intY_Size-1)
	// if x < floatX(cutoff) {
	// 	result = uintY(x)
	// } else {
	// 	y = x - floatX(cutoff)
	// 	z = uintY(y)
	// 	result = z | -(cutoff)
	// }
	cutoff := cvttab.floatValue(s, ft, float64(cvttab.cutoff))
	cmp := s.newValue2(cvttab.ltf, types.Types[types.TBOOL], x, cutoff)
	b := s.endBlock()
	b.Kind = ssa.BlockIf
	b.SetControl(cmp)
	b.Likely = ssa.BranchLikely

	bThen := s.f.NewBlock(ssa.BlockPlain)
	bElse := s.f.NewBlock(ssa.BlockPlain)
	bAfter := s.f.NewBlock(ssa.BlockPlain)

	b.AddEdgeTo(bThen)
	s.startBlock(bThen)
	a0 := s.newValue1(cvttab.cvt2U, tt, x)
	s.vars[n] = a0
	s.endBlock()
	bThen.AddEdgeTo(bAfter)

	b.AddEdgeTo(bElse)
	s.startBlock(bElse)
	y := s.newValue2(cvttab.subf, ft, x, cutoff)
	y = s.newValue1(cvttab.cvt2U, tt, y)
	z := cvttab.intValue(s, tt, int64(-cvttab.cutoff))
	a1 := s.newValue2(cvttab.or, tt, y, z)
	s.vars[n] = a1
	s.endBlock()
	bElse.AddEdgeTo(bAfter)

	s.startBlock(bAfter)
	return s.variable(n, n.Type())
}

// dottype generates SSA for a type assertion node.
// commaok indicates whether to panic or return a bool.
// If commaok is false, resok will be nil.
func (s *state) dottype(n *ir.TypeAssertExpr, commaok bool) (res, resok *ssa.Value) {
	iface := s.expr(n.X)              // input interface
	target := s.reflectType(n.Type()) // target type
	byteptr := s.f.Config.Types.BytePtr

	if n.Type().IsInterface() {
		if n.Type().IsEmptyInterface() {
			// Converting to an empty interface.
			// Input could be an empty or nonempty interface.
			if base.Debug.TypeAssert > 0 {
				base.WarnfAt(n.Pos(), "type assertion inlined")
			}

			// Get itab/type field from input.
			itab := s.newValue1(ssa.OpITab, byteptr, iface)
			// Conversion succeeds iff that field is not nil.
			cond := s.newValue2(ssa.OpNeqPtr, types.Types[types.TBOOL], itab, s.constNil(byteptr))

			if n.X.Type().IsEmptyInterface() && commaok {
				// Converting empty interface to empty interface with ,ok is just a nil check.
				return iface, cond
			}

			// Branch on nilness.
			b := s.endBlock()
			b.Kind = ssa.BlockIf
			b.SetControl(cond)
			b.Likely = ssa.BranchLikely
			bOk := s.f.NewBlock(ssa.BlockPlain)
			bFail := s.f.NewBlock(ssa.BlockPlain)
			b.AddEdgeTo(bOk)
			b.AddEdgeTo(bFail)

			if !commaok {
				// On failure, panic by calling panicnildottype.
				s.startBlock(bFail)
				s.rtcall(ir.Syms.Panicnildottype, false, nil, target)

				// On success, return (perhaps modified) input interface.
				s.startBlock(bOk)
				if n.X.Type().IsEmptyInterface() {
					res = iface // Use input interface unchanged.
					return
				}
				// Load type out of itab, build interface with existing idata.
				off := s.newValue1I(ssa.OpOffPtr, byteptr, int64(types.PtrSize), itab)
				typ := s.load(byteptr, off)
				idata := s.newValue1(ssa.OpIData, byteptr, iface)
				res = s.newValue2(ssa.OpIMake, n.Type(), typ, idata)
				return
			}

			s.startBlock(bOk)
			// nonempty -> empty
			// Need to load type from itab
			off := s.newValue1I(ssa.OpOffPtr, byteptr, int64(types.PtrSize), itab)
			s.vars[typVar] = s.load(byteptr, off)
			s.endBlock()

			// itab is nil, might as well use that as the nil result.
			s.startBlock(bFail)
			s.vars[typVar] = itab
			s.endBlock()

			// Merge point.
			bEnd := s.f.NewBlock(ssa.BlockPlain)
			bOk.AddEdgeTo(bEnd)
			bFail.AddEdgeTo(bEnd)
			s.startBlock(bEnd)
			idata := s.newValue1(ssa.OpIData, byteptr, iface)
			res = s.newValue2(ssa.OpIMake, n.Type(), s.variable(typVar, byteptr), idata)
			resok = cond
			delete(s.vars, typVar)
			return
		}
		// converting to a nonempty interface needs a runtime call.
		if base.Debug.TypeAssert > 0 {
			base.WarnfAt(n.Pos(), "type assertion not inlined")
		}
		if !commaok {
			fn := ir.Syms.AssertI2I
			if n.X.Type().IsEmptyInterface() {
				fn = ir.Syms.AssertE2I
			}
			data := s.newValue1(ssa.OpIData, types.Types[types.TUNSAFEPTR], iface)
			tab := s.newValue1(ssa.OpITab, byteptr, iface)
			tab = s.rtcall(fn, true, []*types.Type{byteptr}, target, tab)[0]
			return s.newValue2(ssa.OpIMake, n.Type(), tab, data), nil
		}
		fn := ir.Syms.AssertI2I2
		if n.X.Type().IsEmptyInterface() {
			fn = ir.Syms.AssertE2I2
		}
		res = s.rtcall(fn, true, []*types.Type{n.Type()}, target, iface)[0]
		resok = s.newValue2(ssa.OpNeqInter, types.Types[types.TBOOL], res, s.constInterface(n.Type()))
		return
	}

	if base.Debug.TypeAssert > 0 {
		base.WarnfAt(n.Pos(), "type assertion inlined")
	}

	// Converting to a concrete type.
	direct := types.IsDirectIface(n.Type())
	itab := s.newValue1(ssa.OpITab, byteptr, iface) // type word of interface
	if base.Debug.TypeAssert > 0 {
		base.WarnfAt(n.Pos(), "type assertion inlined")
	}
	var targetITab *ssa.Value
	if n.X.Type().IsEmptyInterface() {
		// Looking for pointer to target type.
		targetITab = target
	} else {
		// Looking for pointer to itab for target type and source interface.
		targetITab = s.expr(n.Itab)
	}

	var tmp ir.Node     // temporary for use with large types
	var addr *ssa.Value // address of tmp
	if commaok && !TypeOK(n.Type()) {
		// unSSAable type, use temporary.
		// TODO: get rid of some of these temporaries.
		tmp, addr = s.temp(n.Pos(), n.Type())
	}

	cond := s.newValue2(ssa.OpEqPtr, types.Types[types.TBOOL], itab, targetITab)
	b := s.endBlock()
	b.Kind = ssa.BlockIf
	b.SetControl(cond)
	b.Likely = ssa.BranchLikely

	bOk := s.f.NewBlock(ssa.BlockPlain)
	bFail := s.f.NewBlock(ssa.BlockPlain)
	b.AddEdgeTo(bOk)
	b.AddEdgeTo(bFail)

	if !commaok {
		// on failure, panic by calling panicdottype
		s.startBlock(bFail)
		taddr := s.reflectType(n.X.Type())
		if n.X.Type().IsEmptyInterface() {
			s.rtcall(ir.Syms.PanicdottypeE, false, nil, itab, target, taddr)
		} else {
			s.rtcall(ir.Syms.PanicdottypeI, false, nil, itab, target, taddr)
		}

		// on success, return data from interface
		s.startBlock(bOk)
		if direct {
			return s.newValue1(ssa.OpIData, n.Type(), iface), nil
		}
		p := s.newValue1(ssa.OpIData, types.NewPtr(n.Type()), iface)
		return s.load(n.Type(), p), nil
	}

	// commaok is the more complicated case because we have
	// a control flow merge point.
	bEnd := s.f.NewBlock(ssa.BlockPlain)
	// Note that we need a new valVar each time (unlike okVar where we can
	// reuse the variable) because it might have a different type every time.
	valVar := ssaMarker("val")

	// type assertion succeeded
	s.startBlock(bOk)
	if tmp == nil {
		if direct {
			s.vars[valVar] = s.newValue1(ssa.OpIData, n.Type(), iface)
		} else {
			p := s.newValue1(ssa.OpIData, types.NewPtr(n.Type()), iface)
			s.vars[valVar] = s.load(n.Type(), p)
		}
	} else {
		p := s.newValue1(ssa.OpIData, types.NewPtr(n.Type()), iface)
		s.move(n.Type(), addr, p)
	}
	s.vars[okVar] = s.constBool(true)
	s.endBlock()
	bOk.AddEdgeTo(bEnd)

	// type assertion failed
	s.startBlock(bFail)
	if tmp == nil {
		s.vars[valVar] = s.zeroVal(n.Type())
	} else {
		s.zero(n.Type(), addr)
	}
	s.vars[okVar] = s.constBool(false)
	s.endBlock()
	bFail.AddEdgeTo(bEnd)

	// merge point
	s.startBlock(bEnd)
	if tmp == nil {
		res = s.variable(valVar, n.Type())
		delete(s.vars, valVar)
	} else {
		res = s.load(n.Type(), addr)
		s.vars[memVar] = s.newValue1A(ssa.OpVarKill, types.TypeMem, tmp.(*ir.Name), s.mem())
	}
	resok = s.variable(okVar, types.Types[types.TBOOL])
	delete(s.vars, okVar)
	return res, resok
}

// temp allocates a temp of type t at position pos
func (s *state) temp(pos src.XPos, t *types.Type) (*ir.Name, *ssa.Value) {
	tmp := typecheck.TempAt(pos, s.curfn, t)
	s.vars[memVar] = s.newValue1A(ssa.OpVarDef, types.TypeMem, tmp, s.mem())
	addr := s.addr(tmp)
	return tmp, addr
}

// variable returns the value of a variable at the current location.
func (s *state) variable(n ir.Node, t *types.Type) *ssa.Value {
	v := s.vars[n]
	if v != nil {
		return v
	}
	v = s.fwdVars[n]
	if v != nil {
		return v
	}

	if s.curBlock == s.f.Entry {
		// No variable should be live at entry.
		s.Fatalf("Value live at entry. It shouldn't be. func %s, node %v, value %v", s.f.Name, n, v)
	}
	// Make a FwdRef, which records a value that's live on block input.
	// We'll find the matching definition as part of insertPhis.
	v = s.newValue0A(ssa.OpFwdRef, t, fwdRefAux{N: n})
	s.fwdVars[n] = v
	if n.Op() == ir.ONAME {
		s.addNamedValue(n.(*ir.Name), v)
	}
	return v
}

func (s *state) mem() *ssa.Value {
	return s.variable(memVar, types.TypeMem)
}

func (s *state) addNamedValue(n *ir.Name, v *ssa.Value) {
	if n.Class == ir.Pxxx {
		// Don't track our marker nodes (memVar etc.).
		return
	}
	if ir.IsAutoTmp(n) {
		// Don't track temporary variables.
		return
	}
	if n.Class == ir.PPARAMOUT {
		// Don't track named output values.  This prevents return values
		// from being assigned too early. See #14591 and #14762. TODO: allow this.
		return
	}
	loc := ssa.LocalSlot{N: n, Type: n.Type(), Off: 0}
	values, ok := s.f.NamedValues[loc]
	if !ok {
		s.f.Names = append(s.f.Names, &loc)
		s.f.CanonicalLocalSlots[loc] = &loc
	}
	s.f.NamedValues[loc] = append(values, v)
}

// Branch is an unresolved branch.
type Branch struct {
	P *obj.Prog  // branch instruction
	B *ssa.Block // target
}

// State contains state needed during Prog generation.
type State struct {
	ABI obj.ABI

	pp *objw.Progs

	// Branches remembers all the branch instructions we've seen
	// and where they would like to go.
	Branches []Branch

	// bstart remembers where each block starts (indexed by block ID)
	bstart []*obj.Prog

	maxarg int64 // largest frame size for arguments to calls made by the function

	// Map from GC safe points to liveness index, generated by
	// liveness analysis.
	livenessMap liveness.Map

	// partLiveArgs includes arguments that may be partially live, for which we
	// need to generate instructions that spill the argument registers.
	partLiveArgs map[*ir.Name]bool

	// lineRunStart records the beginning of the current run of instructions
	// within a single block sharing the same line number
	// Used to move statement marks to the beginning of such runs.
	lineRunStart *obj.Prog

	// wasm: The number of values on the WebAssembly stack. This is only used as a safeguard.
	OnWasmStackSkipped int
}

func (s *State) FuncInfo() *obj.FuncInfo {
	return s.pp.CurFunc.LSym.Func()
}

// Prog appends a new Prog.
func (s *State) Prog(as obj.As) *obj.Prog {
	p := s.pp.Prog(as)
	if objw.LosesStmtMark(as) {
		return p
	}
	// Float a statement start to the beginning of any same-line run.
	// lineRunStart is reset at block boundaries, which appears to work well.
	if s.lineRunStart == nil || s.lineRunStart.Pos.Line() != p.Pos.Line() {
		s.lineRunStart = p
	} else if p.Pos.IsStmt() == src.PosIsStmt {
		s.lineRunStart.Pos = s.lineRunStart.Pos.WithIsStmt()
		p.Pos = p.Pos.WithNotStmt()
	}
	return p
}

// Pc returns the current Prog.
func (s *State) Pc() *obj.Prog {
	return s.pp.Next
}

// SetPos sets the current source position.
func (s *State) SetPos(pos src.XPos) {
	s.pp.Pos = pos
}

// Br emits a single branch instruction and returns the instruction.
// Not all architectures need the returned instruction, but otherwise
// the boilerplate is common to all.
func (s *State) Br(op obj.As, target *ssa.Block) *obj.Prog {
	p := s.Prog(op)
	p.To.Type = obj.TYPE_BRANCH
	s.Branches = append(s.Branches, Branch{P: p, B: target})
	return p
}

// DebugFriendlySetPosFrom adjusts Pos.IsStmt subject to heuristics
// that reduce "jumpy" line number churn when debugging.
// Spill/fill/copy instructions from the register allocator,
// phi functions, and instructions with a no-pos position
// are examples of instructions that can cause churn.
func (s *State) DebugFriendlySetPosFrom(v *ssa.Value) {
	switch v.Op {
	case ssa.OpPhi, ssa.OpCopy, ssa.OpLoadReg, ssa.OpStoreReg:
		// These are not statements
		s.SetPos(v.Pos.WithNotStmt())
	default:
		p := v.Pos
		if p != src.NoXPos {
			// If the position is defined, update the position.
			// Also convert default IsStmt to NotStmt; only
			// explicit statement boundaries should appear
			// in the generated code.
			if p.IsStmt() != src.PosIsStmt {
				p = p.WithNotStmt()
				// Calls use the pos attached to v, but copy the statement mark from State
			}
			s.SetPos(p)
		} else {
			s.SetPos(s.pp.Pos.WithNotStmt())
		}
	}
}

// emit argument info (locations on stack) for traceback.
func emitArgInfo(e *ssafn, f *ssa.Func, pp *objw.Progs) {
	ft := e.curfn.Type()
	if ft.NumRecvs() == 0 && ft.NumParams() == 0 {
		return
	}

	x := EmitArgInfo(e.curfn, f.OwnAux.ABIInfo())
	e.curfn.LSym.Func().ArgInfo = x

	// Emit a funcdata pointing at the arg info data.
	p := pp.Prog(obj.AFUNCDATA)
	p.From.SetConst(objabi.FUNCDATA_ArgInfo)
	p.To.Type = obj.TYPE_MEM
	p.To.Name = obj.NAME_EXTERN
	p.To.Sym = x
}

// emit argument info (locations on stack) of f for traceback.
func EmitArgInfo(f *ir.Func, abiInfo *abi.ABIParamResultInfo) *obj.LSym {
	x := base.Ctxt.Lookup(fmt.Sprintf("%s.arginfo%d", f.LSym.Name, f.ABI))

	PtrSize := int64(types.PtrSize)
	uintptrTyp := types.Types[types.TUINTPTR]

	isAggregate := func(t *types.Type) bool {
		return t.IsStruct() || t.IsArray() || t.IsComplex() || t.IsInterface() || t.IsString() || t.IsSlice()
	}

	// Populate the data.
	// The data is a stream of bytes, which contains the offsets and sizes of the
	// non-aggregate arguments or non-aggregate fields/elements of aggregate-typed
	// arguments, along with special "operators". Specifically,
	// - for each non-aggrgate arg/field/element, its offset from FP (1 byte) and
	//   size (1 byte)
	// - special operators:
	//   - 0xff - end of sequence
	//   - 0xfe - print { (at the start of an aggregate-typed argument)
	//   - 0xfd - print } (at the end of an aggregate-typed argument)
	//   - 0xfc - print ... (more args/fields/elements)
	//   - 0xfb - print _ (offset too large)
	// These constants need to be in sync with runtime.traceback.go:printArgs.
	const (
		_endSeq         = 0xff
		_startAgg       = 0xfe
		_endAgg         = 0xfd
		_dotdotdot      = 0xfc
		_offsetTooLarge = 0xfb
		_special        = 0xf0 // above this are operators, below this are ordinary offsets
	)

	const (
		limit    = 10 // print no more than 10 args/components
		maxDepth = 5  // no more than 5 layers of nesting

		// maxLen is a (conservative) upper bound of the byte stream length. For
		// each arg/component, it has no more than 2 bytes of data (size, offset),
		// and no more than one {, }, ... at each level (it cannot have both the
		// data and ... unless it is the last one, just be conservative). Plus 1
		// for _endSeq.
		maxLen = (maxDepth*3+2)*limit + 1
	)

	wOff := 0
	n := 0
	writebyte := func(o uint8) { wOff = objw.Uint8(x, wOff, o) }

	// Write one non-aggrgate arg/field/element.
	write1 := func(sz, offset int64) {
		if offset >= _special {
			writebyte(_offsetTooLarge)
		} else {
			writebyte(uint8(offset))
			writebyte(uint8(sz))
		}
		n++
	}

	// Visit t recursively and write it out.
	// Returns whether to continue visiting.
	var visitType func(baseOffset int64, t *types.Type, depth int) bool
	visitType = func(baseOffset int64, t *types.Type, depth int) bool {
		if n >= limit {
			writebyte(_dotdotdot)
			return false
		}
		if !isAggregate(t) {
			write1(t.Size(), baseOffset)
			return true
		}
		writebyte(_startAgg)
		depth++
		if depth >= maxDepth {
			writebyte(_dotdotdot)
			writebyte(_endAgg)
			n++
			return true
		}
		switch {
		case t.IsInterface(), t.IsString():
			_ = visitType(baseOffset, uintptrTyp, depth) &&
				visitType(baseOffset+PtrSize, uintptrTyp, depth)
		case t.IsSlice():
			_ = visitType(baseOffset, uintptrTyp, depth) &&
				visitType(baseOffset+PtrSize, uintptrTyp, depth) &&
				visitType(baseOffset+PtrSize*2, uintptrTyp, depth)
		case t.IsComplex():
			_ = visitType(baseOffset, types.FloatForComplex(t), depth) &&
				visitType(baseOffset+t.Size()/2, types.FloatForComplex(t), depth)
		case t.IsArray():
			if t.NumElem() == 0 {
				n++ // {} counts as a component
				break
			}
			for i := int64(0); i < t.NumElem(); i++ {
				if !visitType(baseOffset, t.Elem(), depth) {
					break
				}
				baseOffset += t.Elem().Size()
			}
		case t.IsStruct():
			if t.NumFields() == 0 {
				n++ // {} counts as a component
				break
			}
			for _, field := range t.Fields().Slice() {
				if !visitType(baseOffset+field.Offset, field.Type, depth) {
					break
				}
			}
		}
		writebyte(_endAgg)
		return true
	}

	for _, a := range abiInfo.InParams() {
		if !visitType(a.FrameOffset(abiInfo), a.Type, 0) {
			break
		}
	}
	writebyte(_endSeq)
	if wOff > maxLen {
		base.Fatalf("ArgInfo too large")
	}

	return x
}

// genssa appends entries to pp for each instruction in f.
func genssa(f *ssa.Func, pp *objw.Progs) {
	var s State
	s.ABI = f.OwnAux.Fn.ABI()

	e := f.Frontend().(*ssafn)

	s.livenessMap, s.partLiveArgs = liveness.Compute(e.curfn, f, e.stkptrsize, pp)
	emitArgInfo(e, f, pp)

	openDeferInfo := e.curfn.LSym.Func().OpenCodedDeferInfo
	if openDeferInfo != nil {
		// This function uses open-coded defers -- write out the funcdata
		// info that we computed at the end of genssa.
		p := pp.Prog(obj.AFUNCDATA)
		p.From.SetConst(objabi.FUNCDATA_OpenCodedDeferInfo)
		p.To.Type = obj.TYPE_MEM
		p.To.Name = obj.NAME_EXTERN
		p.To.Sym = openDeferInfo
	}

	// Remember where each block starts.
	s.bstart = make([]*obj.Prog, f.NumBlocks())
	s.pp = pp
	var progToValue map[*obj.Prog]*ssa.Value
	var progToBlock map[*obj.Prog]*ssa.Block
	var valueToProgAfter []*obj.Prog // The first Prog following computation of a value v; v is visible at this point.
	if f.PrintOrHtmlSSA {
		progToValue = make(map[*obj.Prog]*ssa.Value, f.NumValues())
		progToBlock = make(map[*obj.Prog]*ssa.Block, f.NumBlocks())
		f.Logf("genssa %s\n", f.Name)
		progToBlock[s.pp.Next] = f.Blocks[0]
	}

	if base.Ctxt.Flag_locationlists {
		if cap(f.Cache.ValueToProgAfter) < f.NumValues() {
			f.Cache.ValueToProgAfter = make([]*obj.Prog, f.NumValues())
		}
		valueToProgAfter = f.Cache.ValueToProgAfter[:f.NumValues()]
		for i := range valueToProgAfter {
			valueToProgAfter[i] = nil
		}
	}

	// If the very first instruction is not tagged as a statement,
	// debuggers may attribute it to previous function in program.
	firstPos := src.NoXPos
	for _, v := range f.Entry.Values {
		if v.Pos.IsStmt() == src.PosIsStmt {
			firstPos = v.Pos
			v.Pos = firstPos.WithDefaultStmt()
			break
		}
	}

	// inlMarks has an entry for each Prog that implements an inline mark.
	// It maps from that Prog to the global inlining id of the inlined body
	// which should unwind to this Prog's location.
	var inlMarks map[*obj.Prog]int32
	var inlMarkList []*obj.Prog

	// inlMarksByPos maps from a (column 1) source position to the set of
	// Progs that are in the set above and have that source position.
	var inlMarksByPos map[src.XPos][]*obj.Prog

	// Emit basic blocks
	for i, b := range f.Blocks {
		s.bstart[b.ID] = s.pp.Next
		s.lineRunStart = nil

		// Attach a "default" liveness info. Normally this will be
		// overwritten in the Values loop below for each Value. But
		// for an empty block this will be used for its control
		// instruction. We won't use the actual liveness map on a
		// control instruction. Just mark it something that is
		// preemptible, unless this function is "all unsafe".
		s.pp.NextLive = objw.LivenessIndex{StackMapIndex: -1, IsUnsafePoint: liveness.IsUnsafe(f)}

		// Emit values in block
		Arch.SSAMarkMoves(&s, b)
		for _, v := range b.Values {
			x := s.pp.Next
			s.DebugFriendlySetPosFrom(v)

			if v.Op.ResultInArg0() && v.ResultReg() != v.Args[0].Reg() {
				v.Fatalf("input[0] and output not in same register %s", v.LongString())
			}

			switch v.Op {
			case ssa.OpInitMem:
				// memory arg needs no code
			case ssa.OpArg:
				// input args need no code
			case ssa.OpSP, ssa.OpSB:
				// nothing to do
			case ssa.OpSelect0, ssa.OpSelect1, ssa.OpSelectN, ssa.OpMakeResult:
				// nothing to do
			case ssa.OpGetG:
				// nothing to do when there's a g register,
				// and checkLower complains if there's not
			case ssa.OpVarDef, ssa.OpVarLive, ssa.OpKeepAlive, ssa.OpVarKill:
				// nothing to do; already used by liveness
			case ssa.OpPhi:
				CheckLoweredPhi(v)
			case ssa.OpConvert:
				// nothing to do; no-op conversion for liveness
				if v.Args[0].Reg() != v.Reg() {
					v.Fatalf("OpConvert should be a no-op: %s; %s", v.Args[0].LongString(), v.LongString())
				}
			case ssa.OpInlMark:
				p := Arch.Ginsnop(s.pp)
				if inlMarks == nil {
					inlMarks = map[*obj.Prog]int32{}
					inlMarksByPos = map[src.XPos][]*obj.Prog{}
				}
				inlMarks[p] = v.AuxInt32()
				inlMarkList = append(inlMarkList, p)
				pos := v.Pos.AtColumn1()
				inlMarksByPos[pos] = append(inlMarksByPos[pos], p)

			default:
				// Special case for first line in function; move it to the start (which cannot be a register-valued instruction)
				if firstPos != src.NoXPos && v.Op != ssa.OpArgIntReg && v.Op != ssa.OpArgFloatReg && v.Op != ssa.OpLoadReg && v.Op != ssa.OpStoreReg {
					s.SetPos(firstPos)
					firstPos = src.NoXPos
				}
				// Attach this safe point to the next
				// instruction.
				s.pp.NextLive = s.livenessMap.Get(v)

				// let the backend handle it
				Arch.SSAGenValue(&s, v)
			}

			if base.Ctxt.Flag_locationlists {
				valueToProgAfter[v.ID] = s.pp.Next
			}

			if f.PrintOrHtmlSSA {
				for ; x != s.pp.Next; x = x.Link {
					progToValue[x] = v
				}
			}
		}
		// If this is an empty infinite loop, stick a hardware NOP in there so that debuggers are less confused.
		if s.bstart[b.ID] == s.pp.Next && len(b.Succs) == 1 && b.Succs[0].Block() == b {
			p := Arch.Ginsnop(s.pp)
			p.Pos = p.Pos.WithIsStmt()
			if b.Pos == src.NoXPos {
				b.Pos = p.Pos // It needs a file, otherwise a no-file non-zero line causes confusion.  See #35652.
				if b.Pos == src.NoXPos {
					b.Pos = pp.Text.Pos // Sometimes p.Pos is empty.  See #35695.
				}
			}
			b.Pos = b.Pos.WithBogusLine() // Debuggers are not good about infinite loops, force a change in line number
		}
		// Emit control flow instructions for block
		var next *ssa.Block
		if i < len(f.Blocks)-1 && base.Flag.N == 0 {
			// If -N, leave next==nil so every block with successors
			// ends in a JMP (except call blocks - plive doesn't like
			// select{send,recv} followed by a JMP call).  Helps keep
			// line numbers for otherwise empty blocks.
			next = f.Blocks[i+1]
		}
		x := s.pp.Next
		s.SetPos(b.Pos)
		Arch.SSAGenBlock(&s, b, next)
		if f.PrintOrHtmlSSA {
			for ; x != s.pp.Next; x = x.Link {
				progToBlock[x] = b
			}
		}
	}
	if f.Blocks[len(f.Blocks)-1].Kind == ssa.BlockExit {
		// We need the return address of a panic call to
		// still be inside the function in question. So if
		// it ends in a call which doesn't return, add a
		// nop (which will never execute) after the call.
		Arch.Ginsnop(pp)
	}
	if openDeferInfo != nil {
		// When doing open-coded defers, generate a disconnected call to
		// deferreturn and a return. This will be used to during panic
		// recovery to unwind the stack and return back to the runtime.
		s.pp.NextLive = s.livenessMap.DeferReturn
		p := pp.Prog(obj.ACALL)
		p.To.Type = obj.TYPE_MEM
		p.To.Name = obj.NAME_EXTERN
		p.To.Sym = ir.Syms.Deferreturn

		// Load results into registers. So when a deferred function
		// recovers a panic, it will return to caller with right results.
		// The results are already in memory, because they are not SSA'd
		// when the function has defers (see canSSAName).
		if f.OwnAux.ABIInfo().OutRegistersUsed() != 0 {
			Arch.LoadRegResults(&s, f)
		}

		pp.Prog(obj.ARET)
	}

	if inlMarks != nil {
		// We have some inline marks. Try to find other instructions we're
		// going to emit anyway, and use those instructions instead of the
		// inline marks.
		for p := pp.Text; p != nil; p = p.Link {
			if p.As == obj.ANOP || p.As == obj.AFUNCDATA || p.As == obj.APCDATA || p.As == obj.ATEXT || p.As == obj.APCALIGN || Arch.LinkArch.Family == sys.Wasm {
				// Don't use 0-sized instructions as inline marks, because we need
				// to identify inline mark instructions by pc offset.
				// (Some of these instructions are sometimes zero-sized, sometimes not.
				// We must not use anything that even might be zero-sized.)
				// TODO: are there others?
				continue
			}
			if _, ok := inlMarks[p]; ok {
				// Don't use inline marks themselves. We don't know
				// whether they will be zero-sized or not yet.
				continue
			}
			pos := p.Pos.AtColumn1()
			s := inlMarksByPos[pos]
			if len(s) == 0 {
				continue
			}
			for _, m := range s {
				// We found an instruction with the same source position as
				// some of the inline marks.
				// Use this instruction instead.
				p.Pos = p.Pos.WithIsStmt() // promote position to a statement
				pp.CurFunc.LSym.Func().AddInlMark(p, inlMarks[m])
				// Make the inline mark a real nop, so it doesn't generate any code.
				m.As = obj.ANOP
				m.Pos = src.NoXPos
				m.From = obj.Addr{}
				m.To = obj.Addr{}
			}
			delete(inlMarksByPos, pos)
		}
		// Any unmatched inline marks now need to be added to the inlining tree (and will generate a nop instruction).
		for _, p := range inlMarkList {
			if p.As != obj.ANOP {
				pp.CurFunc.LSym.Func().AddInlMark(p, inlMarks[p])
			}
		}
	}

	if base.Ctxt.Flag_locationlists {
		var debugInfo *ssa.FuncDebug
		if e.curfn.ABI == obj.ABIInternal && base.Flag.N != 0 {
			debugInfo = ssa.BuildFuncDebugNoOptimized(base.Ctxt, f, base.Debug.LocationLists > 1, StackOffset)
		} else {
			debugInfo = ssa.BuildFuncDebug(base.Ctxt, f, base.Debug.LocationLists > 1, StackOffset)
		}
		e.curfn.DebugInfo = debugInfo
		bstart := s.bstart
		idToIdx := make([]int, f.NumBlocks())
		for i, b := range f.Blocks {
			idToIdx[b.ID] = i
		}
		// Note that at this moment, Prog.Pc is a sequence number; it's
		// not a real PC until after assembly, so this mapping has to
		// be done later.
		debugInfo.GetPC = func(b, v ssa.ID) int64 {
			switch v {
			case ssa.BlockStart.ID:
				if b == f.Entry.ID {
					return 0 // Start at the very beginning, at the assembler-generated prologue.
					// this should only happen for function args (ssa.OpArg)
				}
				return bstart[b].Pc
			case ssa.BlockEnd.ID:
				blk := f.Blocks[idToIdx[b]]
				nv := len(blk.Values)
				return valueToProgAfter[blk.Values[nv-1].ID].Pc
			case ssa.FuncEnd.ID:
				return e.curfn.LSym.Size
			default:
				return valueToProgAfter[v].Pc
			}
		}
	}

	// Resolve branches, and relax DefaultStmt into NotStmt
	for _, br := range s.Branches {
		br.P.To.SetTarget(s.bstart[br.B.ID])
		if br.P.Pos.IsStmt() != src.PosIsStmt {
			br.P.Pos = br.P.Pos.WithNotStmt()
		} else if v0 := br.B.FirstPossibleStmtValue(); v0 != nil && v0.Pos.Line() == br.P.Pos.Line() && v0.Pos.IsStmt() == src.PosIsStmt {
			br.P.Pos = br.P.Pos.WithNotStmt()
		}

	}

	if e.log { // spew to stdout
		filename := ""
		for p := pp.Text; p != nil; p = p.Link {
			if p.Pos.IsKnown() && p.InnermostFilename() != filename {
				filename = p.InnermostFilename()
				f.Logf("# %s\n", filename)
			}

			var s string
			if v, ok := progToValue[p]; ok {
				s = v.String()
			} else if b, ok := progToBlock[p]; ok {
				s = b.String()
			} else {
				s = "   " // most value and branch strings are 2-3 characters long
			}
			f.Logf(" %-6s\t%.5d (%s)\t%s\n", s, p.Pc, p.InnermostLineNumber(), p.InstructionString())
		}
	}
	if f.HTMLWriter != nil { // spew to ssa.html
		var buf bytes.Buffer
		buf.WriteString("<code>")
		buf.WriteString("<dl class=\"ssa-gen\">")
		filename := ""
		for p := pp.Text; p != nil; p = p.Link {
			// Don't spam every line with the file name, which is often huge.
			// Only print changes, and "unknown" is not a change.
			if p.Pos.IsKnown() && p.InnermostFilename() != filename {
				filename = p.InnermostFilename()
				buf.WriteString("<dt class=\"ssa-prog-src\"></dt><dd class=\"ssa-prog\">")
				buf.WriteString(html.EscapeString("# " + filename))
				buf.WriteString("</dd>")
			}

			buf.WriteString("<dt class=\"ssa-prog-src\">")
			if v, ok := progToValue[p]; ok {
				buf.WriteString(v.HTML())
			} else if b, ok := progToBlock[p]; ok {
				buf.WriteString("<b>" + b.HTML() + "</b>")
			}
			buf.WriteString("</dt>")
			buf.WriteString("<dd class=\"ssa-prog\">")
			buf.WriteString(fmt.Sprintf("%.5d <span class=\"l%v line-number\">(%s)</span> %s", p.Pc, p.InnermostLineNumber(), p.InnermostLineNumberHTML(), html.EscapeString(p.InstructionString())))
			buf.WriteString("</dd>")
		}
		buf.WriteString("</dl>")
		buf.WriteString("</code>")
		f.HTMLWriter.WriteColumn("genssa", "genssa", "ssa-prog", buf.String())
	}

	defframe(&s, e, f)

	f.HTMLWriter.Close()
	f.HTMLWriter = nil
}

func defframe(s *State, e *ssafn, f *ssa.Func) {
	pp := s.pp

	frame := types.Rnd(s.maxarg+e.stksize, int64(types.RegSize))
	if Arch.PadFrame != nil {
		frame = Arch.PadFrame(frame)
	}

	// Fill in argument and frame size.
	pp.Text.To.Type = obj.TYPE_TEXTSIZE
	pp.Text.To.Val = int32(types.Rnd(f.OwnAux.ArgWidth(), int64(types.RegSize)))
	pp.Text.To.Offset = frame

	p := pp.Text

	// Insert code to spill argument registers if the named slot may be partially
	// live. That is, the named slot is considered live by liveness analysis,
	// (because a part of it is live), but we may not spill all parts into the
	// slot. This can only happen with aggregate-typed arguments that are SSA-able
	// and not address-taken (for non-SSA-able or address-taken arguments we always
	// spill upfront).
	// Note: spilling is unnecessary in the -N/no-optimize case, since all values
	// will be considered non-SSAable and spilled up front.
	// TODO(register args) Make liveness more fine-grained to that partial spilling is okay.
	if f.OwnAux.ABIInfo().InRegistersUsed() != 0 && base.Flag.N == 0 {
		// First, see if it is already spilled before it may be live. Look for a spill
		// in the entry block up to the first safepoint.
		type nameOff struct {
			n   *ir.Name
			off int64
		}
		partLiveArgsSpilled := make(map[nameOff]bool)
		for _, v := range f.Entry.Values {
			if v.Op.IsCall() {
				break
			}
			if v.Op != ssa.OpStoreReg || v.Args[0].Op != ssa.OpArgIntReg {
				continue
			}
			n, off := ssa.AutoVar(v)
			if n.Class != ir.PPARAM || n.Addrtaken() || !TypeOK(n.Type()) || !s.partLiveArgs[n] {
				continue
			}
			partLiveArgsSpilled[nameOff{n, off}] = true
		}

		// Then, insert code to spill registers if not already.
		for _, a := range f.OwnAux.ABIInfo().InParams() {
			n, ok := a.Name.(*ir.Name)
			if !ok || n.Addrtaken() || !TypeOK(n.Type()) || !s.partLiveArgs[n] || len(a.Registers) <= 1 {
				continue
			}
			rts, offs := a.RegisterTypesAndOffsets()
			for i := range a.Registers {
				if !rts[i].HasPointers() {
					continue
				}
				if partLiveArgsSpilled[nameOff{n, offs[i]}] {
					continue // already spilled
				}
				reg := ssa.ObjRegForAbiReg(a.Registers[i], f.Config)
				p = Arch.SpillArgReg(pp, p, f, rts[i], reg, n, offs[i])
			}
		}
	}

	// Insert code to zero ambiguously live variables so that the
	// garbage collector only sees initialized values when it
	// looks for pointers.
	var lo, hi int64

	// Opaque state for backend to use. Current backends use it to
	// keep track of which helper registers have been zeroed.
	var state uint32

	// Iterate through declarations. Autos are sorted in decreasing
	// frame offset order.
	for _, n := range e.curfn.Dcl {
		if !n.Needzero() {
			continue
		}
		if n.Class != ir.PAUTO {
			e.Fatalf(n.Pos(), "needzero class %d", n.Class)
		}
		if n.Type().Size()%int64(types.PtrSize) != 0 || n.FrameOffset()%int64(types.PtrSize) != 0 || n.Type().Size() == 0 {
			e.Fatalf(n.Pos(), "var %L has size %d offset %d", n, n.Type().Size(), n.Offset_)
		}

		if lo != hi && n.FrameOffset()+n.Type().Size() >= lo-int64(2*types.RegSize) {
			// Merge with range we already have.
			lo = n.FrameOffset()
			continue
		}

		// Zero old range
		p = Arch.ZeroRange(pp, p, frame+lo, hi-lo, &state)

		// Set new range.
		lo = n.FrameOffset()
		hi = lo + n.Type().Size()
	}

	// Zero final range.
	Arch.ZeroRange(pp, p, frame+lo, hi-lo, &state)
}

// For generating consecutive jump instructions to model a specific branching
type IndexJump struct {
	Jump  obj.As
	Index int
}

func (s *State) oneJump(b *ssa.Block, jump *IndexJump) {
	p := s.Br(jump.Jump, b.Succs[jump.Index].Block())
	p.Pos = b.Pos
}

// CombJump generates combinational instructions (2 at present) for a block jump,
// thereby the behaviour of non-standard condition codes could be simulated
func (s *State) CombJump(b, next *ssa.Block, jumps *[2][2]IndexJump) {
	switch next {
	case b.Succs[0].Block():
		s.oneJump(b, &jumps[0][0])
		s.oneJump(b, &jumps[0][1])
	case b.Succs[1].Block():
		s.oneJump(b, &jumps[1][0])
		s.oneJump(b, &jumps[1][1])
	default:
		var q *obj.Prog
		if b.Likely != ssa.BranchUnlikely {
			s.oneJump(b, &jumps[1][0])
			s.oneJump(b, &jumps[1][1])
			q = s.Br(obj.AJMP, b.Succs[1].Block())
		} else {
			s.oneJump(b, &jumps[0][0])
			s.oneJump(b, &jumps[0][1])
			q = s.Br(obj.AJMP, b.Succs[0].Block())
		}
		q.Pos = b.Pos
	}
}

// AddAux adds the offset in the aux fields (AuxInt and Aux) of v to a.
func AddAux(a *obj.Addr, v *ssa.Value) {
	AddAux2(a, v, v.AuxInt)
}
func AddAux2(a *obj.Addr, v *ssa.Value, offset int64) {
	if a.Type != obj.TYPE_MEM && a.Type != obj.TYPE_ADDR {
		v.Fatalf("bad AddAux addr %v", a)
	}
	// add integer offset
	a.Offset += offset

	// If no additional symbol offset, we're done.
	if v.Aux == nil {
		return
	}
	// Add symbol's offset from its base register.
	switch n := v.Aux.(type) {
	case *ssa.AuxCall:
		a.Name = obj.NAME_EXTERN
		a.Sym = n.Fn
	case *obj.LSym:
		a.Name = obj.NAME_EXTERN
		a.Sym = n
	case *ir.Name:
		if n.Class == ir.PPARAM || (n.Class == ir.PPARAMOUT && !n.IsOutputParamInRegisters()) {
			a.Name = obj.NAME_PARAM
			a.Sym = ir.Orig(n).(*ir.Name).Linksym()
			a.Offset += n.FrameOffset()
			break
		}
		a.Name = obj.NAME_AUTO
		if n.Class == ir.PPARAMOUT {
			a.Sym = ir.Orig(n).(*ir.Name).Linksym()
		} else {
			a.Sym = n.Linksym()
		}
		a.Offset += n.FrameOffset()
	default:
		v.Fatalf("aux in %s not implemented %#v", v, v.Aux)
	}
}

// extendIndex extends v to a full int width.
// panic with the given kind if v does not fit in an int (only on 32-bit archs).
func (s *state) extendIndex(idx, len *ssa.Value, kind ssa.BoundsKind, bounded bool) *ssa.Value {
	size := idx.Type.Size()
	if size == s.config.PtrSize {
		return idx
	}
	if size > s.config.PtrSize {
		// truncate 64-bit indexes on 32-bit pointer archs. Test the
		// high word and branch to out-of-bounds failure if it is not 0.
		var lo *ssa.Value
		if idx.Type.IsSigned() {
			lo = s.newValue1(ssa.OpInt64Lo, types.Types[types.TINT], idx)
		} else {
			lo = s.newValue1(ssa.OpInt64Lo, types.Types[types.TUINT], idx)
		}
		if bounded || base.Flag.B != 0 {
			return lo
		}
		bNext := s.f.NewBlock(ssa.BlockPlain)
		bPanic := s.f.NewBlock(ssa.BlockExit)
		hi := s.newValue1(ssa.OpInt64Hi, types.Types[types.TUINT32], idx)
		cmp := s.newValue2(ssa.OpEq32, types.Types[types.TBOOL], hi, s.constInt32(types.Types[types.TUINT32], 0))
		if !idx.Type.IsSigned() {
			switch kind {
			case ssa.BoundsIndex:
				kind = ssa.BoundsIndexU
			case ssa.BoundsSliceAlen:
				kind = ssa.BoundsSliceAlenU
			case ssa.BoundsSliceAcap:
				kind = ssa.BoundsSliceAcapU
			case ssa.BoundsSliceB:
				kind = ssa.BoundsSliceBU
			case ssa.BoundsSlice3Alen:
				kind = ssa.BoundsSlice3AlenU
			case ssa.BoundsSlice3Acap:
				kind = ssa.BoundsSlice3AcapU
			case ssa.BoundsSlice3B:
				kind = ssa.BoundsSlice3BU
			case ssa.BoundsSlice3C:
				kind = ssa.BoundsSlice3CU
			}
		}
		b := s.endBlock()
		b.Kind = ssa.BlockIf
		b.SetControl(cmp)
		b.Likely = ssa.BranchLikely
		b.AddEdgeTo(bNext)
		b.AddEdgeTo(bPanic)

		s.startBlock(bPanic)
		mem := s.newValue4I(ssa.OpPanicExtend, types.TypeMem, int64(kind), hi, lo, len, s.mem())
		s.endBlock().SetControl(mem)
		s.startBlock(bNext)

		return lo
	}

	// Extend value to the required size
	var op ssa.Op
	if idx.Type.IsSigned() {
		switch 10*size + s.config.PtrSize {
		case 14:
			op = ssa.OpSignExt8to32
		case 18:
			op = ssa.OpSignExt8to64
		case 24:
			op = ssa.OpSignExt16to32
		case 28:
			op = ssa.OpSignExt16to64
		case 48:
			op = ssa.OpSignExt32to64
		default:
			s.Fatalf("bad signed index extension %s", idx.Type)
		}
	} else {
		switch 10*size + s.config.PtrSize {
		case 14:
			op = ssa.OpZeroExt8to32
		case 18:
			op = ssa.OpZeroExt8to64
		case 24:
			op = ssa.OpZeroExt16to32
		case 28:
			op = ssa.OpZeroExt16to64
		case 48:
			op = ssa.OpZeroExt32to64
		default:
			s.Fatalf("bad unsigned index extension %s", idx.Type)
		}
	}
	return s.newValue1(op, types.Types[types.TINT], idx)
}

// CheckLoweredPhi checks that regalloc and stackalloc correctly handled phi values.
// Called during ssaGenValue.
func CheckLoweredPhi(v *ssa.Value) {
	if v.Op != ssa.OpPhi {
		v.Fatalf("CheckLoweredPhi called with non-phi value: %v", v.LongString())
	}
	if v.Type.IsMemory() {
		return
	}
	f := v.Block.Func
	loc := f.RegAlloc[v.ID]
	for _, a := range v.Args {
		if aloc := f.RegAlloc[a.ID]; aloc != loc { // TODO: .Equal() instead?
			v.Fatalf("phi arg at different location than phi: %v @ %s, but arg %v @ %s\n%s\n", v, loc, a, aloc, v.Block.Func)
		}
	}
}

// CheckLoweredGetClosurePtr checks that v is the first instruction in the function's entry block,
// except for incoming in-register arguments.
// The output of LoweredGetClosurePtr is generally hardwired to the correct register.
// That register contains the closure pointer on closure entry.
func CheckLoweredGetClosurePtr(v *ssa.Value) {
	entry := v.Block.Func.Entry
	if entry != v.Block {
		base.Fatalf("in %s, badly placed LoweredGetClosurePtr: %v %v", v.Block.Func.Name, v.Block, v)
	}
	for _, w := range entry.Values {
		if w == v {
			break
		}
		switch w.Op {
		case ssa.OpArgIntReg, ssa.OpArgFloatReg:
			// okay
		default:
			base.Fatalf("in %s, badly placed LoweredGetClosurePtr: %v %v", v.Block.Func.Name, v.Block, v)
		}
	}
}

// CheckArgReg ensures that v is in the function's entry block.
func CheckArgReg(v *ssa.Value) {
	entry := v.Block.Func.Entry
	if entry != v.Block {
		base.Fatalf("in %s, badly placed ArgIReg or ArgFReg: %v %v", v.Block.Func.Name, v.Block, v)
	}
}

func AddrAuto(a *obj.Addr, v *ssa.Value) {
	n, off := ssa.AutoVar(v)
	a.Type = obj.TYPE_MEM
	a.Sym = n.Linksym()
	a.Reg = int16(Arch.REGSP)
	a.Offset = n.FrameOffset() + off
	if n.Class == ir.PPARAM || (n.Class == ir.PPARAMOUT && !n.IsOutputParamInRegisters()) {
		a.Name = obj.NAME_PARAM
	} else {
		a.Name = obj.NAME_AUTO
	}
}

// Call returns a new CALL instruction for the SSA value v.
// It uses PrepareCall to prepare the call.
func (s *State) Call(v *ssa.Value) *obj.Prog {
	pPosIsStmt := s.pp.Pos.IsStmt() // The statement-ness fo the call comes from ssaGenState
	s.PrepareCall(v)

	p := s.Prog(obj.ACALL)
	if pPosIsStmt == src.PosIsStmt {
		p.Pos = v.Pos.WithIsStmt()
	} else {
		p.Pos = v.Pos.WithNotStmt()
	}
	if sym, ok := v.Aux.(*ssa.AuxCall); ok && sym.Fn != nil {
		p.To.Type = obj.TYPE_MEM
		p.To.Name = obj.NAME_EXTERN
		p.To.Sym = sym.Fn
	} else {
		// TODO(mdempsky): Can these differences be eliminated?
		switch Arch.LinkArch.Family {
		case sys.AMD64, sys.I386, sys.PPC64, sys.RISCV64, sys.S390X, sys.Wasm:
			p.To.Type = obj.TYPE_REG
		case sys.ARM, sys.ARM64, sys.MIPS, sys.MIPS64:
			p.To.Type = obj.TYPE_MEM
		default:
			base.Fatalf("unknown indirect call family")
		}
		p.To.Reg = v.Args[0].Reg()
	}
	return p
}

// PrepareCall prepares to emit a CALL instruction for v and does call-related bookkeeping.
// It must be called immediately before emitting the actual CALL instruction,
// since it emits PCDATA for the stack map at the call (calls are safe points).
func (s *State) PrepareCall(v *ssa.Value) {
	idx := s.livenessMap.Get(v)
	if !idx.StackMapValid() {
		// See Liveness.hasStackMap.
		if sym, ok := v.Aux.(*ssa.AuxCall); !ok || !(sym.Fn == ir.Syms.Typedmemclr || sym.Fn == ir.Syms.Typedmemmove) {
			base.Fatalf("missing stack map index for %v", v.LongString())
		}
	}

	call, ok := v.Aux.(*ssa.AuxCall)

	if ok && call.Fn == ir.Syms.Deferreturn {
		// Deferred calls will appear to be returning to
		// the CALL deferreturn(SB) that we are about to emit.
		// However, the stack trace code will show the line
		// of the instruction byte before the return PC.
		// To avoid that being an unrelated instruction,
		// insert an actual hardware NOP that will have the right line number.
		// This is different from obj.ANOP, which is a virtual no-op
		// that doesn't make it into the instruction stream.
		Arch.Ginsnopdefer(s.pp)
	}

	if ok {
		// Record call graph information for nowritebarrierrec
		// analysis.
		if nowritebarrierrecCheck != nil {
			nowritebarrierrecCheck.recordCall(s.pp.CurFunc, call.Fn, v.Pos)
		}
	}

	if s.maxarg < v.AuxInt {
		s.maxarg = v.AuxInt
	}
}

// UseArgs records the fact that an instruction needs a certain amount of
// callee args space for its use.
func (s *State) UseArgs(n int64) {
	if s.maxarg < n {
		s.maxarg = n
	}
}

// fieldIdx finds the index of the field referred to by the ODOT node n.
func fieldIdx(n *ir.SelectorExpr) int {
	t := n.X.Type()
	if !t.IsStruct() {
		panic("ODOT's LHS is not a struct")
	}

	for i, f := range t.Fields().Slice() {
		if f.Sym == n.Sel {
			if f.Offset != n.Offset() {
				panic("field offset doesn't match")
			}
			return i
		}
	}
	panic(fmt.Sprintf("can't find field in expr %v\n", n))

	// TODO: keep the result of this function somewhere in the ODOT Node
	// so we don't have to recompute it each time we need it.
}

// ssafn holds frontend information about a function that the backend is processing.
// It also exports a bunch of compiler services for the ssa backend.
type ssafn struct {
	curfn      *ir.Func
	strings    map[string]*obj.LSym // map from constant string to data symbols
	stksize    int64                // stack size for current frame
	stkptrsize int64                // prefix of stack containing pointers
	log        bool                 // print ssa debug to the stdout
}

// StringData returns a symbol which
// is the data component of a global string constant containing s.
func (e *ssafn) StringData(s string) *obj.LSym {
	if aux, ok := e.strings[s]; ok {
		return aux
	}
	if e.strings == nil {
		e.strings = make(map[string]*obj.LSym)
	}
	data := staticdata.StringSym(e.curfn.Pos(), s)
	e.strings[s] = data
	return data
}

func (e *ssafn) Auto(pos src.XPos, t *types.Type) *ir.Name {
	return typecheck.TempAt(pos, e.curfn, t) // Note: adds new auto to e.curfn.Func.Dcl list
}

func (e *ssafn) DerefItab(it *obj.LSym, offset int64) *obj.LSym {
	return reflectdata.ITabSym(it, offset)
}

// SplitSlot returns a slot representing the data of parent starting at offset.
func (e *ssafn) SplitSlot(parent *ssa.LocalSlot, suffix string, offset int64, t *types.Type) ssa.LocalSlot {
	node := parent.N

	if node.Class != ir.PAUTO || node.Addrtaken() {
		// addressed things and non-autos retain their parents (i.e., cannot truly be split)
		return ssa.LocalSlot{N: node, Type: t, Off: parent.Off + offset}
	}

	s := &types.Sym{Name: node.Sym().Name + suffix, Pkg: types.LocalPkg}
	n := ir.NewNameAt(parent.N.Pos(), s)
	s.Def = n
	ir.AsNode(s.Def).Name().SetUsed(true)
	n.SetType(t)
	n.Class = ir.PAUTO
	n.SetEsc(ir.EscNever)
	n.Curfn = e.curfn
	e.curfn.Dcl = append(e.curfn.Dcl, n)
	types.CalcSize(t)
	return ssa.LocalSlot{N: n, Type: t, Off: 0, SplitOf: parent, SplitOffset: offset}
}

func (e *ssafn) CanSSA(t *types.Type) bool {
	return TypeOK(t)
}

func (e *ssafn) Line(pos src.XPos) string {
	return base.FmtPos(pos)
}

// Log logs a message from the compiler.
func (e *ssafn) Logf(msg string, args ...interface{}) {
	if e.log {
		fmt.Printf(msg, args...)
	}
}

func (e *ssafn) Log() bool {
	return e.log
}

// Fatal reports a compiler error and exits.
func (e *ssafn) Fatalf(pos src.XPos, msg string, args ...interface{}) {
	base.Pos = pos
	nargs := append([]interface{}{ir.FuncName(e.curfn)}, args...)
	base.Fatalf("'%s': "+msg, nargs...)
}

// Warnl reports a "warning", which is usually flag-triggered
// logging output for the benefit of tests.
func (e *ssafn) Warnl(pos src.XPos, fmt_ string, args ...interface{}) {
	base.WarnfAt(pos, fmt_, args...)
}

func (e *ssafn) Debug_checknil() bool {
	return base.Debug.Nil != 0
}

func (e *ssafn) UseWriteBarrier() bool {
	return base.Flag.WB
}

func (e *ssafn) Syslook(name string) *obj.LSym {
	switch name {
	case "goschedguarded":
		return ir.Syms.Goschedguarded
	case "writeBarrier":
		return ir.Syms.WriteBarrier
	case "gcWriteBarrier":
		return ir.Syms.GCWriteBarrier
	case "typedmemmove":
		return ir.Syms.Typedmemmove
	case "typedmemclr":
		return ir.Syms.Typedmemclr
	}
	e.Fatalf(src.NoXPos, "unknown Syslook func %v", name)
	return nil
}

func (e *ssafn) SetWBPos(pos src.XPos) {
	e.curfn.SetWBPos(pos)
}

func (e *ssafn) MyImportPath() string {
	return base.Ctxt.Pkgpath
}

func clobberBase(n ir.Node) ir.Node {
	if n.Op() == ir.ODOT {
		n := n.(*ir.SelectorExpr)
		if n.X.Type().NumFields() == 1 {
			return clobberBase(n.X)
		}
	}
	if n.Op() == ir.OINDEX {
		n := n.(*ir.IndexExpr)
		if n.X.Type().IsArray() && n.X.Type().NumElem() == 1 {
			return clobberBase(n.X)
		}
	}
	return n
}

// callTargetLSym returns the correct LSym to call 'callee' using its ABI.
func callTargetLSym(callee *ir.Name) *obj.LSym {
	if callee.Func == nil {
		// TODO(austin): This happens in a few cases of
		// compiler-generated functions. These are all
		// ABIInternal. It would be better if callee.Func was
		// never nil and we didn't need this case.
		return callee.Linksym()
	}

	return callee.LinksymABI(callee.Func.ABI)
}

func min8(a, b int8) int8 {
	if a < b {
		return a
	}
	return b
}

func max8(a, b int8) int8 {
	if a > b {
		return a
	}
	return b
}

// deferstruct makes a runtime._defer structure, with additional space for
// stksize bytes of args.
func deferstruct(stksize int64) *types.Type {
	makefield := func(name string, typ *types.Type) *types.Field {
		// Unlike the global makefield function, this one needs to set Pkg
		// because these types might be compared (in SSA CSE sorting).
		// TODO: unify this makefield and the global one above.
		sym := &types.Sym{Name: name, Pkg: types.LocalPkg}
		return types.NewField(src.NoXPos, sym, typ)
	}
	argtype := types.NewArray(types.Types[types.TUINT8], stksize)
	argtype.Width = stksize
	argtype.Align = 1
	// These fields must match the ones in runtime/runtime2.go:_defer and
	// cmd/compile/internal/gc/ssa.go:(*state).call.
	fields := []*types.Field{
		makefield("siz", types.Types[types.TUINT32]),
		makefield("started", types.Types[types.TBOOL]),
		makefield("heap", types.Types[types.TBOOL]),
		makefield("openDefer", types.Types[types.TBOOL]),
		makefield("sp", types.Types[types.TUINTPTR]),
		makefield("pc", types.Types[types.TUINTPTR]),
		// Note: the types here don't really matter. Defer structures
		// are always scanned explicitly during stack copying and GC,
		// so we make them uintptr type even though they are real pointers.
		makefield("fn", types.Types[types.TUINTPTR]),
		makefield("_panic", types.Types[types.TUINTPTR]),
		makefield("link", types.Types[types.TUINTPTR]),
		makefield("framepc", types.Types[types.TUINTPTR]),
		makefield("varp", types.Types[types.TUINTPTR]),
		makefield("fd", types.Types[types.TUINTPTR]),
		makefield("args", argtype),
	}

	// build struct holding the above fields
	s := types.NewStruct(types.NoPkg, fields)
	s.SetNoalg(true)
	types.CalcStructSize(s)
	return s
}

// SlotAddr uses LocalSlot information to initialize an obj.Addr
// The resulting addr is used in a non-standard context -- in the prologue
// of a function, before the frame has been constructed, so the standard
// addressing for the parameters will be wrong.
func SpillSlotAddr(spill ssa.Spill, baseReg int16, extraOffset int64) obj.Addr {
	return obj.Addr{
		Name:   obj.NAME_NONE,
		Type:   obj.TYPE_MEM,
		Reg:    baseReg,
		Offset: spill.Offset + extraOffset,
	}
}

var (
	BoundsCheckFunc [ssa.BoundsKindCount]*obj.LSym
	ExtendCheckFunc [ssa.BoundsKindCount]*obj.LSym
)

// GCWriteBarrierReg maps from registers to gcWriteBarrier implementation LSyms.
var GCWriteBarrierReg map[int16]*obj.LSym
