// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file implements the Check function, which drives type-checking.

package types

import (
	"errors"
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
)

// debugging/development support
const (
	debug = false // leave on during development
	trace = false // turn on for detailed type resolution traces
)

// If forceStrict is set, the type-checker enforces additional
// rules not specified by the Go 1 spec, but which will
// catch guaranteed run-time errors if the respective
// code is executed. In other words, programs passing in
// strict mode are Go 1 compliant, but not all Go 1 programs
// will pass in strict mode. The additional rules are:
//
// - A type assertion x.(T) where T is an interface type
//   is invalid if any (statically known) method that exists
//   for both x and T have different signatures.
//
const forceStrict = false

// exprInfo stores information about an untyped expression.
type exprInfo struct {
	isLhs bool // expression is lhs operand of a shift with delayed type-check
	mode  operandMode
	typ   *Basic
	val   constant.Value // constant value; or nil (if not a constant)
}

// A context represents the context within which an object is type-checked.
type context struct {
	decl          *declInfo              // package-level declaration whose init expression/function body is checked
	scope         *Scope                 // top-most scope for lookups
	pos           token.Pos              // if valid, identifiers are looked up as if at position pos (used by Eval)
	iota          constant.Value         // value of iota in a constant declaration; nil otherwise
	errpos        positioner             // if set, identifier position of a constant with inherited initializer
	sig           *Signature             // function signature if inside a function; nil otherwise
	isPanic       map[*ast.CallExpr]bool // set of panic call expressions (used for termination check)
	hasLabel      bool                   // set if a function makes use of labels (only ~1% of functions); unused outside functions
	hasCallOrRecv bool                   // set if an expression contains a function call or channel receive operation
}

// lookup looks up name in the current context and returns the matching object, or nil.
func (ctxt *context) lookup(name string) Object {
	_, obj := ctxt.scope.LookupParent(name, ctxt.pos)
	return obj
}

// An importKey identifies an imported package by import path and source directory
// (directory containing the file containing the import). In practice, the directory
// may always be the same, or may not matter. Given an (import path, directory), an
// importer must always return the same package (but given two different import paths,
// an importer may still return the same package by mapping them to the same package
// paths).
type importKey struct {
	path, dir string
}

// A dotImportKey describes a dot-imported object in the given scope.
type dotImportKey struct {
	scope *Scope
	obj   Object
}

// A Checker maintains the state of the type checker.
// It must be created with NewChecker.
type Checker struct {
	// package information
	// (initialized by NewChecker, valid for the life-time of checker)
	conf *Config
	fset *token.FileSet
	pkg  *Package
	*Info
	version version                    // accepted language version
	objMap  map[Object]*declInfo       // maps package-level objects and (non-interface) methods to declaration info
	impMap  map[importKey]*Package     // maps (import path, source directory) to (complete or fake) package
	posMap  map[*Interface][]token.Pos // maps interface types to lists of embedded interface positions
	typMap  map[string]*Named          // maps an instantiated named type hash to a *Named type

	// pkgPathMap maps package names to the set of distinct import paths we've
	// seen for that name, anywhere in the import graph. It is used for
	// disambiguating package names in error messages.
	//
	// pkgPathMap is allocated lazily, so that we don't pay the price of building
	// it on the happy path. seenPkgMap tracks the packages that we've already
	// walked.
	pkgPathMap map[string]map[string]bool
	seenPkgMap map[*Package]bool

	// information collected during type-checking of a set of package files
	// (initialized by Files, valid only for the duration of check.Files;
	// maps and lists are allocated on demand)
	files        []*ast.File               // package files
	imports      []*PkgName                // list of imported packages
	dotImportMap map[dotImportKey]*PkgName // maps dot-imported objects to the package they were dot-imported through

	firstErr error                 // first error encountered
	methods  map[*TypeName][]*Func // maps package scope type names to associated non-blank (non-interface) methods
	untyped  map[ast.Expr]exprInfo // map of expressions without final type
	delayed  []func()              // stack of delayed action segments; segments are processed in FIFO order
	objPath  []Object              // path of object dependencies during type inference (for cycle reporting)

	// context within which the current object is type-checked
	// (valid only for the duration of type-checking a specific object)
	context

	// debugging
	indent int // indentation for tracing
}

// addDeclDep adds the dependency edge (check.decl -> to) if check.decl exists
func (check *Checker) addDeclDep(to Object) {
	from := check.decl
	if from == nil {
		return // not in a package-level init expression
	}
	if _, found := check.objMap[to]; !found {
		return // to is not a package-level object
	}
	from.addDep(to)
}

