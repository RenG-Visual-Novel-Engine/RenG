// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"bytes"
	"cmd/internal/sys"
	"debug/macho"
	"internal/testenv"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

var AuthorPaidByTheColumnInch struct {
	fog int `text:"London. Michaelmas term lately over, and the Lord Chancellor sitting in Lincoln’s Inn Hall. Implacable November weather. As much mud in the streets as if the waters had but newly retired from the face of the earth, and it would not be wonderful to meet a Megalosaurus, forty feet long or so, waddling like an elephantine lizard up Holborn Hill. Smoke lowering down from chimney-pots, making a soft black drizzle, with flakes of soot in it as big as full-grown snowflakes—gone into mourning, one might imagine, for the death of the sun. Dogs, undistinguishable in mire. Horses, scarcely better; splashed to their very blinkers. Foot passengers, jostling one another’s umbrellas in a general infection of ill temper, and losing their foot-hold at street-corners, where tens of thousands of other foot passengers have been slipping and sliding since the day broke (if this day ever broke), adding new deposits to the crust upon crust of mud, sticking at those points tenaciously to the pavement, and accumulating at compound interest.  	Fog everywhere. Fog up the river, where it flows among green aits and meadows; fog down the river, where it rolls defiled among the tiers of shipping and the waterside pollutions of a great (and dirty) city. Fog on the Essex marshes, fog on the Kentish heights. Fog creeping into the cabooses of collier-brigs; fog lying out on the yards and hovering in the rigging of great ships; fog drooping on the gunwales of barges and small boats. Fog in the eyes and throats of ancient Greenwich pensioners, wheezing by the firesides of their wards; fog in the stem and bowl of the afternoon pipe of the wrathful skipper, down in his close cabin; fog cruelly pinching the toes and fingers of his shivering little ‘prentice boy on deck. Chance people on the bridges peeping over the parapets into a nether sky of fog, with fog all round them, as if they were up in a balloon and hanging in the misty clouds.  	Gas looming through the fog in divers places in the streets, much as the sun may, from the spongey fields, be seen to loom by husbandman and ploughboy. Most of the shops lighted two hours before their time—as the gas seems to know, for it has a haggard and unwilling look.  	The raw afternoon is rawest, and the dense fog is densest, and the muddy streets are muddiest near that leaden-headed old obstruction, appropriate ornament for the threshold of a leaden-headed old corporation, Temple Bar. And hard by Temple Bar, in Lincoln’s Inn Hall, at the very heart of the fog, sits the Lord High Chancellor in his High Court of Chancery."`

	wind int `text:"It was grand to see how the wind awoke, and bent the trees, and drove the rain before it like a cloud of smoke; and to hear the solemn thunder, and to see the lightning; and while thinking with awe of the tremendous powers by which our little lives are encompassed, to consider how beneficent they are, and how upon the smallest flower and leaf there was already a freshness poured from all this seeming rage, which seemed to make creation new again."`

	jarndyce int `text:"Jarndyce and Jarndyce drones on. This scarecrow of a suit has, over the course of time, become so complicated, that no man alive knows what it means. The parties to it understand it least; but it has been observed that no two Chancery lawyers can talk about it for five minutes, without coming to a total disagreement as to all the premises. Innumerable children have been born into the cause; innumerable young people have married into it; innumerable old people have died out of it. Scores of persons have deliriously found themselves made parties in Jarndyce and Jarndyce, without knowing how or why; whole families have inherited legendary hatreds with the suit. The little plaintiff or defendant, who was promised a new rocking-horse when Jarndyce and Jarndyce should be settled, has grown up, possessed himself of a real horse, and trotted away into the other world. Fair wards of court have faded into mothers and grandmothers; a long procession of Chancellors has come in and gone out; the legion of bills in the suit have been transformed into mere bills of mortality; there are not three Jarndyces left upon the earth perhaps, since old Tom Jarndyce in despair blew his brains out at a coffee-house in Chancery Lane; but Jarndyce and Jarndyce still drags its dreary length before the Court, perennially hopeless."`

	principle int `text:"The one great principle of the English law is, to make business for itself. There is no other principle distinctly, certainly, and consistently maintained through all its narrow turnings. Viewed by this light it becomes a coherent scheme, and not the monstrous maze the laity are apt to think it. Let them but once clearly perceive that its grand principle is to make business for itself at their expense, and surely they will cease to grumble."`
}

