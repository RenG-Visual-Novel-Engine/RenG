// Inferno utils/8l/asm.c
// https://bitbucket.org/inferno-os/inferno-os/src/master/utils/8l/asm.c
//
//	Copyright © 1994-1999 Lucent Technologies Inc.  All rights reserved.
//	Portions Copyright © 1995-1997 C H Forsyth (forsyth@terzarima.net)
//	Portions Copyright © 1997-1999 Vita Nuova Limited
//	Portions Copyright © 2000-2007 Vita Nuova Holdings Limited (www.vitanuova.com)
//	Portions Copyright © 2004,2006 Bruce Ellis
//	Portions Copyright © 2005-2007 C H Forsyth (forsyth@terzarima.net)
//	Revisions Copyright © 2000-2007 Lucent Technologies Inc. and others
//	Portions Copyright © 2009 The Go Authors. All rights reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.  IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package ld

import (
	"bytes"
	"cmd/internal/bio"
	"cmd/internal/goobj"
	"cmd/internal/obj"
	"cmd/internal/objabi"
	"cmd/internal/sys"
	"cmd/link/internal/loadelf"
	"cmd/link/internal/loader"
	"cmd/link/internal/loadmacho"
	"cmd/link/internal/loadpe"
	"cmd/link/internal/loadxcoff"
	"cmd/link/internal/sym"
	"crypto/sha1"
	"debug/elf"
	"debug/macho"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"internal/buildcfg"
	exec "internal/execabs"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// Data layout and relocation.

// Derived from Inferno utils/6l/l.h
// https://bitbucket.org/inferno-os/inferno-os/src/master/utils/6l/l.h
//
//	Copyright © 1994-1999 Lucent Technologies Inc.  All rights reserved.
//	Portions Copyright © 1995-1997 C H Forsyth (forsyth@terzarima.net)
//	Portions Copyright © 1997-1999 Vita Nuova Limited
//	Portions Copyright © 2000-2007 Vita Nuova Holdings Limited (www.vitanuova.com)
//	Portions Copyright © 2004,2006 Bruce Ellis
//	Portions Copyright © 2005-2007 C H Forsyth (forsyth@terzarima.net)
//	Revisions Copyright © 2000-2007 Lucent Technologies Inc. and others
//	Portions Copyright © 2009 The Go Authors. All rights reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.  IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

// ArchSyms holds a number of architecture specific symbols used during
// relocation.  Rather than allowing them universal access to all symbols,
// we keep a subset for relocation application.
type ArchSyms struct {
	Rel     loader.Sym
	Rela    loader.Sym
	RelPLT  loader.Sym
	RelaPLT loader.Sym

	LinkEditGOT loader.Sym
	LinkEditPLT loader.Sym

	TOC    loader.Sym
	DotTOC []loader.Sym // for each version

	GOT    loader.Sym
	PLT    loader.Sym
	GOTPLT loader.Sym

	Tlsg      loader.Sym
	Tlsoffset int

	Dynamic loader.Sym
	DynSym  loader.Sym
	DynStr  loader.Sym

	unreachableMethod loader.Sym
}

// mkArchSym is a helper for setArchSyms, to set up a special symbol.
func (ctxt *Link) mkArchSym(name string, ver int, ls *loader.Sym) {
	*ls = ctxt.loader.LookupOrCreateSym(name, ver)
	ctxt.loader.SetAttrReachable(*ls, true)
}

// mkArchVecSym is similar to  setArchSyms, but operates on elements within
// a slice, where each element corresponds to some symbol version.
func (ctxt *Link) mkArchSymVec(name string, ver int, ls []loader.Sym) {
	ls[ver] = ctxt.loader.LookupOrCreateSym(name, ver)
	ctxt.loader.SetAttrReachable(ls[ver], true)
}

// setArchSyms sets up the ArchSyms structure, and must be called before
// relocations are applied.
func (ctxt *Link) setArchSyms() {
	ctxt.mkArchSym(".got", 0, &ctxt.GOT)
	ctxt.mkArchSym(".plt", 0, &ctxt.PLT)
	ctxt.mkArchSym(".got.plt", 0, &ctxt.GOTPLT)
	ctxt.mkArchSym(".dynamic", 0, &ctxt.Dynamic)
	ctxt.mkArchSym(".dynsym", 0, &ctxt.DynSym)
	ctxt.mkArchSym(".dynstr", 0, &ctxt.DynStr)
	ctxt.mkArchSym("runtime.unreachableMethod", sym.SymVerABIInternal, &ctxt.unreachableMethod)

	if ctxt.IsPPC64() {
		ctxt.mkArchSym("TOC", 0, &ctxt.TOC)

		// NB: note the +2 below for DotTOC2 compared to the +1 for
		// DocTOC. This is because loadlibfull() creates an additional
		// syms version during conversion of loader.Sym symbols to
		// *sym.Symbol symbols. Symbols that are assigned this final
		// version are not going to have TOC references, so it should
		// be ok for them to inherit an invalid .TOC. symbol.
		// TODO: revisit the +2, now that loadlibfull is gone.
		ctxt.DotTOC = make([]loader.Sym, ctxt.MaxVersion()+2)
		for i := 0; i <= ctxt.MaxVersion(); i++ {
			if i >= 2 && i < sym.SymVerStatic { // these versions are not used currently
				continue
			}
			ctxt.mkArchSymVec(".TOC.", i, ctxt.DotTOC)
		}
	}
	if ctxt.IsElf() {
		ctxt.mkArchSym(".rel", 0, &ctxt.Rel)
		ctxt.mkArchSym(".rela", 0, &ctxt.Rela)
		ctxt.mkArchSym(".rel.plt", 0, &ctxt.RelPLT)
		ctxt.mkArchSym(".rela.plt", 0, &ctxt.RelaPLT)
	}
	if ctxt.IsDarwin() {
		ctxt.mkArchSym(".linkedit.got", 0, &ctxt.LinkEditGOT)
		ctxt.mkArchSym(".linkedit.plt", 0, &ctxt.LinkEditPLT)
	}
}

type Arch struct {
	Funcalign  int
	Maxalign   int
	Minalign   int
	Dwarfregsp int
	Dwarfreglr int

	// Threshold of total text size, used for trampoline insertion. If the total
	// text size is smaller than TrampLimit, we won't need to insert trampolines.
	// It is pretty close to the offset range of a direct CALL machine instruction.
	// We leave some room for extra stuff like PLT stubs.
	TrampLimit uint64

	Androiddynld   string
	Linuxdynld     string
	Freebsddynld   string
	Netbsddynld    string
	Openbsddynld   string
	Dragonflydynld string
	Solarisdynld   string

	// Empty spaces between codeblocks will be padded with this value.
	// For example an architecture might want to pad with a trap instruction to
	// catch wayward programs. Architectures that do not define a padding value
	// are padded with zeros.
	CodePad []byte

	// Plan 9 variables.
	Plan9Magic  uint32
	Plan9_64Bit bool

	Adddynrel func(*Target, *loader.Loader, *ArchSyms, loader.Sym, loader.Reloc, int) bool
	Archinit  func(*Link)
	// Archreloc is an arch-specific hook that assists in relocation processing
	// (invoked by 'relocsym'); it handles target-specific relocation tasks.
	// Here "rel" is the current relocation being examined, "sym" is the symbol
	// containing the chunk of data to which the relocation applies, and "off"
	// is the contents of the to-be-relocated data item (from sym.P). Return
	// value is the appropriately relocated value (to be written back to the
	// same spot in sym.P), number of external _host_ relocations needed (i.e.
	// ELF/Mach-O/etc. relocations, not Go relocations, this must match Elfreloc1,
	// etc.), and a boolean indicating success/failure (a failing value indicates
	// a fatal error).
	Archreloc func(*Target, *loader.Loader, *ArchSyms, loader.Reloc, loader.Sym,
		int64) (relocatedOffset int64, nExtReloc int, ok bool)
	// Archrelocvariant is a second arch-specific hook used for
	// relocation processing; it handles relocations where r.Type is
	// insufficient to describe the relocation (r.Variant !=
	// sym.RV_NONE). Here "rel" is the relocation being applied, "sym"
	// is the symbol containing the chunk of data to which the
	// relocation applies, and "off" is the contents of the
	// to-be-relocated data item (from sym.P). Return is an updated
	// offset value.
	Archrelocvariant func(target *Target, ldr *loader.Loader, rel loader.Reloc,
		rv sym.RelocVariant, sym loader.Sym, offset int64, data []byte) (relocatedOffset int64)

	// Generate a trampoline for a call from s to rs if necessary. ri is
	// index of the relocation.
	Trampoline func(ctxt *Link, ldr *loader.Loader, ri int, rs, s loader.Sym)

	// Assembling the binary breaks into two phases, writing the code/data/
	// dwarf information (which is rather generic), and some more architecture
	// specific work like setting up the elf headers/dynamic relocations, etc.
	// The phases are called "Asmb" and "Asmb2". Asmb2 needs to be defined for
	// every architecture, but only if architecture has an Asmb function will
	// it be used for assembly.  Otherwise a generic assembly Asmb function is
	// used.
	Asmb  func(*Link, *loader.Loader)
	Asmb2 func(*Link, *loader.Loader)

	// Extreloc is an arch-specific hook that converts a Go relocation to an
	// external relocation. Return the external relocation and whether it is
	// needed.
	Extreloc func(*Target, *loader.Loader, loader.Reloc, loader.Sym) (loader.ExtReloc, bool)

	Elfreloc1      func(*Link, *OutBuf, *loader.Loader, loader.Sym, loader.ExtReloc, int, int64) bool
	ElfrelocSize   uint32 // size of an ELF relocation record, must match Elfreloc1.
	Elfsetupplt    func(ctxt *Link, plt, gotplt *loader.SymbolBuilder, dynamic loader.Sym)
	Gentext        func(*Link, *loader.Loader) // Generate text before addressing has been performed.
	Machoreloc1    func(*sys.Arch, *OutBuf, *loader.Loader, loader.Sym, loader.ExtReloc, int64) bool
	MachorelocSize uint32 // size of an Mach-O relocation record, must match Machoreloc1.
	PEreloc1       func(*sys.Arch, *OutBuf, *loader.Loader, loader.Sym, loader.ExtReloc, int64) bool
	Xcoffreloc1    func(*sys.Arch, *OutBuf, *loader.Loader, loader.Sym, loader.ExtReloc, int64) bool

	// Generate additional symbols for the native symbol table just prior to
	// code generation.
	GenSymsLate func(*Link, *loader.Loader)

	// TLSIEtoLE converts a TLS Initial Executable relocation to
	// a TLS Local Executable relocation.
	//
	// This is possible when a TLS IE relocation refers to a local
	// symbol in an executable, which is typical when internally
	// linking PIE binaries.
	TLSIEtoLE func(P []byte, off, size int)

	// optional override for assignAddress
	AssignAddress func(ldr *loader.Loader, sect *sym.Section, n int, s loader.Sym, va uint64, isTramp bool) (*sym.Section, int, uint64)
}

var (
	thearch Arch
	lcSize  int32
	rpath   Rpath
	spSize  int32
	symSize int32
)

const (
	MINFUNC = 16 // minimum size for a function
)

// DynlinkingGo reports whether we are producing Go code that can live
// in separate shared libraries linked together at runtime.
func (ctxt *Link) DynlinkingGo() bool {
	if !ctxt.Loaded {
		panic("DynlinkingGo called before all symbols loaded")
	}
	return ctxt.BuildMode == BuildModeShared || ctxt.linkShared || ctxt.BuildMode == BuildModePlugin || ctxt.canUsePlugins
}

// CanUsePlugins reports whether a plugins can be used
func (ctxt *Link) CanUsePlugins() bool {
	if !ctxt.Loaded {
		panic("CanUsePlugins called before all symbols loaded")
	}
	return ctxt.canUsePlugins
}

// NeedCodeSign reports whether we need to code-sign the output binary.
func (ctxt *Link) NeedCodeSign() bool {
	return ctxt.IsDarwin() && ctxt.IsARM64()
}

var (
	dynlib          []string
	ldflag          []string
	havedynamic     int
	Funcalign       int
	iscgo           bool
	elfglobalsymndx int
	interpreter     string

	debug_s bool // backup old value of debug['s']
	HEADR   int32

	nerrors  int
	liveness int64

	// See -strictdups command line flag.
	checkStrictDups   int // 0=off 1=warning 2=error
	strictDupMsgCount int
)

var (
	Segtext      sym.Segment
	Segrodata    sym.Segment
	Segrelrodata sym.Segment
	Segdata      sym.Segment
	Segdwarf     sym.Segment

	Segments = []*sym.Segment{&Segtext, &Segrodata, &Segrelrodata, &Segdata, &Segdwarf}
)

const pkgdef = "__.PKGDEF"

var (
	// externalobj is set to true if we see an object compiled by
	// the host compiler that is not from a package that is known
	// to support internal linking mode.
	externalobj = false

	// unknownObjFormat is set to true if we see an object whose
	// format we don't recognize.
	unknownObjFormat = false

	theline string
)

func Lflag(ctxt *Link, arg string) {
	ctxt.Libdir = append(ctxt.Libdir, arg)
}

/*
 * Unix doesn't like it when we write to a running (or, sometimes,
 * recently run) binary, so remove the output file before writing it.
 * On Windows 7, remove() can force a subsequent create() to fail.
 * S_ISREG() does not exist on Plan 9.
 */
func mayberemoveoutfile() {
	if fi, err := os.Lstat(*flagOutfile); err == nil && !fi.Mode().IsRegular() {
		return
	}
	os.Remove(*flagOutfile)
}

func libinit(ctxt *Link) {
	Funcalign = thearch.Funcalign

	// add goroot to the end of the libdir list.
	suffix := ""

	suffixsep := ""
	if *flagInstallSuffix != "" {
		suffixsep = "_"
		suffix = *flagInstallSuffix
	} else if *flagRace {
		suffixsep = "_"
		suffix = "race"
	} else if *flagMsan {
		suffixsep = "_"
		suffix = "msan"
	}

	Lflag(ctxt, filepath.Join(buildcfg.GOROOT, "pkg", fmt.Sprintf("%s_%s%s%s", buildcfg.GOOS, buildcfg.GOARCH, suffixsep, suffix)))

	mayberemoveoutfile()

	if err := ctxt.Out.Open(*flagOutfile); err != nil {
		Exitf("cannot create %s: %v", *flagOutfile, err)
	}

	if *flagEntrySymbol == "" {
		switch ctxt.BuildMode {
		case BuildModeCShared, BuildModeCArchive:
			*flagEntrySymbol = fmt.Sprintf("_rt0_%s_%s_lib", buildcfg.GOARCH, buildcfg.GOOS)
		case BuildModeExe, BuildModePIE:
			*flagEntrySymbol = fmt.Sprintf("_rt0_%s_%s", buildcfg.GOARCH, buildcfg.GOOS)
		case BuildModeShared, BuildModePlugin:
			// No *flagEntrySymbol for -buildmode=shared and plugin
		default:
			Errorf(nil, "unknown *flagEntrySymbol for buildmode %v", ctxt.BuildMode)
		}
	}
}

func exitIfErrors() {
	if nerrors != 0 || checkStrictDups > 1 && strictDupMsgCount > 0 {
		mayberemoveoutfile()
		Exit(2)
	}

}

func errorexit() {
	exitIfErrors()
	Exit(0)
}

func loadinternal(ctxt *Link, name string) *sym.Library {
	zerofp := goobj.FingerprintType{}
	if ctxt.linkShared && ctxt.PackageShlib != nil {
		if shlib := ctxt.PackageShlib[name]; shlib != "" {
			return addlibpath(ctxt, "internal", "internal", "", name, shlib, zerofp)
		}
	}
	if ctxt.PackageFile != nil {
		if pname := ctxt.PackageFile[name]; pname != "" {
			return addlibpath(ctxt, "internal", "internal", pname, name, "", zerofp)
		}
		ctxt.Logf("loadinternal: cannot find %s\n", name)
		return nil
	}

	for _, libdir := range ctxt.Libdir {
		if ctxt.linkShared {
			shlibname := filepath.Join(libdir, name+".shlibname")
			if ctxt.Debugvlog != 0 {
				ctxt.Logf("searching for %s.a in %s\n", name, shlibname)
			}
			if _, err := os.Stat(shlibname); err == nil {
				return addlibpath(ctxt, "internal", "internal", "", name, shlibname, zerofp)
			}
		}
		pname := filepath.Join(libdir, name+".a")
		if ctxt.Debugvlog != 0 {
			ctxt.Logf("searching for %s.a in %s\n", name, pname)
		}
		if _, err := os.Stat(pname); err == nil {
			return addlibpath(ctxt, "internal", "internal", pname, name, "", zerofp)
		}
	}

	ctxt.Logf("warning: unable to find %s.a\n", name)
	return nil
}

// extld returns the current external linker.
func (ctxt *Link) extld() string {
	if *flagExtld == "" {
		*flagExtld = "gcc"
	}
	return *flagExtld
}

// findLibPathCmd uses cmd command to find gcc library libname.
// It returns library full path if found, or "none" if not found.
func (ctxt *Link) findLibPathCmd(cmd, libname string) string {
	extld := ctxt.extld()
	args := hostlinkArchArgs(ctxt.Arch)
	args = append(args, cmd)
	if ctxt.Debugvlog != 0 {
		ctxt.Logf("%s %v\n", extld, args)
	}
	out, err := exec.Command(extld, args...).Output()
	if err != nil {
		if ctxt.Debugvlog != 0 {
			ctxt.Logf("not using a %s file because compiler failed\n%v\n%s\n", libname, err, out)
		}
		return "none"
	}
	return strings.TrimSpace(string(out))
}

// findLibPath searches for library libname.
// It returns library full path if found, or "none" if not found.
func (ctxt *Link) findLibPath(libname string) string {
	return ctxt.findLibPathCmd("--print-file-name="+libname, libname)
}

func (ctxt *Link) loadlib() {
	var flags uint32
	switch *FlagStrictDups {
	case 0:
		// nothing to do
	case 1, 2:
		flags |= loader.FlagStrictDups
	default:
		log.Fatalf("invalid -strictdups flag value %d", *FlagStrictDups)
	}
	if !buildcfg.Experiment.RegabiWrappers {
		// Use ABI aliases if ABI wrappers are not used.
		flags |= loader.FlagUseABIAlias
	}
	elfsetstring1 := func(str string, off int) { elfsetstring(ctxt, 0, str, off) }
	ctxt.loader = loader.NewLoader(flags, elfsetstring1, &ctxt.ErrorReporter.ErrorReporter)
	ctxt.ErrorReporter.SymName = func(s loader.Sym) string {
		return ctxt.loader.SymName(s)
	}

	// ctxt.Library grows during the loop, so not a range loop.
	i := 0
	for ; i < len(ctxt.Library); i++ {
		lib := ctxt.Library[i]
		if lib.Shlib == "" {
			if ctxt.Debugvlog > 1 {
				ctxt.Logf("autolib: %s (from %s)\n", lib.File, lib.Objref)
			}
			loadobjfile(ctxt, lib)
		}
	}

	// load internal packages, if not already
	if *flagRace {
		loadinternal(ctxt, "runtime/race")
	}
	if *flagMsan {
		loadinternal(ctxt, "runtime/msan")
	}
	loadinternal(ctxt, "runtime")
	for ; i < len(ctxt.Library); i++ {
		lib := ctxt.Library[i]
		if lib.Shlib == "" {
			loadobjfile(ctxt, lib)
		}
	}
	// At this point, the Go objects are "preloaded". Not all the symbols are
	// added to the symbol table (only defined package symbols are). Looking
	// up symbol by name may not get expected result.

	iscgo = ctxt.LibraryByPkg["runtime/cgo"] != nil

	// Plugins a require cgo support to function. Similarly, plugins may require additional
	// internal linker support on some platforms which may not be implemented.
	ctxt.canUsePlugins = ctxt.LibraryByPkg["plugin"] != nil && iscgo

	// We now have enough information to determine the link mode.
	determineLinkMode(ctxt)

	if ctxt.LinkMode == LinkExternal && !iscgo && !(buildcfg.GOOS == "darwin" && ctxt.BuildMode != BuildModePlugin && ctxt.Arch.Family == sys.AMD64) {
		// This indicates a user requested -linkmode=external.
		// The startup code uses an import of runtime/cgo to decide
		// whether to initialize the TLS.  So give it one. This could
		// be handled differently but it's an unusual case.
		if lib := loadinternal(ctxt, "runtime/cgo"); lib != nil && lib.Shlib == "" {
			if ctxt.BuildMode == BuildModeShared || ctxt.linkShared {
				Exitf("cannot implicitly include runtime/cgo in a shared library")
			}
			for ; i < len(ctxt.Library); i++ {
				lib := ctxt.Library[i]
				if lib.Shlib == "" {
					loadobjfile(ctxt, lib)
				}
			}
		}
	}

	// Add non-package symbols and references of externally defined symbols.
	ctxt.loader.LoadSyms(ctxt.Arch)

	// Load symbols from shared libraries, after all Go object symbols are loaded.
	for _, lib := range ctxt.Library {
		if lib.Shlib != "" {
			if ctxt.Debugvlog > 1 {
				ctxt.Logf("autolib: %s (from %s)\n", lib.Shlib, lib.Objref)
			}
			ldshlibsyms(ctxt, lib.Shlib)
		}
	}

	// Process cgo directives (has to be done before host object loading).
	ctxt.loadcgodirectives()

	// Conditionally load host objects, or setup for external linking.
	hostobjs(ctxt)
	hostlinksetup(ctxt)

	if ctxt.LinkMode == LinkInternal && len(hostobj) != 0 {
		// If we have any undefined symbols in external
		// objects, try to read them from the libgcc file.
		any := false
		undefs := ctxt.loader.UndefinedRelocTargets(1)
		if len(undefs) > 0 {
			any = true
		}
		if any {
			if *flagLibGCC == "" {
				*flagLibGCC = ctxt.findLibPathCmd("--print-libgcc-file-name", "libgcc")
			}
			if runtime.GOOS == "openbsd" && *flagLibGCC == "libgcc.a" {
				// On OpenBSD `clang --print-libgcc-file-name` returns "libgcc.a".
				// In this case we fail to load libgcc.a and can encounter link
				// errors - see if we can find libcompiler_rt.a instead.
				*flagLibGCC = ctxt.findLibPathCmd("--print-file-name=libcompiler_rt.a", "libcompiler_rt")
			}
			if ctxt.HeadType == objabi.Hwindows {
				if p := ctxt.findLibPath("libmingwex.a"); p != "none" {
					hostArchive(ctxt, p)
				}
				if p := ctxt.findLibPath("libmingw32.a"); p != "none" {
					hostArchive(ctxt, p)
				}
				// Link libmsvcrt.a to resolve '__acrt_iob_func' symbol
				// (see https://golang.org/issue/23649 for details).
				if p := ctxt.findLibPath("libmsvcrt.a"); p != "none" {
					hostArchive(ctxt, p)
				}
				// TODO: maybe do something similar to peimporteddlls to collect all lib names
				// and try link them all to final exe just like libmingwex.a and libmingw32.a:
				/*
					for:
					#cgo windows LDFLAGS: -lmsvcrt -lm
					import:
					libmsvcrt.a libm.a
				*/
			}
			if *flagLibGCC != "none" {
				hostArchive(ctxt, *flagLibGCC)
			}
		}
	}

	// We've loaded all the code now.
	ctxt.Loaded = true

	importcycles()

	strictDupMsgCount = ctxt.loader.NStrictDupMsgs()
}

// loadcgodirectives reads the previously discovered cgo directives, creating
// symbols in preparation for host object loading or use later in the link.
func (ctxt *Link) loadcgodirectives() {
	l := ctxt.loader
	hostObjSyms := make(map[loader.Sym]struct{})
	for _, d := range ctxt.cgodata {
		setCgoAttr(ctxt, d.file, d.pkg, d.directives, hostObjSyms)
	}
	ctxt.cgodata = nil

	if ctxt.LinkMode == LinkInternal {
		// Drop all the cgo_import_static declarations.
		// Turns out we won't be needing them.
		for symIdx := range hostObjSyms {
			if l.SymType(symIdx) == sym.SHOSTOBJ {
				// If a symbol was marked both
				// cgo_import_static and cgo_import_dynamic,
				// then we want to make it cgo_import_dynamic
				// now.
				su := l.MakeSymbolUpdater(symIdx)
				if l.SymExtname(symIdx) != "" && l.SymDynimplib(symIdx) != "" && !(l.AttrCgoExportStatic(symIdx) || l.AttrCgoExportDynamic(symIdx)) {
					su.SetType(sym.SDYNIMPORT)
				} else {
					su.SetType(0)
				}
			}
		}
	}
}

// Set up flags and special symbols depending on the platform build mode.
// This version works with loader.Loader.
func (ctxt *Link) linksetup() {
	switch ctxt.BuildMode {
	case BuildModeCShared, BuildModePlugin:
		symIdx := ctxt.loader.LookupOrCreateSym("runtime.islibrary", 0)
		sb := ctxt.loader.MakeSymbolUpdater(symIdx)
		sb.SetType(sym.SNOPTRDATA)
		sb.AddUint8(1)
	case BuildModeCArchive:
		symIdx := ctxt.loader.LookupOrCreateSym("runtime.isarchive", 0)
		sb := ctxt.loader.MakeSymbolUpdater(symIdx)
		sb.SetType(sym.SNOPTRDATA)
		sb.AddUint8(1)
	}

	// Recalculate pe parameters now that we have ctxt.LinkMode set.
	if ctxt.HeadType == objabi.Hwindows {
		Peinit(ctxt)
	}

	if ctxt.HeadType == objabi.Hdarwin && ctxt.LinkMode == LinkExternal {
		*FlagTextAddr = 0
	}

	// If there are no dynamic libraries needed, gcc disables dynamic linking.
	// Because of this, glibc's dynamic ELF loader occasionally (like in version 2.13)
	// assumes that a dynamic binary always refers to at least one dynamic library.
	// Rather than be a source of test cases for glibc, disable dynamic linking
	// the same way that gcc would.
	//
	// Exception: on OS X, programs such as Shark only work with dynamic
	// binaries, so leave it enabled on OS X (Mach-O) binaries.
	// Also leave it enabled on Solaris which doesn't support
	// statically linked binaries.
	if ctxt.BuildMode == BuildModeExe {
		if havedynamic == 0 && ctxt.HeadType != objabi.Hdarwin && ctxt.HeadType != objabi.Hsolaris {
			*FlagD = true
		}
	}

	if ctxt.LinkMode == LinkExternal && ctxt.Arch.Family == sys.PPC64 && buildcfg.GOOS != "aix" {
		toc := ctxt.loader.LookupOrCreateSym(".TOC.", 0)
		sb := ctxt.loader.MakeSymbolUpdater(toc)
		sb.SetType(sym.SDYNIMPORT)
	}

	// The Android Q linker started to complain about underalignment of the our TLS
	// section. We don't actually use the section on android, so don't
	// generate it.
	if buildcfg.GOOS != "android" {
		tlsg := ctxt.loader.LookupOrCreateSym("runtime.tlsg", 0)
		sb := ctxt.loader.MakeSymbolUpdater(tlsg)

		// runtime.tlsg is used for external linking on platforms that do not define
		// a variable to hold g in assembly (currently only intel).
		if sb.Type() == 0 {
			sb.SetType(sym.STLSBSS)
			sb.SetSize(int64(ctxt.Arch.PtrSize))
		} else if sb.Type() != sym.SDYNIMPORT {
			Errorf(nil, "runtime declared tlsg variable %v", sb.Type())
		}
		ctxt.loader.SetAttrReachable(tlsg, true)
		ctxt.Tlsg = tlsg
	}

	var moduledata loader.Sym
	var mdsb *loader.SymbolBuilder
	if ctxt.BuildMode == BuildModePlugin {
		moduledata = ctxt.loader.LookupOrCreateSym("local.pluginmoduledata", 0)
		mdsb = ctxt.loader.MakeSymbolUpdater(moduledata)
		ctxt.loader.SetAttrLocal(moduledata, true)
	} else {
		moduledata = ctxt.loader.LookupOrCreateSym("runtime.firstmoduledata", 0)
		mdsb = ctxt.loader.MakeSymbolUpdater(moduledata)
	}
	if mdsb.Type() != 0 && mdsb.Type() != sym.SDYNIMPORT {
		// If the module (toolchain-speak for "executable or shared
		// library") we are linking contains the runtime package, it
		// will define the runtime.firstmoduledata symbol and we
		// truncate it back to 0 bytes so we can define its entire
		// contents in symtab.go:symtab().
		mdsb.SetSize(0)

		// In addition, on ARM, the runtime depends on the linker
		// recording the value of GOARM.
		if ctxt.Arch.Family == sys.ARM {
			goarm := ctxt.loader.LookupOrCreateSym("runtime.goarm", 0)
			sb := ctxt.loader.MakeSymbolUpdater(goarm)
			sb.SetType(sym.SDATA)
			sb.SetSize(0)
			sb.AddUint8(uint8(buildcfg.GOARM))
		}

		// Set runtime.disableMemoryProfiling bool if
		// runtime.MemProfile is not retained in the binary after
		// deadcode (and we're not dynamically linking).
		memProfile := ctxt.loader.Lookup("runtime.MemProfile", sym.SymVerABIInternal)
		if memProfile != 0 && !ctxt.loader.AttrReachable(memProfile) && !ctxt.DynlinkingGo() {
			memProfSym := ctxt.loader.LookupOrCreateSym("runtime.disableMemoryProfiling", 0)
			sb := ctxt.loader.MakeSymbolUpdater(memProfSym)
			sb.SetType(sym.SDATA)
			sb.SetSize(0)
			sb.AddUint8(1) // true bool
		}
	} else {
		// If OTOH the module does not contain the runtime package,
		// create a local symbol for the moduledata.
		moduledata = ctxt.loader.LookupOrCreateSym("local.moduledata", 0)
		mdsb = ctxt.loader.MakeSymbolUpdater(moduledata)
		ctxt.loader.SetAttrLocal(moduledata, true)
	}
	// In all cases way we mark the moduledata as noptrdata to hide it from
	// the GC.
	mdsb.SetType(sym.SNOPTRDATA)
	ctxt.loader.SetAttrReachable(moduledata, true)
	ctxt.Moduledata = moduledata

	if ctxt.Arch == sys.Arch386 && ctxt.HeadType != objabi.Hwindows {
		if (ctxt.BuildMode == BuildModeCArchive && ctxt.IsELF) || ctxt.BuildMode == BuildModeCShared || ctxt.BuildMode == BuildModePIE || ctxt.DynlinkingGo() {
			got := ctxt.loader.LookupOrCreateSym("_GLOBAL_OFFSET_TABLE_", 0)
			sb := ctxt.loader.MakeSymbolUpdater(got)
			sb.SetType(sym.SDYNIMPORT)
			ctxt.loader.SetAttrReachable(got, true)
		}
	}

	// DWARF-gen and other phases require that the unit Textp slices
	// be populated, so that it can walk the functions in each unit.
	// Call into the loader to do this (requires that we collect the
	// set of internal libraries first). NB: might be simpler if we
	// moved isRuntimeDepPkg to cmd/internal and then did the test in
	// loader.AssignTextSymbolOrder.
	ctxt.Library = postorder(ctxt.Library)
	intlibs := []bool{}
	for _, lib := range ctxt.Library {
		intlibs = append(intlibs, isRuntimeDepPkg(lib.Pkg))
	}
	ctxt.Textp = ctxt.loader.AssignTextSymbolOrder(ctxt.Library, intlibs, ctxt.Textp)
}

// mangleTypeSym shortens the names of symbols that represent Go types
// if they are visible in the symbol table.
//
// As the names of these symbols are derived from the string of
// the type, they can run to many kilobytes long. So we shorten
// them using a SHA-1 when the name appears in the final binary.
// This also removes characters that upset external linkers.
//
// These are the symbols that begin with the prefix 'type.' and
// contain run-time type information used by the runtime and reflect
// packages. All Go binaries contain these symbols, but only
// those programs loaded dynamically in multiple parts need these
// symbols to have entries in the symbol table.
func (ctxt *Link) mangleTypeSym() {
	if ctxt.BuildMode != BuildModeShared && !ctxt.linkShared && ctxt.BuildMode != BuildModePlugin && !ctxt.CanUsePlugins() {
		return
	}

	ldr := ctxt.loader
	for s := loader.Sym(1); s < loader.Sym(ldr.NSym()); s++ {
		if !ldr.AttrReachable(s) && !ctxt.linkShared {
			// If -linkshared, the GCProg generation code may need to reach
			// out to the shared library for the type descriptor's data, even
			// the type descriptor itself is not actually needed at run time
			// (therefore not reachable). We still need to mangle its name,
			// so it is consistent with the one stored in the shared library.
			continue
		}
		name := ldr.SymName(s)
		newName := typeSymbolMangle(name)
		if newName != name {
			ldr.SetSymExtname(s, newName)

			// When linking against a shared library, the Go object file may
			// have reference to the original symbol name whereas the shared
			// library provides a symbol with the mangled name. We need to
			// copy the payload of mangled to original.
			// XXX maybe there is a better way to do this.
			dup := ldr.Lookup(newName, ldr.SymVersion(s))
			if dup != 0 {
				st := ldr.SymType(s)
				dt := ldr.SymType(dup)
				if st == sym.Sxxx && dt != sym.Sxxx {
					ldr.CopySym(dup, s)
				}
			}
		}
	}
}

// typeSymbolMangle mangles the given symbol name into something shorter.
//
// Keep the type.. prefix, which parts of the linker (like the
// DWARF generator) know means the symbol is not decodable.
// Leave type.runtime. symbols alone, because other parts of
// the linker manipulates them.
func typeSymbolMangle(name string) string {
	if !strings.HasPrefix(name, "type.") {
		return name
	}
	if strings.HasPrefix(name, "type.runtime.") {
		return name
	}
	if len(name) <= 14 && !strings.Contains(name, "@") { // Issue 19529
		return name
	}
	hash := sha1.Sum([]byte(name))
	prefix := "type."
	if name[5] == '.' {
		prefix = "type.."
	}
	return prefix + base64.StdEncoding.EncodeToString(hash[:6])
}

/*
 * look for the next file in an archive.
 * adapted from libmach.
 */
func nextar(bp *bio.Reader, off int64, a *ArHdr) int64 {
	if off&1 != 0 {
		off++
	}
	bp.MustSeek(off, 0)
	var buf [SAR_HDR]byte
	if n, err := io.ReadFull(bp, buf[:]); err != nil {
		if n == 0 && err != io.EOF {
			return -1
		}
		return 0
	}

	a.name = artrim(buf[0:16])
	a.date = artrim(buf[16:28])
	a.uid = artrim(buf[28:34])
	a.gid = artrim(buf[34:40])
	a.mode = artrim(buf[40:48])
	a.size = artrim(buf[48:58])
	a.fmag = artrim(buf[58:60])

	arsize := atolwhex(a.size)
	if arsize&1 != 0 {
		arsize++
	}
	return arsize + SAR_HDR
}

func loadobjfile(ctxt *Link, lib *sym.Library) {
	pkg := objabi.PathToPrefix(lib.Pkg)

	if ctxt.Debugvlog > 1 {
		ctxt.Logf("ldobj: %s (%s)\n", lib.File, pkg)
	}
	f, err := bio.Open(lib.File)
	if err != nil {
		Exitf("cannot open file %s: %v", lib.File, err)
	}
	defer f.Close()
	defer func() {
		if pkg == "main" && !lib.Main {
			Exitf("%s: not package main", lib.File)
		}
	}()

	for i := 0; i < len(ARMAG); i++ {
		if c, err := f.ReadByte(); err == nil && c == ARMAG[i] {
			continue
		}

		/* load it as a regular file */
		l := f.MustSeek(0, 2)
		f.MustSeek(0, 0)
		ldobj(ctxt, f, lib, l, lib.File, lib.File)
		return
	}

	/*
	 * load all the object files from the archive now.
	 * this gives us sequential file access and keeps us
	 * from needing to come back later to pick up more
	 * objects.  it breaks the usual C archive model, but
	 * this is Go, not C.  the common case in Go is that
	 * we need to load all the objects, and then we throw away
	 * the individual symbols that are unused.
	 *
	 * loading every object will also make it possible to
	 * load foreign objects not referenced by __.PKGDEF.
	 */
	var arhdr ArHdr
	off := f.Offset()
	for {
		l := nextar(f, off, &arhdr)
		if l == 0 {
			break
		}
		if l < 0 {
			Exitf("%s: malformed archive", lib.File)
		}
		off += l

		// __.PKGDEF isn't a real Go object file, and it's
		// absent in -linkobj builds anyway. Skipping it
		// ensures consistency between -linkobj and normal
		// build modes.
		if arhdr.name == pkgdef {
			continue
		}

		// Skip other special (non-object-file) sections that
		// build tools may have added. Such sections must have
		// short names so that the suffix is not truncated.
		if len(arhdr.name) < 16 {
			if ext := filepath.Ext(arhdr.name); ext != ".o" && ext != ".syso" {
				continue
			}
		}

		pname := fmt.Sprintf("%s(%s)", lib.File, arhdr.name)
		l = atolwhex(arhdr.size)
		ldobj(ctxt, f, lib, l, pname, lib.File)
	}
}

type Hostobj struct {
	ld     func(*Link, *bio.Reader, string, int64, string)
	pkg    string
	pn     string
	file   string
	off    int64
	length int64
}

var hostobj []Hostobj

// These packages can use internal linking mode.
// Others trigger external mode.
var internalpkg = []string{
	"crypto/x509",
	"net",
	"os/user",
	"runtime/cgo",
	"runtime/race",
	"runtime/msan",
}

func ldhostobj(ld func(*Link, *bio.Reader, string, int64, string), headType objabi.HeadType, f *bio.Reader, pkg string, length int64, pn string, file string) *Hostobj {
	isinternal := false
	for _, intpkg := range internalpkg {
		if pkg == intpkg {
			isinternal = true
			break
		}
	}

	// DragonFly declares errno with __thread, which results in a symbol
	// type of R_386_TLS_GD or R_X86_64_TLSGD. The Go linker does not
	// currently know how to handle TLS relocations, hence we have to
	// force external linking for any libraries that link in code that
	// uses errno. This can be removed if the Go linker ever supports
	// these relocation types.
	if headType == objabi.Hdragonfly {
		if pkg == "net" || pkg == "os/user" {
			isinternal = false
		}
	}

	if !isinternal {
		externalobj = true
	}

	hostobj = append(hostobj, Hostobj{})
	h := &hostobj[len(hostobj)-1]
	h.ld = ld
	h.pkg = pkg
	h.pn = pn
	h.file = file
	h.off = f.Offset()
	h.length = length
	return h
}

func hostobjs(ctxt *Link) {
	if ctxt.LinkMode != LinkInternal {
		return
	}
	var h *Hostobj

	for i := 0; i < len(hostobj); i++ {
		h = &hostobj[i]
		f, err := bio.Open(h.file)
		if err != nil {
			Exitf("cannot reopen %s: %v", h.pn, err)
		}

		f.MustSeek(h.off, 0)
		if h.ld == nil {
			Errorf(nil, "%s: unrecognized object file format", h.pn)
			continue
		}
		h.ld(ctxt, f, h.pkg, h.length, h.pn)
		f.Close()
	}
}

func hostlinksetup(ctxt *Link) {
	if ctxt.LinkMode != LinkExternal {
		return
	}

	// For external link, record that we need to tell the external linker -s,
	// and turn off -s internally: the external linker needs the symbol
	// information for its final link.
	debug_s = *FlagS
	*FlagS = false

	// create temporary directory and arrange cleanup
	if *flagTmpdir == "" {
		dir, err := ioutil.TempDir("", "go-link-")
		if err != nil {
			log.Fatal(err)
		}
		*flagTmpdir = dir
		ownTmpDir = true
		AtExit(func() {
			ctxt.Out.Close()
			os.RemoveAll(*flagTmpdir)
		})
	}

	// change our output to temporary object file
	if err := ctxt.Out.Close(); err != nil {
		Exitf("error closing output file")
	}
	mayberemoveoutfile()

	p := filepath.Join(*flagTmpdir, "go.o")
	if err := ctxt.Out.Open(p); err != nil {
		Exitf("cannot create %s: %v", p, err)
	}
}

// hostobjCopy creates a copy of the object files in hostobj in a
// temporary directory.
func hostobjCopy() (paths []string) {
	var wg sync.WaitGroup
	sema := make(chan struct{}, runtime.NumCPU()) // limit open file descriptors
	for i, h := range hostobj {
		h := h
		dst := filepath.Join(*flagTmpdir, fmt.Sprintf("%06d.o", i))
		paths = append(paths, dst)

		wg.Add(1)
		go func() {
			sema <- struct{}{}
			defer func() {
				<-sema
				wg.Done()
			}()
			f, err := os.Open(h.file)
			if err != nil {
				Exitf("cannot reopen %s: %v", h.pn, err)
			}
			defer f.Close()
			if _, err := f.Seek(h.off, 0); err != nil {
				Exitf("cannot seek %s: %v", h.pn, err)
			}

			w, err := os.Create(dst)
			if err != nil {
				Exitf("cannot create %s: %v", dst, err)
			}
			if _, err := io.CopyN(w, f, h.length); err != nil {
				Exitf("cannot write %s: %v", dst, err)
			}
			if err := w.Close(); err != nil {
				Exitf("cannot close %s: %v", dst, err)
			}
		}()
	}
	wg.Wait()
	return paths
}

// writeGDBLinkerScript creates gcc linker script file in temp
// directory. writeGDBLinkerScript returns created file path.
// The script is used to work around gcc bug
// (see https://golang.org/issue/20183 for details).
func writeGDBLinkerScript() string {
	name := "fix_debug_gdb_scripts.ld"
	path := filepath.Join(*flagTmpdir, name)
	src := `SECTIONS
{
  .debug_gdb_scripts BLOCK(__section_alignment__) (NOLOAD) :
  {
    *(.debug_gdb_scripts)
  }
}
INSERT AFTER .debug_types;
`
	err := ioutil.WriteFile(path, []byte(src), 0666)
	if err != nil {
		Errorf(nil, "WriteFile %s failed: %v", name, err)
	}
	return path
}

// archive builds a .a archive from the hostobj object files.
func (ctxt *Link) archive() {
	if ctxt.BuildMode != BuildModeCArchive {
		return
	}

	exitIfErrors()

	if *flagExtar == "" {
		*flagExtar = "ar"
	}

	mayberemoveoutfile()

	// Force the buffer to flush here so that external
	// tools will see a complete file.
	if err := ctxt.Out.Close(); err != nil {
		Exitf("error closing %v", *flagOutfile)
	}

	argv := []string{*flagExtar, "-q", "-c", "-s"}
	if ctxt.HeadType == objabi.Haix {
		argv = append(argv, "-X64")
	}
	argv = append(argv, *flagOutfile)
	argv = append(argv, filepath.Join(*flagTmpdir, "go.o"))
	argv = append(argv, hostobjCopy()...)

	if ctxt.Debugvlog != 0 {
		ctxt.Logf("archive: %s\n", strings.Join(argv, " "))
	}

	// If supported, use syscall.Exec() to invoke the archive command,
	// which should be the final remaining step needed for the link.
	// This will reduce peak RSS for the link (and speed up linking of
	// large applications), since when the archive command runs we
	// won't be holding onto all of the linker's live memory.
	if syscallExecSupported && !ownTmpDir {
		runAtExitFuncs()
		ctxt.execArchive(argv)
		panic("should not get here")
	}

	// Otherwise invoke 'ar' in the usual way (fork + exec).
	if out, err := exec.Command(argv[0], argv[1:]...).CombinedOutput(); err != nil {
		Exitf("running %s failed: %v\n%s", argv[0], err, out)
	}
}

func (ctxt *Link) hostlink() {
	if ctxt.LinkMode != LinkExternal || nerrors > 0 {
		return
	}
	if ctxt.BuildMode == BuildModeCArchive {
		return
	}

	var argv []string
	argv = append(argv, ctxt.extld())
	argv = append(argv, hostlinkArchArgs(ctxt.Arch)...)

	if *FlagS || debug_s {
		if ctxt.HeadType == objabi.Hdarwin {
			// Recent versions of macOS print
			//	ld: warning: option -s is obsolete and being ignored
			// so do not pass any arguments.
		} else {
			argv = append(argv, "-s")
		}
	}

	// On darwin, whether to combine DWARF into executable.
	// Only macOS supports unmapped segments such as our __DWARF segment.
	combineDwarf := ctxt.IsDarwin() && !*FlagS && !*FlagW && !debug_s && machoPlatform == PLATFORM_MACOS

	switch ctxt.HeadType {
	case objabi.Hdarwin:
		if combineDwarf {
			// Leave room for DWARF combining.
			// -headerpad is incompatible with -fembed-bitcode.
			argv = append(argv, "-Wl,-headerpad,1144")
		}
		if ctxt.DynlinkingGo() && buildcfg.GOOS != "ios" {
			// -flat_namespace is deprecated on iOS.
			// It is useful for supporting plugins. We don't support plugins on iOS.
			argv = append(argv, "-Wl,-flat_namespace")
		}
		if !combineDwarf {
			argv = append(argv, "-Wl,-S") // suppress STAB (symbolic debugging) symbols
		}
	case objabi.Hopenbsd:
		argv = append(argv, "-Wl,-nopie")
		argv = append(argv, "-pthread")
	case objabi.Hwindows:
		if windowsgui {
			argv = append(argv, "-mwindows")
		} else {
			argv = append(argv, "-mconsole")
		}
		// Mark as having awareness of terminal services, to avoid
		// ancient compatibility hacks.
		argv = append(argv, "-Wl,--tsaware")

		// Enable DEP
		argv = append(argv, "-Wl,--nxcompat")

		argv = append(argv, fmt.Sprintf("-Wl,--major-os-version=%d", PeMinimumTargetMajorVersion))
		argv = append(argv, fmt.Sprintf("-Wl,--minor-os-version=%d", PeMinimumTargetMinorVersion))
		argv = append(argv, fmt.Sprintf("-Wl,--major-subsystem-version=%d", PeMinimumTargetMajorVersion))
		argv = append(argv, fmt.Sprintf("-Wl,--minor-subsystem-version=%d", PeMinimumTargetMinorVersion))
	case objabi.Haix:
		argv = append(argv, "-pthread")
		// prevent ld to reorder .text functions to keep the same
		// first/last functions for moduledata.
		argv = append(argv, "-Wl,-bnoobjreorder")
		// mcmodel=large is needed for every gcc generated files, but
		// ld still need -bbigtoc in order to allow larger TOC.
		argv = append(argv, "-mcmodel=large")
		argv = append(argv, "-Wl,-bbigtoc")
	}

	// Enable ASLR on Windows.
	addASLRargs := func(argv []string) []string {
		// Enable ASLR.
		argv = append(argv, "-Wl,--dynamicbase")
		// enable high-entropy ASLR on 64-bit.
		if ctxt.Arch.PtrSize >= 8 {
			argv = append(argv, "-Wl,--high-entropy-va")
		}
		return argv
	}

	switch ctxt.BuildMode {
	case BuildModeExe:
		if ctxt.HeadType == objabi.Hdarwin {
			if machoPlatform == PLATFORM_MACOS && ctxt.IsAMD64() {
				argv = append(argv, "-Wl,-no_pie")
				argv = append(argv, "-Wl,-pagezero_size,4000000")
			}
		}
	case BuildModePIE:
		switch ctxt.HeadType {
		case objabi.Hdarwin, objabi.Haix:
		case objabi.Hwindows:
			argv = addASLRargs(argv)
		default:
			// ELF.
			if ctxt.UseRelro() {
				argv = append(argv, "-Wl,-z,relro")
			}
			argv = append(argv, "-pie")
		}
	case BuildModeCShared:
		if ctxt.HeadType == objabi.Hdarwin {
			argv = append(argv, "-dynamiclib")
		} else {
			if ctxt.UseRelro() {
				argv = append(argv, "-Wl,-z,relro")
			}
			argv = append(argv, "-shared")
			if ctxt.HeadType == objabi.Hwindows {
				if *flagAslr {
					argv = addASLRargs(argv)
				}
			} else {
				// Pass -z nodelete to mark the shared library as
				// non-closeable: a dlclose will do nothing.
				argv = append(argv, "-Wl,-z,nodelete")
				// Only pass Bsymbolic on non-Windows.
				argv = append(argv, "-Wl,-Bsymbolic")
			}
		}
	case BuildModeShared:
		if ctxt.UseRelro() {
			argv = append(argv, "-Wl,-z,relro")
		}
		argv = append(argv, "-shared")
	case BuildModePlugin:
		if ctxt.HeadType == objabi.Hdarwin {
			argv = append(argv, "-dynamiclib")
		} else {
			if ctxt.UseRelro() {
				argv = append(argv, "-Wl,-z,relro")
			}
			argv = append(argv, "-shared")
		}
	}

	var altLinker string
	if ctxt.IsELF && ctxt.DynlinkingGo() {
		// We force all symbol resolution to be done at program startup
		// because lazy PLT resolution can use large amounts of stack at
		// times we cannot allow it to do so.
		argv = append(argv, "-Wl,-znow")

		// Do not let the host linker generate COPY relocations. These
		// can move symbols out of sections that rely on stable offsets
		// from the beginning of the section (like sym.STYPE).
		argv = append(argv, "-Wl,-znocopyreloc")

		if buildcfg.GOOS == "android" {
			// Use lld to avoid errors from default linker (issue #38838)
			altLinker = "lld"
		}

		if ctxt.Arch.InFamily(sys.ARM, sys.ARM64) && buildcfg.GOOS == "linux" {
			// On ARM, the GNU linker will generate COPY relocations
			// even with -znocopyreloc set.
			// https://sourceware.org/bugzilla/show_bug.cgi?id=19962
			//
			// On ARM64, the GNU linker will fail instead of
			// generating COPY relocations.
			//
			// In both cases, switch to gold.
			altLinker = "gold"

			// If gold is not installed, gcc will silently switch
			// back to ld.bfd. So we parse the version information
			// and provide a useful error if gold is missing.
			cmd := exec.Command(*flagExtld, "-fuse-ld=gold", "-Wl,--version")
			if out, err := cmd.CombinedOutput(); err == nil {
				if !bytes.Contains(out, []byte("GNU gold")) {
					log.Fatalf("ARM external linker must be gold (issue #15696), but is not: %s", out)
				}
			}
		}
	}
	if ctxt.Arch.Family == sys.ARM64 && buildcfg.GOOS == "freebsd" {
		// Switch to ld.bfd on freebsd/arm64.
		altLinker = "bfd"

		// Provide a useful error if ld.bfd is missing.
		cmd := exec.Command(*flagExtld, "-fuse-ld=bfd", "-Wl,--version")
		if out, err := cmd.CombinedOutput(); err == nil {
			if !bytes.Contains(out, []byte("GNU ld")) {
				log.Fatalf("ARM64 external linker must be ld.bfd (issue #35197), please install devel/binutils")
			}
		}
	}
	if altLinker != "" {
		argv = append(argv, "-fuse-ld="+altLinker)
	}

	if ctxt.IsELF && len(buildinfo) > 0 {
		argv = append(argv, fmt.Sprintf("-Wl,--build-id=0x%x", buildinfo))
	}

	// On Windows, given -o foo, GCC will append ".exe" to produce
	// "foo.exe".  We have decided that we want to honor the -o
	// option. To make this work, we append a '.' so that GCC
	// will decide that the file already has an extension. We
	// only want to do this when producing a Windows output file
	// on a Windows host.
	outopt := *flagOutfile
	if buildcfg.GOOS == "windows" && runtime.GOOS == "windows" && filepath.Ext(outopt) == "" {
		outopt += "."
	}
	argv = append(argv, "-o")
	argv = append(argv, outopt)

	if rpath.val != "" {
		argv = append(argv, fmt.Sprintf("-Wl,-rpath,%s", rpath.val))
	}

	if *flagInterpreter != "" {
		// Many linkers support both -I and the --dynamic-linker flags
		// to set the ELF interpreter, but lld only supports
		// --dynamic-linker so prefer that (ld on very old Solaris only
		// supports -I but that seems less important).
		argv = append(argv, fmt.Sprintf("-Wl,--dynamic-linker,%s", *flagInterpreter))
	}

	// Force global symbols to be exported for dlopen, etc.
	if ctxt.IsELF {
		argv = append(argv, "-rdynamic")
	}
	if ctxt.HeadType == objabi.Haix {
		fileName := xcoffCreateExportFile(ctxt)
		argv = append(argv, "-Wl,-bE:"+fileName)
	}

	const unusedArguments = "-Qunused-arguments"
	if linkerFlagSupported(ctxt.Arch, argv[0], altLinker, unusedArguments) {
		argv = append(argv, unusedArguments)
	}

	const compressDWARF = "-Wl,--compress-debug-sections=zlib-gnu"
	if ctxt.compressDWARF && linkerFlagSupported(ctxt.Arch, argv[0], altLinker, compressDWARF) {
		argv = append(argv, compressDWARF)
	}

	argv = append(argv, filepath.Join(*flagTmpdir, "go.o"))
	argv = append(argv, hostobjCopy()...)
	if ctxt.HeadType == objabi.Haix {
		// We want to have C files after Go files to remove
		// trampolines csects made by ld.
		argv = append(argv, "-nostartfiles")
		argv = append(argv, "/lib/crt0_64.o")

		extld := ctxt.extld()
		// Get starting files.
		getPathFile := func(file string) string {
			args := []string{"-maix64", "--print-file-name=" + file}
			out, err := exec.Command(extld, args...).CombinedOutput()
			if err != nil {
				log.Fatalf("running %s failed: %v\n%s", extld, err, out)
			}
			return strings.Trim(string(out), "\n")
		}
		argv = append(argv, getPathFile("crtcxa.o"))
		argv = append(argv, getPathFile("crtdbase.o"))
	}

	if ctxt.linkShared {
		seenDirs := make(map[string]bool)
		seenLibs := make(map[string]bool)
		addshlib := func(path string) {
			dir, base := filepath.Split(path)
			if !seenDirs[dir] {
				argv = append(argv, "-L"+dir)
				if !rpath.set {
					argv = append(argv, "-Wl,-rpath="+dir)
				}
				seenDirs[dir] = true
			}
			base = strings.TrimSuffix(base, ".so")
			base = strings.TrimPrefix(base, "lib")
			if !seenLibs[base] {
				argv = append(argv, "-l"+base)
				seenLibs[base] = true
			}
		}
		for _, shlib := range ctxt.Shlibs {
			addshlib(shlib.Path)
			for _, dep := range shlib.Deps {
				if dep == "" {
					continue
				}
				libpath := findshlib(ctxt, dep)
				if libpath != "" {
					addshlib(libpath)
				}
			}
		}
	}

	// clang, unlike GCC, passes -rdynamic to the linker
	// even when linking with -static, causing a linker
	// error when using GNU ld. So take out -rdynamic if
	// we added it. We do it in this order, rather than
	// only adding -rdynamic later, so that -extldflags
	// can override -rdynamic without using -static.
	// Similarly for -Wl,--dynamic-linker.
	checkStatic := func(arg string) {
		if ctxt.IsELF && arg == "-static" {
			for i := range argv {
				if argv[i] == "-rdynamic" || strings.HasPrefix(argv[i], "-Wl,--dynamic-linker,") {
					argv[i] = "-static"
				}
			}
		}
	}

	for _, p := range ldflag {
		argv = append(argv, p)
		checkStatic(p)
	}

	// When building a program with the default -buildmode=exe the
	// gc compiler generates code requires DT_TEXTREL in a
	// position independent executable (PIE). On systems where the
	// toolchain creates PIEs by default, and where DT_TEXTREL
	// does not work, the resulting programs will not run. See
	// issue #17847. To avoid this problem pass -no-pie to the
	// toolchain if it is supported.
	if ctxt.BuildMode == BuildModeExe && !ctxt.linkShared && !(ctxt.IsDarwin() && ctxt.IsARM64()) {
		// GCC uses -no-pie, clang uses -nopie.
		for _, nopie := range []string{"-no-pie", "-nopie"} {
			if linkerFlagSupported(ctxt.Arch, argv[0], altLinker, nopie) {
				argv = append(argv, nopie)
				break
			}
		}
	}

	for _, p := range strings.Fields(*flagExtldflags) {
		argv = append(argv, p)
		checkStatic(p)
	}
	if ctxt.HeadType == objabi.Hwindows {
		// Determine which linker we're using. Add in the extldflags in
		// case used has specified "-fuse-ld=...".
		cmd := exec.Command(*flagExtld, *flagExtldflags, "-Wl,--version")
		usingLLD := false
		if out, err := cmd.CombinedOutput(); err == nil {
			if bytes.Contains(out, []byte("LLD ")) {
				usingLLD = true
			}
		}

		// use gcc linker script to work around gcc bug
		// (see https://golang.org/issue/20183 for details).
		if !usingLLD {
			p := writeGDBLinkerScript()
			argv = append(argv, "-Wl,-T,"+p)
		}
		// libmingw32 and libmingwex have some inter-dependencies,
		// so must use linker groups.
		argv = append(argv, "-Wl,--start-group", "-lmingwex", "-lmingw32", "-Wl,--end-group")
		argv = append(argv, peimporteddlls()...)
	}

	if ctxt.Debugvlog != 0 {
		ctxt.Logf("host link:")
		for _, v := range argv {
			ctxt.Logf(" %q", v)
		}
		ctxt.Logf("\n")
	}

	out, err := exec.Command(argv[0], argv[1:]...).CombinedOutput()
	if err != nil {
		Exitf("running %s failed: %v\n%s", argv[0], err, out)
	}

	// Filter out useless linker warnings caused by bugs outside Go.
	// See also cmd/go/internal/work/exec.go's gccld method.
	var save [][]byte
	var skipLines int
	for _, line := range bytes.SplitAfter(out, []byte("\n")) {
		// golang.org/issue/26073 - Apple Xcode bug
		if bytes.Contains(line, []byte("ld: warning: text-based stub file")) {
			continue
		}

		if skipLines > 0 {
			skipLines--
			continue
		}

		// Remove TOC overflow warning on AIX.
		if bytes.Contains(line, []byte("ld: 0711-783")) {
			skipLines = 2
			continue
		}

		save = append(save, line)
	}
	out = bytes.Join(save, nil)

	if len(out) > 0 {
		// always print external output even if the command is successful, so that we don't
		// swallow linker warnings (see https://golang.org/issue/17935).
		ctxt.Logf("%s", out)
	}

	if combineDwarf {
		dsym := filepath.Join(*flagTmpdir, "go.dwarf")
		if out, err := exec.Command("xcrun", "dsymutil", "-f", *flagOutfile, "-o", dsym).CombinedOutput(); err != nil {
			Exitf("%s: running dsymutil failed: %v\n%s", os.Args[0], err, out)
		}
		// Remove STAB (symbolic debugging) symbols after we are done with them (by dsymutil).
		// They contain temporary file paths and make the build not reproducible.
		if out, err := exec.Command("xcrun", "strip", "-S", *flagOutfile).CombinedOutput(); err != nil {
			Exitf("%s: running strip failed: %v\n%s", os.Args[0], err, out)
		}
		// Skip combining if `dsymutil` didn't generate a file. See #11994.
		if _, err := os.Stat(dsym); os.IsNotExist(err) {
			return
		}
		// For os.Rename to work reliably, must be in same directory as outfile.
		combinedOutput := *flagOutfile + "~"
		exef, err := os.Open(*flagOutfile)
		if err != nil {
			Exitf("%s: combining dwarf failed: %v", os.Args[0], err)
		}
		defer exef.Close()
		exem, err := macho.NewFile(exef)
		if err != nil {
			Exitf("%s: parsing Mach-O header failed: %v", os.Args[0], err)
		}
		if err := machoCombineDwarf(ctxt, exef, exem, dsym, combinedOutput); err != nil {
			Exitf("%s: combining dwarf failed: %v", os.Args[0], err)
		}
		os.Remove(*flagOutfile)
		if err := os.Rename(combinedOutput, *flagOutfile); err != nil {
			Exitf("%s: %v", os.Args[0], err)
		}
	}
	if ctxt.NeedCodeSign() {
		err := machoCodeSign(ctxt, *flagOutfile)
		if err != nil {
			Exitf("%s: code signing failed: %v", os.Args[0], err)
		}
	}
}

var createTrivialCOnce sync.Once

func linkerFlagSupported(arch *sys.Arch, linker, altLinker, flag string) bool {
	createTrivialCOnce.Do(func() {
		src := filepath.Join(*flagTmpdir, "trivial.c")
		if err := ioutil.WriteFile(src, []byte("int main() { return 0; }"), 0666); err != nil {
			Errorf(nil, "WriteFile trivial.c failed: %v", err)
		}
	})

	flagsWithNextArgSkip := []string{
		"-F",
		"-l",
		"-L",
		"-framework",
		"-Wl,-framework",
		"-Wl,-rpath",
		"-Wl,-undefined",
	}
	flagsWithNextArgKeep := []string{
		"-arch",
		"-isysroot",
		"--sysroot",
		"-target",
	}
	prefixesToKeep := []string{
		"-f",
		"-m",
		"-p",
		"-Wl,",
		"-arch",
		"-isysroot",
		"--sysroot",
		"-target",
	}

	flags := hostlinkArchArgs(arch)
	keep := false
	skip := false
	extldflags := strings.Fields(*flagExtldflags)
	for _, f := range append(extldflags, ldflag...) {
		if keep {
			flags = append(flags, f)
			keep = false
		} else if skip {
			skip = false
		} else if f == "" || f[0] != '-' {
		} else if contains(flagsWithNextArgSkip, f) {
			skip = true
		} else if contains(flagsWithNextArgKeep, f) {
			flags = append(flags, f)
			keep = true
		} else {
			for _, p := range prefixesToKeep {
				if strings.HasPrefix(f, p) {
					flags = append(flags, f)
					break
				}
			}
		}
	}

	if altLinker != "" {
		flags = append(flags, "-fuse-ld="+altLinker)
	}
	flags = append(flags, flag, "trivial.c")

	cmd := exec.Command(linker, flags...)
	cmd.Dir = *flagTmpdir
	cmd.Env = append([]string{"LC_ALL=C"}, os.Environ()...)
	out, err := cmd.CombinedOutput()
	// GCC says "unrecognized command line option ‘-no-pie’"
	// clang says "unknown argument: '-no-pie'"
	return err == nil && !bytes.Contains(out, []byte("unrecognized")) && !bytes.Contains(out, []byte("unknown"))
}

// hostlinkArchArgs returns arguments to pass to the external linker
// based on the architecture.
func hostlinkArchArgs(arch *sys.Arch) []string {
	switch arch.Family {
	case sys.I386:
		return []string{"-m32"}
	case sys.AMD64:
		if buildcfg.GOOS == "darwin" {
			return []string{"-arch", "x86_64", "-m64"}
		}
		return []string{"-m64"}
	case sys.S390X:
		return []string{"-m64"}
	case sys.ARM:
		return []string{"-marm"}
	case sys.ARM64:
		if buildcfg.GOOS == "darwin" {
			return []string{"-arch", "arm64"}
		}
	case sys.MIPS64:
		return []string{"-mabi=64"}
	case sys.MIPS:
		return []string{"-mabi=32"}
	case sys.PPC64:
		if buildcfg.GOOS == "aix" {
			return []string{"-maix64"}
		} else {
			return []string{"-m64"}
		}

	}
	return nil
}

var wantHdr = objabi.HeaderString()

// ldobj loads an input object. If it is a host object (an object
// compiled by a non-Go compiler) it returns the Hostobj pointer. If
// it is a Go object, it returns nil.
func ldobj(ctxt *Link, f *bio.Reader, lib *sym.Library, length int64, pn string, file string) *Hostobj {
	pkg := objabi.PathToPrefix(lib.Pkg)

	eof := f.Offset() + length
	start := f.Offset()
	c1 := bgetc(f)
	c2 := bgetc(f)
	c3 := bgetc(f)
	c4 := bgetc(f)
	f.MustSeek(start, 0)

	unit := &sym.CompilationUnit{Lib: lib}
	lib.Units = append(lib.Units, unit)

	magic := uint32(c1)<<24 | uint32(c2)<<16 | uint32(c3)<<8 | uint32(c4)
	if magic == 0x7f454c46 { // \x7F E L F
		ldelf := func(ctxt *Link, f *bio.Reader, pkg string, length int64, pn string) {
			textp, flags, err := loadelf.Load(ctxt.loader, ctxt.Arch, ctxt.IncVersion(), f, pkg, length, pn, ehdr.Flags)
			if err != nil {
				Errorf(nil, "%v", err)
				return
			}
			ehdr.Flags = flags
			ctxt.Textp = append(ctxt.Textp, textp...)
		}
		return ldhostobj(ldelf, ctxt.HeadType, f, pkg, length, pn, file)
	}

	if magic&^1 == 0xfeedface || magic&^0x01000000 == 0xcefaedfe {
		ldmacho := func(ctxt *Link, f *bio.Reader, pkg string, length int64, pn string) {
			textp, err := loadmacho.Load(ctxt.loader, ctxt.Arch, ctxt.IncVersion(), f, pkg, length, pn)
			if err != nil {
				Errorf(nil, "%v", err)
				return
			}
			ctxt.Textp = append(ctxt.Textp, textp...)
		}
		return ldhostobj(ldmacho, ctxt.HeadType, f, pkg, length, pn, file)
	}

	switch c1<<8 | c2 {
	case 0x4c01, // 386
		0x6486, // amd64
		0xc401, // arm
		0x64aa: // arm64
		ldpe := func(ctxt *Link, f *bio.Reader, pkg string, length int64, pn string) {
			textp, rsrc, err := loadpe.Load(ctxt.loader, ctxt.Arch, ctxt.IncVersion(), f, pkg, length, pn)
			if err != nil {
				Errorf(nil, "%v", err)
				return
			}
			if len(rsrc) != 0 {
				setpersrc(ctxt, rsrc)
			}
			ctxt.Textp = append(ctxt.Textp, textp...)
		}
		return ldhostobj(ldpe, ctxt.HeadType, f, pkg, length, pn, file)
	}

	if c1 == 0x01 && (c2 == 0xD7 || c2 == 0xF7) {
		ldxcoff := func(ctxt *Link, f *bio.Reader, pkg string, length int64, pn string) {
			textp, err := loadxcoff.Load(ctxt.loader, ctxt.Arch, ctxt.IncVersion(), f, pkg, length, pn)
			if err != nil {
				Errorf(nil, "%v", err)
				return
			}
			ctxt.Textp = append(ctxt.Textp, textp...)
		}
		return ldhostobj(ldxcoff, ctxt.HeadType, f, pkg, length, pn, file)
	}

	if c1 != 'g' || c2 != 'o' || c3 != ' ' || c4 != 'o' {
		// An unrecognized object is just passed to the external linker.
		// If we try to read symbols from this object, we will
		// report an error at that time.
		unknownObjFormat = true
		return ldhostobj(nil, ctxt.HeadType, f, pkg, length, pn, file)
	}

	/* check the header */
	line, err := f.ReadString('\n')
	if err != nil {
		Errorf(nil, "truncated object file: %s: %v", pn, err)
		return nil
	}

	if !strings.HasPrefix(line, "go object ") {
		if strings.HasSuffix(pn, ".go") {
			Exitf("%s: uncompiled .go source file", pn)
			return nil
		}

		if line == ctxt.Arch.Name {
			// old header format: just $GOOS
			Errorf(nil, "%s: stale object file", pn)
			return nil
		}

		Errorf(nil, "%s: not an object file: @%d %q", pn, start, line)
		return nil
	}

	// First, check that the basic GOOS, GOARCH, and Version match.
	if line != wantHdr {
		Errorf(nil, "%s: linked object header mismatch:\nhave %q\nwant %q\n", pn, line, wantHdr)
	}

	// Skip over exports and other info -- ends with \n!\n.
	//
	// Note: It's possible for "\n!\n" to appear within the binary
	// package export data format. To avoid truncating the package
	// definition prematurely (issue 21703), we keep track of
	// how many "$$" delimiters we've seen.

	import0 := f.Offset()

	c1 = '\n' // the last line ended in \n
	c2 = bgetc(f)
	c3 = bgetc(f)
	markers := 0
	for {
		if c1 == '\n' {
			if markers%2 == 0 && c2 == '!' && c3 == '\n' {
				break
			}
			if c2 == '$' && c3 == '$' {
				markers++
			}
		}

		c1 = c2
		c2 = c3
		c3 = bgetc(f)
		if c3 == -1 {
			Errorf(nil, "truncated object file: %s", pn)
			return nil
		}
	}

	import1 := f.Offset()

	f.MustSeek(import0, 0)
	ldpkg(ctxt, f, lib, import1-import0-2, pn) // -2 for !\n
	f.MustSeek(import1, 0)

	fingerprint := ctxt.loader.Preload(ctxt.IncVersion(), f, lib, unit, eof-f.Offset())
	if !fingerprint.IsZero() { // Assembly objects don't have fingerprints. Ignore them.
		// Check fingerprint, to ensure the importing and imported packages
		// have consistent view of symbol indices.
		// Normally the go command should ensure this. But in case something
		// goes wrong, it could lead to obscure bugs like run-time crash.
		// Check it here to be sure.
		if lib.Fingerprint.IsZero() { // Not yet imported. Update its fingerprint.
			lib.Fingerprint = fingerprint
		}
		checkFingerprint(lib, fingerprint, lib.Srcref, lib.Fingerprint)
	}

	addImports(ctxt, lib, pn)
	return nil
}

func checkFingerprint(lib *sym.Library, libfp goobj.FingerprintType, src string, srcfp goobj.FingerprintType) {
	if libfp != srcfp {
		Exitf("fingerprint mismatch: %s has %x, import from %s expecting %x", lib, libfp, src, srcfp)
	}
}

func readelfsymboldata(ctxt *Link, f *elf.File, sym *elf.Symbol) []byte {
	data := make([]byte, sym.Size)
	sect := f.Sections[sym.Section]
	if sect.Type != elf.SHT_PROGBITS && sect.Type != elf.SHT_NOTE {
		Errorf(nil, "reading %s from non-data section", sym.Name)
	}
	n, err := sect.ReadAt(data, int64(sym.Value-sect.Addr))
	if uint64(n) != sym.Size {
		Errorf(nil, "reading contents of %s: %v", sym.Name, err)
	}
	return data
}

func readwithpad(r io.Reader, sz int32) ([]byte, error) {
	data := make([]byte, Rnd(int64(sz), 4))
	_, err := io.ReadFull(r, data)
	if err != nil {
		return nil, err
	}
	data = data[:sz]
	return data, nil
}

func readnote(f *elf.File, name []byte, typ int32) ([]byte, error) {
	for _, sect := range f.Sections {
		if sect.Type != elf.SHT_NOTE {
			continue
		}
		r := sect.Open()
		for {
			var namesize, descsize, noteType int32
			err := binary.Read(r, f.ByteOrder, &namesize)
			if err != nil {
				if err == io.EOF {
					break
				}
				return nil, fmt.Errorf("read namesize failed: %v", err)
			}
			err = binary.Read(r, f.ByteOrder, &descsize)
			if err != nil {
				return nil, fmt.Errorf("read descsize failed: %v", err)
			}
			err = binary.Read(r, f.ByteOrder, &noteType)
			if err != nil {
				return nil, fmt.Errorf("read type failed: %v", err)
			}
			noteName, err := readwithpad(r, namesize)
			if err != nil {
				return nil, fmt.Errorf("read name failed: %v", err)
			}
			desc, err := readwithpad(r, descsize)
			if err != nil {
				return nil, fmt.Errorf("read desc failed: %v", err)
			}
			if string(name) == string(noteName) && typ == noteType {
				return desc, nil
			}
		}
	}
	return nil, nil
}

func findshlib(ctxt *Link, shlib string) string {
	if filepath.IsAbs(shlib) {
		return shlib
	}
	for _, libdir := range ctxt.Libdir {
		libpath := filepath.Join(libdir, shlib)
		if _, err := os.Stat(libpath); err == nil {
			return libpath
		}
	}
	Errorf(nil, "cannot find shared library: %s", shlib)
	return ""
}

func ldshlibsyms(ctxt *Link, shlib string) {
	var libpath string
	if filepath.IsAbs(shlib) {
		libpath = shlib
		shlib = filepath.Base(shlib)
	} else {
		libpath = findshlib(ctxt, shlib)
		if libpath == "" {
			return
		}
	}
	for _, processedlib := range ctxt.Shlibs {
		if processedlib.Path == libpath {
			return
		}
	}
	if ctxt.Debugvlog > 1 {
		ctxt.Logf("ldshlibsyms: found library with name %s at %s\n", shlib, libpath)
	}

	f, err := elf.Open(libpath)
	if err != nil {
		Errorf(nil, "cannot open shared library: %s", libpath)
		return
	}
	// Keep the file open as decodetypeGcprog needs to read from it.
	// TODO: fix. Maybe mmap the file.
	//defer f.Close()

	hash, err := readnote(f, ELF_NOTE_GO_NAME, ELF_NOTE_GOABIHASH_TAG)
	if err != nil {
		Errorf(nil, "cannot read ABI hash from shared library %s: %v", libpath, err)
		return
	}

	depsbytes, err := readnote(f, ELF_NOTE_GO_NAME, ELF_NOTE_GODEPS_TAG)
	if err != nil {
		Errorf(nil, "cannot read dep list from shared library %s: %v", libpath, err)
		return
	}
	var deps []string
	for _, dep := range strings.Split(string(depsbytes), "\n") {
		if dep == "" {
			continue
		}
		if !filepath.IsAbs(dep) {
			// If the dep can be interpreted as a path relative to the shlib
			// in which it was found, do that. Otherwise, we will leave it
			// to be resolved by libdir lookup.
			abs := filepath.Join(filepath.Dir(libpath), dep)
			if _, err := os.Stat(abs); err == nil {
				dep = abs
			}
		}
		deps = append(deps, dep)
	}

	syms, err := f.DynamicSymbols()
	if err != nil {
		Errorf(nil, "cannot read symbols from shared library: %s", libpath)
		return
	}

	for _, elfsym := range syms {
		if elf.ST_TYPE(elfsym.Info) == elf.STT_NOTYPE || elf.ST_TYPE(elfsym.Info) == elf.STT_SECTION {
			continue
		}

		// Symbols whose names start with "type." are compiler
		// generated, so make functions with that prefix internal.
		ver := 0
		symname := elfsym.Name // (unmangled) symbol name
		if elf.ST_TYPE(elfsym.Info) == elf.STT_FUNC && strings.HasPrefix(elfsym.Name, "type.") {
			ver = sym.SymVerABIInternal
		} else if buildcfg.Experiment.RegabiWrappers && elf.ST_TYPE(elfsym.Info) == elf.STT_FUNC {
			// Demangle the ABI name. Keep in sync with symtab.go:mangleABIName.
			if strings.HasSuffix(elfsym.Name, ".abiinternal") {
				ver = sym.SymVerABIInternal
				symname = strings.TrimSuffix(elfsym.Name, ".abiinternal")
			} else if strings.HasSuffix(elfsym.Name, ".abi0") {
				ver = 0
				symname = strings.TrimSuffix(elfsym.Name, ".abi0")
			}
		}

		l := ctxt.loader
		s := l.LookupOrCreateSym(symname, ver)

		// Because loadlib above loads all .a files before loading
		// any shared libraries, any non-dynimport symbols we find
		// that duplicate symbols already loaded should be ignored
		// (the symbols from the .a files "win").
		if l.SymType(s) != 0 && l.SymType(s) != sym.SDYNIMPORT {
			continue
		}
		su := l.MakeSymbolUpdater(s)
		su.SetType(sym.SDYNIMPORT)
		l.SetSymElfType(s, elf.ST_TYPE(elfsym.Info))
		su.SetSize(int64(elfsym.Size))
		if elfsym.Section != elf.SHN_UNDEF {
			// Set .File for the library that actually defines the symbol.
			l.SetSymPkg(s, libpath)

			// The decodetype_* functions in decodetype.go need access to
			// the type data.
			sname := l.SymName(s)
			if strings.HasPrefix(sname, "type.") && !strings.HasPrefix(sname, "type..") {
				su.SetData(readelfsymboldata(ctxt, f, &elfsym))
			}
		}

		if symname != elfsym.Name {
			l.SetSymExtname(s, elfsym.Name)
		}

		// For function symbols, if ABI wrappers are not used, we don't
		// know what ABI is available, so alias it under both ABIs.
		if !buildcfg.Experiment.RegabiWrappers && elf.ST_TYPE(elfsym.Info) == elf.STT_FUNC && ver == 0 {
			alias := ctxt.loader.LookupOrCreateSym(symname, sym.SymVerABIInternal)
			if l.SymType(alias) != 0 {
				continue
			}
			su := l.MakeSymbolUpdater(alias)
			su.SetType(sym.SABIALIAS)
			r, _ := su.AddRel(0) // type doesn't matter
			r.SetSym(s)
		}
	}
	ctxt.Shlibs = append(ctxt.Shlibs, Shlib{Path: libpath, Hash: hash, Deps: deps, File: f})
}

func addsection(ldr *loader.Loader, arch *sys.Arch, seg *sym.Segment, name string, rwx int) *sym.Section {
	sect := ldr.NewSection()
	sect.Rwx = uint8(rwx)
	sect.Name = name
	sect.Seg = seg
	sect.Align = int32(arch.PtrSize) // everything is at least pointer-aligned
	seg.Sections = append(seg.Sections, sect)
	return sect
}

type chain struct {
	sym   loader.Sym
	up    *chain
	limit int // limit on entry to sym
}

func haslinkregister(ctxt *Link) bool {
	return ctxt.FixedFrameSize() != 0
}

func callsize(ctxt *Link) int {
	if haslinkregister(ctxt) {
		return 0
	}
	return ctxt.Arch.RegSize
}

type stkChk struct {
	ldr       *loader.Loader
	ctxt      *Link
	morestack loader.Sym
	done      loader.Bitmap
}

// Walk the call tree and check that there is always enough stack space
// for the call frames, especially for a chain of nosplit functions.
func (ctxt *Link) dostkcheck() {
	ldr := ctxt.loader
	sc := stkChk{
		ldr:       ldr,
		ctxt:      ctxt,
		morestack: ldr.Lookup("runtime.morestack", 0),
		done:      loader.MakeBitmap(ldr.NSym()),
	}

	// Every splitting function ensures that there are at least StackLimit
	// bytes available below SP when the splitting prologue finishes.
	// If the splitting function calls F, then F begins execution with
	// at least StackLimit - callsize() bytes available.
	// Check that every function behaves correctly with this amount
	// of stack, following direct calls in order to piece together chains
	// of non-splitting functions.
	var ch chain
	ch.limit = objabi.StackLimit - callsize(ctxt)
	if buildcfg.GOARCH == "arm64" {
		// need extra 8 bytes below SP to save FP
		ch.limit -= 8
	}

	// Check every function, but do the nosplit functions in a first pass,
	// to make the printed failure chains as short as possible.
	for _, s := range ctxt.Textp {
		if ldr.IsNoSplit(s) {
			ch.sym = s
			sc.check(&ch, 0)
		}
	}

	for _, s := range ctxt.Textp {
		if !ldr.IsNoSplit(s) {
			ch.sym = s
			sc.check(&ch, 0)
		}
	}
}

func (sc *stkChk) check(up *chain, depth int) int {
	limit := up.limit
	s := up.sym
	ldr := sc.ldr
	ctxt := sc.ctxt

	// Don't duplicate work: only need to consider each
	// function at top of safe zone once.
	top := limit == objabi.StackLimit-callsize(ctxt)
	if top {
		if sc.done.Has(s) {
			return 0
		}
		sc.done.Set(s)
	}

	if depth > 500 {
		sc.ctxt.Errorf(s, "nosplit stack check too deep")
		sc.broke(up, 0)
		return -1
	}

	if ldr.AttrExternal(s) {
		// external function.
		// should never be called directly.
		// onlyctxt.Diagnose the direct caller.
		// TODO(mwhudson): actually think about this.
		// TODO(khr): disabled for now. Calls to external functions can only happen on the g0 stack.
		// See the trampolines in src/runtime/sys_darwin_$ARCH.go.
		//if depth == 1 && ldr.SymType(s) != sym.SXREF && !ctxt.DynlinkingGo() &&
		//	ctxt.BuildMode != BuildModeCArchive && ctxt.BuildMode != BuildModePIE && ctxt.BuildMode != BuildModeCShared && ctxt.BuildMode != BuildModePlugin {
		//	Errorf(s, "call to external function")
		//}
		return -1
	}
	info := ldr.FuncInfo(s)
	if !info.Valid() { // external function. see above.
		return -1
	}

	if limit < 0 {
		sc.broke(up, limit)
		return -1
	}

	// morestack looks like it calls functions,
	// but it switches the stack pointer first.
	if s == sc.morestack {
		return 0
	}

	var ch chain
	ch.up = up

	if !ldr.IsNoSplit(s) {
		// Ensure we have enough stack to call morestack.
		ch.limit = limit - callsize(ctxt)
		ch.sym = sc.morestack
		if sc.check(&ch, depth+1) < 0 {
			return -1
		}
		if !top {
			return 0
		}
		// Raise limit to allow frame.
		locals := info.Locals()
		limit = objabi.StackLimit + int(locals) + int(ctxt.FixedFrameSize())
	}

	// Walk through sp adjustments in function, consuming relocs.
	relocs := ldr.Relocs(s)
	var ch1 chain
	pcsp := obj.NewPCIter(uint32(ctxt.Arch.MinLC))
	ri := 0
	for pcsp.Init(ldr.Data(info.Pcsp())); !pcsp.Done; pcsp.Next() {
		// pcsp.value is in effect for [pcsp.pc, pcsp.nextpc).

		// Check stack size in effect for this span.
		if int32(limit)-pcsp.Value < 0 {
			sc.broke(up, int(int32(limit)-pcsp.Value))
			return -1
		}

		// Process calls in this span.
		for ; ri < relocs.Count(); ri++ {
			r := relocs.At(ri)
			if uint32(r.Off()) >= pcsp.NextPC {
				break
			}
			t := r.Type()
			switch {
			case t.IsDirectCall():
				ch.limit = int(int32(limit) - pcsp.Value - int32(callsize(ctxt)))
				ch.sym = r.Sym()
				if sc.check(&ch, depth+1) < 0 {
					return -1
				}

			// Indirect call. Assume it is a call to a splitting function,
			// so we have to make sure it can call morestack.
			// Arrange the data structures to report both calls, so that
			// if there is an error, stkprint shows all the steps involved.
			case t == objabi.R_CALLIND:
				ch.limit = int(int32(limit) - pcsp.Value - int32(callsize(ctxt)))
				ch.sym = 0
				ch1.limit = ch.limit - callsize(ctxt) // for morestack in called prologue
				ch1.up = &ch
				ch1.sym = sc.morestack
				if sc.check(&ch1, depth+2) < 0 {
					return -1
				}
			}
		}
	}

	return 0
}

func (sc *stkChk) broke(ch *chain, limit int) {
	sc.ctxt.Errorf(ch.sym, "nosplit stack overflow")
	sc.print(ch, limit)
}

func (sc *stkChk) print(ch *chain, limit int) {
	ldr := sc.ldr
	ctxt := sc.ctxt
	var name string
	if ch.sym != 0 {
		name = fmt.Sprintf("%s<%d>", ldr.SymName(ch.sym), ldr.SymVersion(ch.sym))
		if ldr.IsNoSplit(ch.sym) {
			name += " (nosplit)"
		}
	} else {
		name = "function pointer"
	}

	if ch.up == nil {
		// top of chain. ch.sym != 0.
		if ldr.IsNoSplit(ch.sym) {
			fmt.Printf("\t%d\tassumed on entry to %s\n", ch.limit, name)
		} else {
			fmt.Printf("\t%d\tguaranteed after split check in %s\n", ch.limit, name)
		}
	} else {
		sc.print(ch.up, ch.limit+callsize(ctxt))
		if !haslinkregister(ctxt) {
			fmt.Printf("\t%d\ton entry to %s\n", ch.limit, name)
		}
	}

	if ch.limit != limit {
		fmt.Printf("\t%d\tafter %s uses %d\n", limit, name, ch.limit-limit)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: link [options] main.o\n")
	objabi.Flagprint(os.Stderr)
	Exit(2)
}

type SymbolType int8 // TODO: after genasmsym is gone, maybe rename to plan9typeChar or something

const (
	// see also https://9p.io/magic/man2html/1/nm
	TextSym      SymbolType = 'T'
	DataSym      SymbolType = 'D'
	BSSSym       SymbolType = 'B'
	UndefinedSym SymbolType = 'U'
	TLSSym       SymbolType = 't'
	FrameSym     SymbolType = 'm'
	ParamSym     SymbolType = 'p'
	AutoSym      SymbolType = 'a'

	// Deleted auto (not a real sym, just placeholder for type)
	DeletedAutoSym = 'x'
)

// defineInternal defines a symbol used internally by the go runtime.
func (ctxt *Link) defineInternal(p string, t sym.SymKind) loader.Sym {
	s := ctxt.loader.CreateSymForUpdate(p, 0)
	s.SetType(t)
	s.SetSpecial(true)
	s.SetLocal(true)
	return s.Sym()
}

func (ctxt *Link) xdefine(p string, t sym.SymKind, v int64) loader.Sym {
	s := ctxt.defineInternal(p, t)
	ctxt.loader.SetSymValue(s, v)
	return s
}

func datoff(ldr *loader.Loader, s loader.Sym, addr int64) int64 {
	if uint64(addr) >= Segdata.Vaddr {
		return int64(uint64(addr) - Segdata.Vaddr + Segdata.Fileoff)
	}
	if uint64(addr) >= Segtext.Vaddr {
		return int64(uint64(addr) - Segtext.Vaddr + Segtext.Fileoff)
	}
	ldr.Errorf(s, "invalid datoff %#x", addr)
	return 0
}

func Entryvalue(ctxt *Link) int64 {
	a := *flagEntrySymbol
	if a[0] >= '0' && a[0] <= '9' {
		return atolwhex(a)
	}
	ldr := ctxt.loader
	s := ldr.Lookup(a, 0)
	st := ldr.SymType(s)
	if st == 0 {
		return *FlagTextAddr
	}
	if !ctxt.IsAIX() && st != sym.STEXT {
		ldr.Errorf(s, "entry not text")
	}
	return ldr.SymValue(s)
}

func (ctxt *Link) callgraph() {
	if !*FlagC {
		return
	}

	ldr := ctxt.loader
	for _, s := range ctxt.Textp {
		relocs := ldr.Relocs(s)
		for i := 0; i < relocs.Count(); i++ {
			r := relocs.At(i)
			rs := r.Sym()
			if rs == 0 {
				continue
			}
			if r.Type().IsDirectCall() && (ldr.SymType(rs) == sym.STEXT || ldr.SymType(rs) == sym.SABIALIAS) {
				ctxt.Logf("%s calls %s\n", ldr.SymName(s), ldr.SymName(rs))
			}
		}
	}
}

func Rnd(v int64, r int64) int64 {
	if r <= 0 {
		return v
	}
	v += r - 1
	c := v % r
	if c < 0 {
		c += r
	}
	v -= c
	return v
}

func bgetc(r *bio.Reader) int {
	c, err := r.ReadByte()
	if err != nil {
		if err != io.EOF {
			log.Fatalf("reading input: %v", err)
		}
		return -1
	}
	return int(c)
}

type markKind uint8 // for postorder traversal
const (
	_ markKind = iota
	visiting
	visited
)

func postorder(libs []*sym.Library) []*sym.Library {
	order := make([]*sym.Library, 0, len(libs)) // hold the result
	mark := make(map[*sym.Library]markKind, len(libs))
	for _, lib := range libs {
		dfs(lib, mark, &order)
	}
	return order
}

func dfs(lib *sym.Library, mark map[*sym.Library]markKind, order *[]*sym.Library) {
	if mark[lib] == visited {
		return
	}
	if mark[lib] == visiting {
		panic("found import cycle while visiting " + lib.Pkg)
	}
	mark[lib] = visiting
	for _, i := range lib.Imports {
		dfs(i, mark, order)
	}
	mark[lib] = visited
	*order = append(*order, lib)
}

func ElfSymForReloc(ctxt *Link, s loader.Sym) int32 {
	// If putelfsym created a local version of this symbol, use that in all
	// relocations.
	les := ctxt.loader.SymLocalElfSym(s)
	if les != 0 {
		return les
	} else {
		return ctxt.loader.SymElfSym(s)
	}
}

func AddGotSym(target *Target, ldr *loader.Loader, syms *ArchSyms, s loader.Sym, elfRelocTyp uint32) {
	if ldr.SymGot(s) >= 0 {
		return
	}

	Adddynsym(ldr, target, syms, s)
	got := ldr.MakeSymbolUpdater(syms.GOT)
	ldr.SetGot(s, int32(got.Size()))
	got.AddUint(target.Arch, 0)

	if target.IsElf() {
		if target.Arch.PtrSize == 8 {
			rela := ldr.MakeSymbolUpdater(syms.Rela)
			rela.AddAddrPlus(target.Arch, got.Sym(), int64(ldr.SymGot(s)))
			rela.AddUint64(target.Arch, elf.R_INFO(uint32(ldr.SymDynid(s)), elfRelocTyp))
			rela.AddUint64(target.Arch, 0)
		} else {
			rel := ldr.MakeSymbolUpdater(syms.Rel)
			rel.AddAddrPlus(target.Arch, got.Sym(), int64(ldr.SymGot(s)))
			rel.AddUint32(target.Arch, elf.R_INFO32(uint32(ldr.SymDynid(s)), elfRelocTyp))
		}
	} else if target.IsDarwin() {
		leg := ldr.MakeSymbolUpdater(syms.LinkEditGOT)
		leg.AddUint32(target.Arch, uint32(ldr.SymDynid(s)))
		if target.IsPIE() && target.IsInternal() {
			// Mach-O relocations are a royal pain to lay out.
			// They use a compact stateful bytecode representation.
			// Here we record what are needed and encode them later.
			MachoAddBind(int64(ldr.SymGot(s)), s)
		}
	} else {
		ldr.Errorf(s, "addgotsym: unsupported binary format")
	}
}