func (check *Checker) rememberUntyped(e ast.Expr, lhs bool, mode operandMode, typ *Basic, val constant.Value) {
	m := check.untyped
	if m == nil {
		m = make(map[ast.Expr]exprInfo)
		check.untyped = m
	}
	m[e] = exprInfo{lhs, mode, typ, val}
}

// later pushes f on to the stack of actions that will be processed later;
// either at the end of the current statement, or in case of a local constant
// or variable declaration, before the constant or variable is in scope
// (so that f still sees the scope before any new declarations).
func (check *Checker) later(f func()) {
	check.delayed = append(check.delayed, f)
}

// push pushes obj onto the object path and returns its index in the path.
func (check *Checker) push(obj Object) int {
	check.objPath = append(check.objPath, obj)
	return len(check.objPath) - 1
}

// pop pops and returns the topmost object from the object path.
func (check *Checker) pop() Object {
	i := len(check.objPath) - 1
	obj := check.objPath[i]
	check.objPath[i] = nil
	check.objPath = check.objPath[:i]
	return obj
}

// NewChecker returns a new Checker instance for a given package.
// Package files may be added incrementally via checker.Files.
func NewChecker(conf *Config, fset *token.FileSet, pkg *Package, info *Info) *Checker {
	// make sure we have a configuration
	if conf == nil {
		conf = new(Config)
	}

	// make sure we have an info struct
	if info == nil {
		info = new(Info)
	}

	version, err := parseGoVersion(conf.goVersion)
	if err != nil {
		panic(fmt.Sprintf("invalid Go version %q (%v)", conf.goVersion, err))
	}

	return &Checker{
		conf:    conf,
		fset:    fset,
		pkg:     pkg,
		Info:    info,
		version: version,
		objMap:  make(map[Object]*declInfo),
		impMap:  make(map[importKey]*Package),
		posMap:  make(map[*Interface][]token.Pos),
		typMap:  make(map[string]*Named),
	}
}

// initFiles initializes the files-specific portion of checker.
// The provided files must all belong to the same package.
func (check *Checker) initFiles(files []*ast.File) {
	// start with a clean slate (check.Files may be called multiple times)
	check.files = nil
	check.imports = nil
	check.dotImportMap = nil

	check.firstErr = nil
	check.methods = nil
	check.untyped = nil
	check.delayed = nil

	// determine package name and collect valid files
	pkg := check.pkg
	for _, file := range files {
		switch name := file.Name.Name; pkg.name {
		case "":
			if name != "_" {
				pkg.name = name
			} else {
				check.errorf(file.Name, _BlankPkgName, "invalid package name _")
			}
			fallthrough

		case name:
			check.files = append(check.files, file)

		default:
			check.errorf(atPos(file.Package), _MismatchedPkgName, "package %s; expected %s", name, pkg.name)
			// ignore this file
		}
	}
}

// A bailout panic is used for early termination.
type bailout struct{}

func (check *Checker) handleBailout(err *error) {
	switch p := recover().(type) {
	case nil, bailout:
		// normal return or early exit
		*err = check.firstErr
	default:
		// re-panic
		panic(p)
	}
}

// Files checks the provided files as part of the checker's package.
func (check *Checker) Files(files []*ast.File) error { return check.checkFiles(files) }

var errBadCgo = errors.New("cannot use FakeImportC and go115UsesCgo together")

func (check *Checker) checkFiles(files []*ast.File) (err error) {
	if check.conf.FakeImportC && check.conf.go115UsesCgo {
		return errBadCgo
	}

	defer check.handleBailout(&err)

	check.initFiles(files)

	check.collectObjects()

	check.packageObjects()

	check.processDelayed(0) // incl. all functions

	check.initOrder()

	if !check.conf.DisableUnusedImportCheck {
		check.unusedImports()
	}

	check.recordUntyped()

	if check.Info != nil {
		sanitizeInfo(check.Info)
	}

	check.pkg.complete = true

	// no longer needed - release memory
	check.imports = nil
	check.dotImportMap = nil
	check.pkgPathMap = nil
	check.seenPkgMap = nil

	// TODO(rFindley) There's more memory we should release at this point.

	return
}

// processDelayed processes all delayed actions pushed after top.
func (check *Checker) processDelayed(top int) {
	// If each delayed action pushes a new action, the
	// stack will continue to grow during this loop.
	// However, it is only processing functions (which
	// are processed in a delayed fashion) that may
	// add more actions (such as nested functions), so
	// this is a sufficiently bounded process.
	for i := top; i < len(check.delayed); i++ {
		check.delayed[i]() // may append to check.delayed
	}
	assert(top <= len(check.delayed)) // stack must not have shrunk
	check.delayed = check.delayed[:top]
}