func TestLargeSymName(t *testing.T) {
	// The compiler generates a symbol name using the string form of the
	// type. This tests that the linker can read symbol names larger than
	// the bufio buffer. Issue #15104.
	_ = AuthorPaidByTheColumnInch
}

func TestIssue21703(t *testing.T) {
	t.Parallel()

	testenv.MustHaveGoBuild(t)

	const source = `
package main
const X = "\n!\n"
func main() {}
`

	tmpdir := t.TempDir()

	err := ioutil.WriteFile(filepath.Join(tmpdir, "main.go"), []byte(source), 0666)
	if err != nil {
		t.Fatalf("failed to write main.go: %v\n", err)
	}

	cmd := exec.Command(testenv.GoToolPath(t), "tool", "compile", "main.go")
	cmd.Dir = tmpdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to compile main.go: %v, output: %s\n", err, out)
	}

	cmd = exec.Command(testenv.GoToolPath(t), "tool", "link", "main.o")
	cmd.Dir = tmpdir
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to link main.o: %v, output: %s\n", err, out)
	}
}

// TestIssue28429 ensures that the linker does not attempt to link
// sections not named *.o. Such sections may be used by a build system
// to, for example, save facts produced by a modular static analysis
// such as golang.org/x/tools/go/analysis.
func TestIssue28429(t *testing.T) {
	t.Parallel()

	testenv.MustHaveGoBuild(t)

	tmpdir := t.TempDir()

	write := func(name, content string) {
		err := ioutil.WriteFile(filepath.Join(tmpdir, name), []byte(content), 0666)
		if err != nil {
			t.Fatal(err)
		}
	}

	runGo := func(args ...string) {
		cmd := exec.Command(testenv.GoToolPath(t), args...)
		cmd.Dir = tmpdir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("'go %s' failed: %v, output: %s",
				strings.Join(args, " "), err, out)
		}
	}

	// Compile a main package.
	write("main.go", "package main; func main() {}")
	runGo("tool", "compile", "-p", "main", "main.go")
	runGo("tool", "pack", "c", "main.a", "main.o")

	// Add an extra section with a short, non-.o name.
	// This simulates an alternative build system.
	write(".facts", "this is not an object file")
	runGo("tool", "pack", "r", "main.a", ".facts")

	// Verify that the linker does not attempt
	// to compile the extra section.
	runGo("tool", "link", "main.a")
}

func TestUnresolved(t *testing.T) {
	testenv.MustHaveGoBuild(t)

	t.Parallel()

	tmpdir := t.TempDir()

	write := func(name, content string) {
		err := ioutil.WriteFile(filepath.Join(tmpdir, name), []byte(content), 0666)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Test various undefined references. Because of issue #29852,
	// this used to give confusing error messages because the
	// linker would find an undefined reference to "zero" created
	// by the runtime package.

	write("go.mod", "module testunresolved\n")
	write("main.go", `package main

func main() {
        x()
}

func x()
`)
	write("main.s", `
TEXT ·x(SB),0,$0
        MOVD zero<>(SB), AX
        MOVD zero(SB), AX
        MOVD ·zero(SB), AX
        RET
`)
	cmd := exec.Command(testenv.GoToolPath(t), "build")
	cmd.Dir = tmpdir
	cmd.Env = append(os.Environ(),
		"GOARCH=amd64", "GOOS=linux", "GOPATH="+filepath.Join(tmpdir, "_gopath"))
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected build to fail, but it succeeded")
	}
	out = regexp.MustCompile("(?m)^#.*\n").ReplaceAll(out, nil)
	got := string(out)
	want := `main.x: relocation target zero not defined
main.x: relocation target zero not defined
main.x: relocation target main.zero not defined
`
	if want != got {
		t.Fatalf("want:\n%sgot:\n%s", want, got)
	}
}

func TestIssue33979(t *testing.T) {
	testenv.MustHaveGoBuild(t)
	testenv.MustHaveCGO(t)
	testenv.MustInternalLink(t)

	// Skip test on platforms that do not support cgo internal linking.
	switch runtime.GOARCH {
	case "mips", "mipsle", "mips64", "mips64le":
		t.Skipf("Skipping on %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOOS == "aix" ||
		runtime.GOOS == "windows" && runtime.GOARCH == "arm64" {
		t.Skipf("Skipping on %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	t.Parallel()

	tmpdir := t.TempDir()

	write := func(name, content string) {
		err := ioutil.WriteFile(filepath.Join(tmpdir, name), []byte(content), 0666)
		if err != nil {
			t.Fatal(err)
		}
	}

	run := func(name string, args ...string) string {
		cmd := exec.Command(name, args...)
		cmd.Dir = tmpdir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("'go %s' failed: %v, output: %s", strings.Join(args, " "), err, out)
		}
		return string(out)
	}
	runGo := func(args ...string) string {
		return run(testenv.GoToolPath(t), args...)
	}

	// Test object with undefined reference that was not generated
	// by Go, resulting in an SXREF symbol being loaded during linking.
	// Because of issue #33979, the SXREF symbol would be found during
	// error reporting, resulting in confusing error messages.

	write("main.go", `package main
func main() {
        x()
}
func x()
`)
	// The following assembly must work on all architectures.
	write("x.s", `
TEXT ·x(SB),0,$0
        CALL foo(SB)
        RET
`)
	write("x.c", `
void undefined();

void foo() {
        undefined();
}
`)

	cc := strings.TrimSpace(runGo("env", "CC"))
	cflags := strings.Fields(runGo("env", "GOGCCFLAGS"))

	// Compile, assemble and pack the Go and C code.
	runGo("tool", "asm", "-gensymabis", "-o", "symabis", "x.s")
	runGo("tool", "compile", "-symabis", "symabis", "-p", "main", "-o", "x1.o", "main.go")
	runGo("tool", "asm", "-o", "x2.o", "x.s")
	run(cc, append(cflags, "-c", "-o", "x3.o", "x.c")...)
	runGo("tool", "pack", "c", "x.a", "x1.o", "x2.o", "x3.o")

	// Now attempt to link using the internal linker.
	cmd := exec.Command(testenv.GoToolPath(t), "tool", "link", "-linkmode=internal", "x.a")
	cmd.Dir = tmpdir
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected link to fail, but it succeeded")
	}
	re := regexp.MustCompile(`(?m)^main\(.*text\): relocation target undefined not defined$`)
	if !re.Match(out) {
		t.Fatalf("got:\n%q\nwant:\n%s", out, re)
	}
}

func TestBuildForTvOS(t *testing.T) {
	testenv.MustHaveCGO(t)
	testenv.MustHaveGoBuild(t)

	// Only run this on darwin/amd64, where we can cross build for tvOS.
	if runtime.GOARCH != "amd64" || runtime.GOOS != "darwin" {
		t.Skip("skipping on non-darwin/amd64 platform")
	}
	if testing.Short() && os.Getenv("GO_BUILDER_NAME") == "" {
		t.Skip("skipping in -short mode with $GO_BUILDER_NAME empty")
	}
	if err := exec.Command("xcrun", "--help").Run(); err != nil {
		t.Skipf("error running xcrun, required for iOS cross build: %v", err)
	}

	t.Parallel()

	sdkPath, err := exec.Command("xcrun", "--sdk", "appletvos", "--show-sdk-path").Output()
	if err != nil {
		t.Skip("failed to locate appletvos SDK, skipping")
	}
	CC := []string{
		"clang",
		"-arch",
		"arm64",
		"-isysroot", strings.TrimSpace(string(sdkPath)),
		"-mtvos-version-min=12.0",
		"-fembed-bitcode",
		"-framework", "CoreFoundation",
	}
	lib := filepath.Join("testdata", "testBuildFortvOS", "lib.go")
	tmpDir := t.TempDir()

	ar := filepath.Join(tmpDir, "lib.a")
	cmd := exec.Command(testenv.GoToolPath(t), "build", "-buildmode=c-archive", "-o", ar, lib)
	cmd.Env = append(os.Environ(),
		"CGO_ENABLED=1",
		"GOOS=ios",
		"GOARCH=arm64",
		"CC="+strings.Join(CC, " "),
		"CGO_CFLAGS=", // ensure CGO_CFLAGS does not contain any flags. Issue #35459
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v: %v:\n%s", cmd.Args, err, out)
	}

	link := exec.Command(CC[0], CC[1:]...)
	link.Args = append(link.Args, "-o", filepath.Join(tmpDir, "a.out")) // Avoid writing to package directory.
	link.Args = append(link.Args, ar, filepath.Join("testdata", "testBuildFortvOS", "main.m"))
	if out, err := link.CombinedOutput(); err != nil {
		t.Fatalf("%v: %v:\n%s", link.Args, err, out)
	}
}

var testXFlagSrc = `
package main
var X = "hello"
var Z = [99999]int{99998:12345} // make it large enough to be mmaped
func main() { println(X) }
`

func TestXFlag(t *testing.T) {
	testenv.MustHaveGoBuild(t)

	t.Parallel()

	tmpdir := t.TempDir()

	src := filepath.Join(tmpdir, "main.go")
	err := ioutil.WriteFile(src, []byte(testXFlagSrc), 0666)
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(testenv.GoToolPath(t), "build", "-ldflags=-X=main.X=meow", "-o", filepath.Join(tmpdir, "main"), src)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("%v: %v:\n%s", cmd.Args, err, out)
	}
}

var testMachOBuildVersionSrc = `
package main
func main() { }
`

func TestMachOBuildVersion(t *testing.T) {
	testenv.MustHaveGoBuild(t)

	t.Parallel()

	tmpdir := t.TempDir()

	src := filepath.Join(tmpdir, "main.go")
	err := ioutil.WriteFile(src, []byte(testMachOBuildVersionSrc), 0666)
	if err != nil {
		t.Fatal(err)
	}

	exe := filepath.Join(tmpdir, "main")
	cmd := exec.Command(testenv.GoToolPath(t), "build", "-ldflags=-linkmode=internal", "-o", exe, src)
	cmd.Env = append(os.Environ(),
		"CGO_ENABLED=0",
		"GOOS=darwin",
		"GOARCH=amd64",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v: %v:\n%s", cmd.Args, err, out)
	}
	exef, err := os.Open(exe)
	if err != nil {
		t.Fatal(err)
	}
	defer exef.Close()
	exem, err := macho.NewFile(exef)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	const LC_BUILD_VERSION = 0x32
	checkMin := func(ver uint32) {
		major, minor := (ver>>16)&0xff, (ver>>8)&0xff
		if major != 10 || minor < 9 {
			t.Errorf("LC_BUILD_VERSION version %d.%d < 10.9", major, minor)
		}
	}
	for _, cmd := range exem.Loads {
		raw := cmd.Raw()
		type_ := exem.ByteOrder.Uint32(raw)
		if type_ != LC_BUILD_VERSION {
			continue
		}
		osVer := exem.ByteOrder.Uint32(raw[12:])
		checkMin(osVer)
		sdkVer := exem.ByteOrder.Uint32(raw[16:])
		checkMin(sdkVer)
		found = true
		break
	}
	if !found {
		t.Errorf("no LC_BUILD_VERSION load command found")
	}
}

const Issue34788src = `

package blah

func Blah(i int) int {
	a := [...]int{1, 2, 3, 4, 5, 6, 7, 8}
	return a[i&7]
}
`

func TestIssue34788Android386TLSSequence(t *testing.T) {
	testenv.MustHaveGoBuild(t)

	// This is a cross-compilation test, so it doesn't make
	// sense to run it on every GOOS/GOARCH combination. Limit
	// the test to amd64 + darwin/linux.
	if runtime.GOARCH != "amd64" ||
		(runtime.GOOS != "darwin" && runtime.GOOS != "linux") {
		t.Skip("skipping on non-{linux,darwin}/amd64 platform")
	}

	t.Parallel()

	tmpdir := t.TempDir()

	src := filepath.Join(tmpdir, "blah.go")
	err := ioutil.WriteFile(src, []byte(Issue34788src), 0666)
	if err != nil {
		t.Fatal(err)
	}

	obj := filepath.Join(tmpdir, "blah.o")
	cmd := exec.Command(testenv.GoToolPath(t), "tool", "compile", "-o", obj, src)
	cmd.Env = append(os.Environ(), "GOARCH=386", "GOOS=android")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to compile blah.go: %v, output: %s\n", err, out)
	}

	// Run objdump on the resulting object.
	cmd = exec.Command(testenv.GoToolPath(t), "tool", "objdump", obj)
	out, oerr := cmd.CombinedOutput()
	if oerr != nil {
		t.Fatalf("failed to objdump blah.o: %v, output: %s\n", oerr, out)
	}

	// Sift through the output; we should not be seeing any R_TLS_LE relocs.
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "R_TLS_LE") {
			t.Errorf("objdump output contains unexpected R_TLS_LE reloc: %s", line)
		}
	}
}