func (check *Checker) record(x *operand) {
	// convert x into a user-friendly set of values
	// TODO(gri) this code can be simplified
	var typ Type
	var val constant.Value
	switch x.mode {
	case invalid:
		typ = Typ[Invalid]
	case novalue:
		typ = (*Tuple)(nil)
	case constant_:
		typ = x.typ
		val = x.val
	default:
		typ = x.typ
	}
	assert(x.expr != nil && typ != nil)

	if isUntyped(typ) {
		// delay type and value recording until we know the type
		// or until the end of type checking
		check.rememberUntyped(x.expr, false, x.mode, typ.(*Basic), val)
	} else {
		check.recordTypeAndValue(x.expr, x.mode, typ, val)
	}
}

func (check *Checker) recordUntyped() {
	if !debug && check.Types == nil {
		return // nothing to do
	}

	for x, info := range check.untyped {
		if debug && isTyped(info.typ) {
			check.dump("%v: %s (type %s) is typed", x.Pos(), x, info.typ)
			unreachable()
		}
		check.recordTypeAndValue(x, info.mode, info.typ, info.val)
	}
}

func (check *Checker) recordTypeAndValue(x ast.Expr, mode operandMode, typ Type, val constant.Value) {
	assert(x != nil)
	assert(typ != nil)
	if mode == invalid {
		return // omit
	}
	if mode == constant_ {
		assert(val != nil)
		// We check is(typ, IsConstType) here as constant expressions may be
		// recorded as type parameters.
		assert(typ == Typ[Invalid] || is(typ, IsConstType))
	}
	if m := check.Types; m != nil {
		m[x] = TypeAndValue{mode, typ, val}
	}
}

func (check *Checker) recordBuiltinType(f ast.Expr, sig *Signature) {
	// f must be a (possibly parenthesized, possibly qualified)
	// identifier denoting a built-in (including unsafe's non-constant
	// functions Add and Slice): record the signature for f and possible
	// children.
	for {
		check.recordTypeAndValue(f, builtin, sig, nil)
		switch p := f.(type) {
		case *ast.Ident, *ast.SelectorExpr:
			return // we're done
		case *ast.ParenExpr:
			f = p.X
		default:
			unreachable()
		}
	}
}

func (check *Checker) recordCommaOkTypes(x ast.Expr, a [2]Type) {
	assert(x != nil)
	if a[0] == nil || a[1] == nil {
		return
	}
	assert(isTyped(a[0]) && isTyped(a[1]) && (isBoolean(a[1]) || a[1] == universeError))
	if m := check.Types; m != nil {
		for {
			tv := m[x]
			assert(tv.Type != nil) // should have been recorded already
			pos := x.Pos()
			tv.Type = NewTuple(
				NewVar(pos, check.pkg, "", a[0]),
				NewVar(pos, check.pkg, "", a[1]),
			)
			m[x] = tv
			// if x is a parenthesized expression (p.X), update p.X
			p, _ := x.(*ast.ParenExpr)
			if p == nil {
				break
			}
			x = p.X
		}
	}
}

func (check *Checker) recordInferred(call ast.Expr, targs []Type, sig *Signature) {
	assert(call != nil)
	assert(sig != nil)
	if m := getInferred(check.Info); m != nil {
		m[call] = _Inferred{targs, sig}
	}
}

func (check *Checker) recordDef(id *ast.Ident, obj Object) {
	assert(id != nil)
	if m := check.Defs; m != nil {
		m[id] = obj
	}
}

func (check *Checker) recordUse(id *ast.Ident, obj Object) {
	assert(id != nil)
	assert(obj != nil)
	if m := check.Uses; m != nil {
		m[id] = obj
	}
}

func (check *Checker) recordImplicit(node ast.Node, obj Object) {
	assert(node != nil)
	assert(obj != nil)
	if m := check.Implicits; m != nil {
		m[node] = obj
	}
}

func (check *Checker) recordSelection(x *ast.SelectorExpr, kind SelectionKind, recv Type, obj Object, index []int, indirect bool) {
	assert(obj != nil && (recv == nil || len(index) > 0))
	check.recordUse(x.Sel, obj)
	if m := check.Selections; m != nil {
		m[x] = &Selection{kind, recv, obj, index, indirect}
	}
}

func (check *Checker) recordScope(node ast.Node, scope *Scope) {
	assert(node != nil)
	assert(scope != nil)
	if m := check.Scopes; m != nil {
		m[node] = scope
	}
}