const testStrictDupGoSrc = `
package main
func f()
func main() { f() }
`

const testStrictDupAsmSrc1 = `
#include "textflag.h"
TEXT	·f(SB), NOSPLIT|DUPOK, $0-0
	RET
`

const testStrictDupAsmSrc2 = `
#include "textflag.h"
TEXT	·f(SB), NOSPLIT|DUPOK, $0-0
	JMP	0(PC)
`

const testStrictDupAsmSrc3 = `
#include "textflag.h"
GLOBL ·rcon(SB), RODATA|DUPOK, $64
`

const testStrictDupAsmSrc4 = `
#include "textflag.h"
GLOBL ·rcon(SB), RODATA|DUPOK, $32
`

func TestStrictDup(t *testing.T) {
	// Check that -strictdups flag works.
	testenv.MustHaveGoBuild(t)

	asmfiles := []struct {
		fname   string
		payload string
	}{
		{"a", testStrictDupAsmSrc1},
		{"b", testStrictDupAsmSrc2},
		{"c", testStrictDupAsmSrc3},
		{"d", testStrictDupAsmSrc4},
	}

	t.Parallel()

	tmpdir := t.TempDir()

	src := filepath.Join(tmpdir, "x.go")
	err := ioutil.WriteFile(src, []byte(testStrictDupGoSrc), 0666)
	if err != nil {
		t.Fatal(err)
	}
	for _, af := range asmfiles {
		src = filepath.Join(tmpdir, af.fname+".s")
		err = ioutil.WriteFile(src, []byte(af.payload), 0666)
		if err != nil {
			t.Fatal(err)
		}
	}
	src = filepath.Join(tmpdir, "go.mod")
	err = ioutil.WriteFile(src, []byte("module teststrictdup\n"), 0666)
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(testenv.GoToolPath(t), "build", "-ldflags=-strictdups=1")
	cmd.Dir = tmpdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("linking with -strictdups=1 failed: %v\n%s", err, string(out))
	}
	if !bytes.Contains(out, []byte("mismatched payload")) {
		t.Errorf("unexpected output:\n%s", out)
	}

	cmd = exec.Command(testenv.GoToolPath(t), "build", "-ldflags=-strictdups=2")
	cmd.Dir = tmpdir
	out, err = cmd.CombinedOutput()
	if err == nil {
		t.Errorf("linking with -strictdups=2 did not fail")
	}
	// NB: on amd64 we get the 'new length' error, on arm64 the 'different
	// contents' error.
	if !(bytes.Contains(out, []byte("mismatched payload: new length")) ||
		bytes.Contains(out, []byte("mismatched payload: same length but different contents"))) ||
		!bytes.Contains(out, []byte("mismatched payload: different sizes")) {
		t.Errorf("unexpected output:\n%s", out)
	}
}

const testFuncAlignSrc = `
package main
import (
	"fmt"
	"reflect"
)
func alignPc()

func main() {
	addr := reflect.ValueOf(alignPc).Pointer()
	if (addr % 512) != 0 {
		fmt.Printf("expected 512 bytes alignment, got %v\n", addr)
	} else {
		fmt.Printf("PASS")
	}
}
`

const testFuncAlignAsmSrc = `
#include "textflag.h"

TEXT	·alignPc(SB),NOSPLIT, $0-0
	MOVD	$2, R0
	PCALIGN	$512
	MOVD	$3, R1
	RET
`

// TestFuncAlign verifies that the address of a function can be aligned
// with a specific value on arm64.
func TestFuncAlign(t *testing.T) {
	if runtime.GOARCH != "arm64" || runtime.GOOS != "linux" {
		t.Skip("skipping on non-linux/arm64 platform")
	}
	testenv.MustHaveGoBuild(t)

	t.Parallel()

	tmpdir := t.TempDir()

	src := filepath.Join(tmpdir, "go.mod")
	err := ioutil.WriteFile(src, []byte("module cmd/link/TestFuncAlign/falign"), 0666)
	if err != nil {
		t.Fatal(err)
	}
	src = filepath.Join(tmpdir, "falign.go")
	err = ioutil.WriteFile(src, []byte(testFuncAlignSrc), 0666)
	if err != nil {
		t.Fatal(err)
	}
	src = filepath.Join(tmpdir, "falign.s")
	err = ioutil.WriteFile(src, []byte(testFuncAlignAsmSrc), 0666)
	if err != nil {
		t.Fatal(err)
	}

	// Build and run with old object file format.
	cmd := exec.Command(testenv.GoToolPath(t), "build", "-o", "falign")
	cmd.Dir = tmpdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("build failed: %v", err)
	}
	cmd = exec.Command(tmpdir + "/falign")
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Errorf("failed to run with err %v, output: %s", err, out)
	}
	if string(out) != "PASS" {
		t.Errorf("unexpected output: %s\n", out)
	}
}

const testTrampSrc = `
package main
import "fmt"
func main() {
	fmt.Println("hello")

	defer func(){
		if e := recover(); e == nil {
			panic("did not panic")
		}
	}()
	f1()
}

// Test deferreturn trampolines. See issue #39049.
func f1() { defer f2() }
func f2() { panic("XXX") }
`

func TestTrampoline(t *testing.T) {
	// Test that trampoline insertion works as expected.
	// For stress test, we set -debugtramp=2 flag, which sets a very low
	// threshold for trampoline generation, and essentially all cross-package
	// calls will use trampolines.
	switch runtime.GOARCH {
	case "arm", "arm64", "ppc64", "ppc64le":
	default:
		t.Skipf("trampoline insertion is not implemented on %s", runtime.GOARCH)
	}

	testenv.MustHaveGoBuild(t)

	t.Parallel()

	tmpdir := t.TempDir()

	src := filepath.Join(tmpdir, "hello.go")
	err := ioutil.WriteFile(src, []byte(testTrampSrc), 0666)
	if err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(tmpdir, "hello.exe")

	cmd := exec.Command(testenv.GoToolPath(t), "build", "-ldflags=-debugtramp=2", "-o", exe, src)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	cmd = exec.Command(exe)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Errorf("executable failed to run: %v\n%s", err, out)
	}
	if string(out) != "hello\n" {
		t.Errorf("unexpected output:\n%s", out)
	}
}

const testTrampCgoSrc = `
package main

// #include <stdio.h>
// void CHello() { printf("hello\n"); fflush(stdout); }
import "C"

func main() {
	C.CHello()
}
`

func TestTrampolineCgo(t *testing.T) {
	// Test that trampoline insertion works for cgo code.
	// For stress test, we set -debugtramp=2 flag, which sets a very low
	// threshold for trampoline generation, and essentially all cross-package
	// calls will use trampolines.
	switch runtime.GOARCH {
	case "arm", "arm64", "ppc64", "ppc64le":
	default:
		t.Skipf("trampoline insertion is not implemented on %s", runtime.GOARCH)
	}

	testenv.MustHaveGoBuild(t)
	testenv.MustHaveCGO(t)

	t.Parallel()

	tmpdir := t.TempDir()

	src := filepath.Join(tmpdir, "hello.go")
	err := ioutil.WriteFile(src, []byte(testTrampCgoSrc), 0666)
	if err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(tmpdir, "hello.exe")

	cmd := exec.Command(testenv.GoToolPath(t), "build", "-ldflags=-debugtramp=2", "-o", exe, src)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	cmd = exec.Command(exe)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Errorf("executable failed to run: %v\n%s", err, out)
	}
	if string(out) != "hello\n" && string(out) != "hello\r\n" {
		t.Errorf("unexpected output:\n%s", out)
	}

	// Test internal linking mode.

	if runtime.GOARCH == "ppc64" || runtime.GOARCH == "ppc64le" || (runtime.GOARCH == "arm64" && runtime.GOOS == "windows") || !testenv.CanInternalLink() {
		return // internal linking cgo is not supported
	}
	cmd = exec.Command(testenv.GoToolPath(t), "build", "-ldflags=-debugtramp=2 -linkmode=internal", "-o", exe, src)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	cmd = exec.Command(exe)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Errorf("executable failed to run: %v\n%s", err, out)
	}
	if string(out) != "hello\n" && string(out) != "hello\r\n" {
		t.Errorf("unexpected output:\n%s", out)
	}
}

func TestIndexMismatch(t *testing.T) {
	// Test that index mismatch will cause a link-time error (not run-time error).
	// This shouldn't happen with "go build". We invoke the compiler and the linker
	// manually, and try to "trick" the linker with an inconsistent object file.
	testenv.MustHaveGoBuild(t)

	t.Parallel()

	tmpdir := t.TempDir()

	aSrc := filepath.Join("testdata", "testIndexMismatch", "a.go")
	bSrc := filepath.Join("testdata", "testIndexMismatch", "b.go")
	mSrc := filepath.Join("testdata", "testIndexMismatch", "main.go")
	aObj := filepath.Join(tmpdir, "a.o")
	mObj := filepath.Join(tmpdir, "main.o")
	exe := filepath.Join(tmpdir, "main.exe")

	// Build a program with main package importing package a.
	cmd := exec.Command(testenv.GoToolPath(t), "tool", "compile", "-o", aObj, aSrc)
	t.Log(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiling a.go failed: %v\n%s", err, out)
	}
	cmd = exec.Command(testenv.GoToolPath(t), "tool", "compile", "-I", tmpdir, "-o", mObj, mSrc)
	t.Log(cmd)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiling main.go failed: %v\n%s", err, out)
	}
	cmd = exec.Command(testenv.GoToolPath(t), "tool", "link", "-L", tmpdir, "-o", exe, mObj)
	t.Log(cmd)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Errorf("linking failed: %v\n%s", err, out)
	}

	// Now, overwrite a.o with the object of b.go. This should
	// result in an index mismatch.
	cmd = exec.Command(testenv.GoToolPath(t), "tool", "compile", "-o", aObj, bSrc)
	t.Log(cmd)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiling a.go failed: %v\n%s", err, out)
	}
	cmd = exec.Command(testenv.GoToolPath(t), "tool", "link", "-L", tmpdir, "-o", exe, mObj)
	t.Log(cmd)
	out, err = cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("linking didn't fail")
	}
	if !bytes.Contains(out, []byte("fingerprint mismatch")) {
		t.Errorf("did not see expected error message. out:\n%s", out)
	}
}

func TestPErsrcBinutils(t *testing.T) {
	// Test that PE rsrc section is handled correctly (issue 39658).
	testenv.MustHaveGoBuild(t)

	if (runtime.GOARCH != "386" && runtime.GOARCH != "amd64") || runtime.GOOS != "windows" {
		// This test is limited to amd64 and 386, because binutils is limited as such
		t.Skipf("this is only for windows/amd64 and windows/386")
	}

	t.Parallel()

	tmpdir := t.TempDir()

	pkgdir := filepath.Join("testdata", "pe-binutils")
	exe := filepath.Join(tmpdir, "a.exe")
	cmd := exec.Command(testenv.GoToolPath(t), "build", "-o", exe)
	cmd.Dir = pkgdir
	// cmd.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64") // uncomment if debugging in a cross-compiling environment
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("building failed: %v, output:\n%s", err, out)
	}

	// Check that the binary contains the rsrc data
	b, err := ioutil.ReadFile(exe)
	if err != nil {
		t.Fatalf("reading output failed: %v", err)
	}
	if !bytes.Contains(b, []byte("Hello Gophers!")) {
		t.Fatalf("binary does not contain expected content")
	}
}

func TestPErsrcLLVM(t *testing.T) {
	// Test that PE rsrc section is handled correctly (issue 39658).
	testenv.MustHaveGoBuild(t)

	if runtime.GOOS != "windows" {
		t.Skipf("this is a windows-only test")
	}

	t.Parallel()

	tmpdir := t.TempDir()

	pkgdir := filepath.Join("testdata", "pe-llvm")
	exe := filepath.Join(tmpdir, "a.exe")
	cmd := exec.Command(testenv.GoToolPath(t), "build", "-o", exe)
	cmd.Dir = pkgdir
	// cmd.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64") // uncomment if debugging in a cross-compiling environment
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("building failed: %v, output:\n%s", err, out)
	}

	// Check that the binary contains the rsrc data
	b, err := ioutil.ReadFile(exe)
	if err != nil {
		t.Fatalf("reading output failed: %v", err)
	}
	if !bytes.Contains(b, []byte("resname RCDATA a.rc")) {
		t.Fatalf("binary does not contain expected content")
	}
}

func TestContentAddressableSymbols(t *testing.T) {
	// Test that the linker handles content-addressable symbols correctly.
	testenv.MustHaveGoBuild(t)

	t.Parallel()

	src := filepath.Join("testdata", "testHashedSyms", "p.go")
	cmd := exec.Command(testenv.GoToolPath(t), "run", src)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("command %s failed: %v\n%s", cmd, err, out)
	}
}

func TestReadOnly(t *testing.T) {
	// Test that read-only data is indeed read-only.
	testenv.MustHaveGoBuild(t)

	t.Parallel()

	src := filepath.Join("testdata", "testRO", "x.go")
	cmd := exec.Command(testenv.GoToolPath(t), "run", src)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Errorf("running test program did not fail. output:\n%s", out)
	}
}

const testIssue38554Src = `
package main

type T [10<<20]byte

//go:noinline
func f() T {
	return T{} // compiler will make a large stmp symbol, but not used.
}

func main() {
	x := f()
	println(x[1])
}
`

func TestIssue38554(t *testing.T) {
	testenv.MustHaveGoBuild(t)

	t.Parallel()

	tmpdir := t.TempDir()

	src := filepath.Join(tmpdir, "x.go")
	err := ioutil.WriteFile(src, []byte(testIssue38554Src), 0666)
	if err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}
	exe := filepath.Join(tmpdir, "x.exe")
	cmd := exec.Command(testenv.GoToolPath(t), "build", "-o", exe, src)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}

	fi, err := os.Stat(exe)
	if err != nil {
		t.Fatalf("failed to stat output file: %v", err)
	}

	// The test program is not much different from a helloworld, which is
	// typically a little over 1 MB. We allow 5 MB. If the bad stmp is live,
	// it will be over 10 MB.
	const want = 5 << 20
	if got := fi.Size(); got > want {
		t.Errorf("binary too big: got %d, want < %d", got, want)
	}
}

const testIssue42396src = `
package main

//go:noinline
//go:nosplit
func callee(x int) {
}

func main() {
	callee(9)
}
`

func TestIssue42396(t *testing.T) {
	testenv.MustHaveGoBuild(t)

	if !sys.RaceDetectorSupported(runtime.GOOS, runtime.GOARCH) {
		t.Skip("no race detector support")
	}

	t.Parallel()

	tmpdir := t.TempDir()

	src := filepath.Join(tmpdir, "main.go")
	err := ioutil.WriteFile(src, []byte(testIssue42396src), 0666)
	if err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}
	exe := filepath.Join(tmpdir, "main.exe")
	cmd := exec.Command(testenv.GoToolPath(t), "build", "-gcflags=-race", "-o", exe, src)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("build unexpectedly succeeded")
	}

	// Check to make sure that we see a reasonable error message
	// and not a panic.
	if strings.Contains(string(out), "panic:") {
		t.Fatalf("build should not fail with panic:\n%s", out)
	}
	const want = "reference to undefined builtin"
	if !strings.Contains(string(out), want) {
		t.Fatalf("error message incorrect: expected it to contain %q but instead got:\n%s\n", want, out)
	}
}

const testLargeRelocSrc = `
package main

var x = [1<<25]byte{1<<23: 23, 1<<24: 24}

func main() {
	check(x[1<<23-1], 0)
	check(x[1<<23], 23)
	check(x[1<<23+1], 0)
	check(x[1<<24-1], 0)
	check(x[1<<24], 24)
	check(x[1<<24+1], 0)
}

func check(x, y byte) {
	if x != y {
		panic("FAIL")
	}
}
`

func TestLargeReloc(t *testing.T) {
	// Test that large relocation addend is handled correctly.
	// In particular, on darwin/arm64 when external linking,
	// Mach-O relocation has only 24-bit addend. See issue #42738.
	testenv.MustHaveGoBuild(t)
	t.Parallel()

	tmpdir := t.TempDir()

	src := filepath.Join(tmpdir, "x.go")
	err := ioutil.WriteFile(src, []byte(testLargeRelocSrc), 0666)
	if err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}
	cmd := exec.Command(testenv.GoToolPath(t), "run", src)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("build failed: %v. output:\n%s", err, out)
	}

	if testenv.HasCGO() { // currently all targets that support cgo can external link
		cmd = exec.Command(testenv.GoToolPath(t), "run", "-ldflags=-linkmode=external", src)
		out, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("build failed: %v. output:\n%s", err, out)
		}
	}
}
