// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file implements typechecking of expressions.

package types2

import (
	"cmd/compile/internal/syntax"
	"fmt"
	"go/constant"
	"go/token"
	"math"
)

/*
Basic algorithm:

Expressions are checked recursively, top down. Expression checker functions
are generally of the form:

  func f(x *operand, e *syntax.Expr, ...)

where e is the expression to be checked, and x is the result of the check.
The check performed by f may fail in which case x.mode == invalid, and
related error messages will have been issued by f.

If a hint argument is present, it is the composite literal element type
of an outer composite literal; it is used to type-check composite literal
elements that have no explicit type specification in the source
(e.g.: []T{{...}, {...}}, the hint is the type T in this case).

All expressions are checked via rawExpr, which dispatches according
to expression kind. Upon returning, rawExpr is recording the types and
constant values for all expressions that have an untyped type (those types
may change on the way up in the expression tree). Usually these are constants,
but the results of comparisons or non-constant shifts of untyped constants
may also be untyped, but not constant.

Untyped expressions may eventually become fully typed (i.e., not untyped),
typically when the value is assigned to a variable, or is used otherwise.
The updateExprType method is used to record this final type and update
the recorded types: the type-checked expression tree is again traversed down,
and the new type is propagated as needed. Untyped constant expression values
that become fully typed must now be representable by the full type (constant
sub-expression trees are left alone except for their roots). This mechanism
ensures that a client sees the actual (run-time) type an untyped value would
have. It also permits type-checking of lhs shift operands "as if the shift
were not present": when updateExprType visits an untyped lhs shift operand
and assigns it it's final type, that type must be an integer type, and a
constant lhs must be representable as an integer.

When an expression gets its final type, either on the way out from rawExpr,
on the way down in updateExprType, or at the end of the type checker run,
the type (and constant value, if any) is recorded via Info.Types, if present.
*/

type opPredicates map[syntax.Operator]func(Type) bool

var unaryOpPredicates opPredicates

func init() {
	// Setting unaryOpPredicates in init avoids declaration cycles.
	unaryOpPredicates = opPredicates{
		syntax.Add: isNumeric,
		syntax.Sub: isNumeric,
		syntax.Xor: isInteger,
		syntax.Not: isBoolean,
	}
}

func (check *Checker) op(m opPredicates, x *operand, op syntax.Operator) bool {
	if pred := m[op]; pred != nil {
		if !pred(x.typ) {
			if check.conf.CompilerErrorMessages {
				check.errorf(x, invalidOp+"operator %s not defined on %s", op, x)
			} else {
				check.errorf(x, invalidOp+"operator %s not defined for %s", op, x)
			}
			return false
		}
	} else {
		check.errorf(x, invalidAST+"unknown operator %s", op)
		return false
	}
	return true
}

// overflow checks that the constant x is representable by its type.
// For untyped constants, it checks that the value doesn't become
// arbitrarily large.
func (check *Checker) overflow(x *operand) {
	assert(x.mode == constant_)

	// If the corresponding expression is an operation, use the
	// operator position rather than the start of the expression
	// as error position.
	pos := syntax.StartPos(x.expr)
	what := "" // operator description, if any
	if op, _ := x.expr.(*syntax.Operation); op != nil {
		pos = op.Pos()
		what = opName(op)
	}

	if x.val.Kind() == constant.Unknown {
		// TODO(gri) We should report exactly what went wrong. At the
		//           moment we don't have the (go/constant) API for that.
		//           See also TODO in go/constant/value.go.
		check.error(pos, "constant result is not representable")
		return
	}

	// Typed constants must be representable in
	// their type after each constant operation.
	if isTyped(x.typ) {
		check.representable(x, asBasic(x.typ))
		return
	}

	// Untyped integer values must not grow arbitrarily.
	const prec = 512 // 512 is the constant precision
	if x.val.Kind() == constant.Int && constant.BitLen(x.val) > prec {
		check.errorf(pos, "constant %s overflow", what)
		x.val = constant.MakeUnknown()
	}
}

// opName returns the name of an operation, or the empty string.
// For now, only operations that might overflow are handled.
// TODO(gri) Expand this to a general mechanism giving names to
//           nodes?
func opName(e *syntax.Operation) string {
	op := int(e.Op)
	if e.Y == nil {
		if op < len(op2str1) {
			return op2str1[op]
		}
	} else {
		if op < len(op2str2) {
			return op2str2[op]
		}
	}
	return ""
}

var op2str1 = [...]string{
	syntax.Xor: "bitwise complement",
}

// This is only used for operations that may cause overflow.
var op2str2 = [...]string{
	syntax.Add: "addition",
	syntax.Sub: "subtraction",
	syntax.Xor: "bitwise XOR",
	syntax.Mul: "multiplication",
	syntax.Shl: "shift",
}

func (check *Checker) unary(x *operand, e *syntax.Operation) {
	check.expr(x, e.X)
	if x.mode == invalid {
		return
	}

	switch e.Op {
	case syntax.And:
		// spec: "As an exception to the addressability
		// requirement x may also be a composite literal."
		if _, ok := unparen(e.X).(*syntax.CompositeLit); !ok && x.mode != variable {
			check.errorf(x, invalidOp+"cannot take address of %s", x)
			x.mode = invalid
			return
		}
		x.mode = value
		x.typ = &Pointer{base: x.typ}
		return

	case syntax.Recv:
		typ := asChan(x.typ)
		if typ == nil {
			check.errorf(x, invalidOp+"cannot receive from non-channel %s", x)
			x.mode = invalid
			return
		}
		if typ.dir == SendOnly {
			check.errorf(x, invalidOp+"cannot receive from send-only channel %s", x)
			x.mode = invalid
			return
		}
		x.mode = commaok
		x.typ = typ.elem
		check.hasCallOrRecv = true
		return
	}

	if !check.op(unaryOpPredicates, x, e.Op) {
		x.mode = invalid
		return
	}

	if x.mode == constant_ {
		if x.val.Kind() == constant.Unknown {
			// nothing to do (and don't cause an error below in the overflow check)
			return
		}
		var prec uint
		if isUnsigned(x.typ) {
			prec = uint(check.conf.sizeof(x.typ) * 8)
		}
		x.val = constant.UnaryOp(op2tok[e.Op], x.val, prec)
		x.expr = e
		check.overflow(x)
		return
	}

	x.mode = value
	// x.typ remains unchanged
}

func isShift(op syntax.Operator) bool {
	return op == syntax.Shl || op == syntax.Shr
}

func isComparison(op syntax.Operator) bool {
	// Note: tokens are not ordered well to make this much easier
	switch op {
	case syntax.Eql, syntax.Neq, syntax.Lss, syntax.Leq, syntax.Gtr, syntax.Geq:
		return true
	}
	return false
}

func fitsFloat32(x constant.Value) bool {
	f32, _ := constant.Float32Val(x)
	f := float64(f32)
	return !math.IsInf(f, 0)
}

func roundFloat32(x constant.Value) constant.Value {
	f32, _ := constant.Float32Val(x)
	f := float64(f32)
	if !math.IsInf(f, 0) {
		return constant.MakeFloat64(f)
	}
	return nil
}

func fitsFloat64(x constant.Value) bool {
	f, _ := constant.Float64Val(x)
	return !math.IsInf(f, 0)
}

func roundFloat64(x constant.Value) constant.Value {
	f, _ := constant.Float64Val(x)
	if !math.IsInf(f, 0) {
		return constant.MakeFloat64(f)
	}
	return nil
}

// representableConst reports whether x can be represented as
// value of the given basic type and for the configuration
// provided (only needed for int/uint sizes).
//
// If rounded != nil, *rounded is set to the rounded value of x for
// representable floating-point and complex values, and to an Int
// value for integer values; it is left alone otherwise.
// It is ok to provide the addressof the first argument for rounded.
//
// The check parameter may be nil if representableConst is invoked
// (indirectly) through an exported API call (AssignableTo, ConvertibleTo)
// because we don't need the Checker's config for those calls.
func representableConst(x constant.Value, check *Checker, typ *Basic, rounded *constant.Value) bool {
	if x.Kind() == constant.Unknown {
		return true // avoid follow-up errors
	}

	var conf *Config
	if check != nil {
		conf = check.conf
	}

	switch {
	case isInteger(typ):
		x := constant.ToInt(x)
		if x.Kind() != constant.Int {
			return false
		}
		if rounded != nil {
			*rounded = x
		}
		if x, ok := constant.Int64Val(x); ok {
			switch typ.kind {
			case Int:
				var s = uint(conf.sizeof(typ)) * 8
				return int64(-1)<<(s-1) <= x && x <= int64(1)<<(s-1)-1
			case Int8:
				const s = 8
				return -1<<(s-1) <= x && x <= 1<<(s-1)-1
			case Int16:
				const s = 16
				return -1<<(s-1) <= x && x <= 1<<(s-1)-1
			case Int32:
				const s = 32
				return -1<<(s-1) <= x && x <= 1<<(s-1)-1
			case Int64, UntypedInt:
				return true
			case Uint, Uintptr:
				if s := uint(conf.sizeof(typ)) * 8; s < 64 {
					return 0 <= x && x <= int64(1)<<s-1
				}
				return 0 <= x
			case Uint8:
				const s = 8
				return 0 <= x && x <= 1<<s-1
			case Uint16:
				const s = 16
				return 0 <= x && x <= 1<<s-1
			case Uint32:
				const s = 32
				return 0 <= x && x <= 1<<s-1
			case Uint64:
				return 0 <= x
			default:
				unreachable()
			}
		}
		// x does not fit into int64
		switch n := constant.BitLen(x); typ.kind {
		case Uint, Uintptr:
			var s = uint(conf.sizeof(typ)) * 8
			return constant.Sign(x) >= 0 && n <= int(s)
		case Uint64:
			return constant.Sign(x) >= 0 && n <= 64
		case UntypedInt:
			return true
		}

	case isFloat(typ):
		x := constant.ToFloat(x)
		if x.Kind() != constant.Float {
			return false
		}
		switch typ.kind {
		case Float32:
			if rounded == nil {
				return fitsFloat32(x)
			}
			r := roundFloat32(x)
			if r != nil {
				*rounded = r
				return true
			}
		case Float64:
			if rounded == nil {
				return fitsFloat64(x)
			}
			r := roundFloat64(x)
			if r != nil {
				*rounded = r
				return true
			}
		case UntypedFloat:
			return true
		default:
			unreachable()
		}

	case isComplex(typ):
		x := constant.ToComplex(x)
		if x.Kind() != constant.Complex {
			return false
		}
		switch typ.kind {
		case Complex64:
			if rounded == nil {
				return fitsFloat32(constant.Real(x)) && fitsFloat32(constant.Imag(x))
			}
			re := roundFloat32(constant.Real(x))
			im := roundFloat32(constant.Imag(x))
			if re != nil && im != nil {
				*rounded = constant.BinaryOp(re, token.ADD, constant.MakeImag(im))
				return true
			}
		case Complex128:
			if rounded == nil {
				return fitsFloat64(constant.Real(x)) && fitsFloat64(constant.Imag(x))
			}
			re := roundFloat64(constant.Real(x))
			im := roundFloat64(constant.Imag(x))
			if re != nil && im != nil {
				*rounded = constant.BinaryOp(re, token.ADD, constant.MakeImag(im))
				return true
			}
		case UntypedComplex:
			return true
		default:
			unreachable()
		}

	case isString(typ):
		return x.Kind() == constant.String

	case isBoolean(typ):
		return x.Kind() == constant.Bool
	}

	return false
}

// An errorCode is a (constant) value uniquely identifing a specific error.
type errorCode int

// The following error codes are "borrowed" from go/types which codes for
// all errors. Here we list the few codes currently needed by the various
// conversion checking functions.
// Eventually we will switch to reporting codes for all errors, using a
// an error code table shared between types2 and go/types.
const (
	_ = errorCode(iota)
	_TruncatedFloat
	_NumericOverflow
	_InvalidConstVal
	_InvalidUntypedConversion

	// The following error codes are only returned by operand.assignableTo
	// and none of its callers use the error. Still, we keep returning the
	// error codes to make the transition to reporting error codes all the
	// time easier in the future.
	_IncompatibleAssign
	_InvalidIfaceAssign
	_InvalidChanAssign
)

// representable checks that a constant operand is representable in the given
// basic type.
func (check *Checker) representable(x *operand, typ *Basic) {
	v, code := check.representation(x, typ)
	if code != 0 {
		check.invalidConversion(code, x, typ)
		x.mode = invalid
		return
	}
	assert(v != nil)
	x.val = v
}

// representation returns the representation of the constant operand x as the
// basic type typ.
//
// If no such representation is possible, it returns a non-zero error code.
func (check *Checker) representation(x *operand, typ *Basic) (constant.Value, errorCode) {
	assert(x.mode == constant_)
	v := x.val
	if !representableConst(x.val, check, typ, &v) {
		if isNumeric(x.typ) && isNumeric(typ) {
			// numeric conversion : error msg
			//
			// integer -> integer : overflows
			// integer -> float   : overflows (actually not possible)
			// float   -> integer : truncated
			// float   -> float   : overflows
			//
			if !isInteger(x.typ) && isInteger(typ) {
				return nil, _TruncatedFloat
			} else {
				return nil, _NumericOverflow
			}
		}
		return nil, _InvalidConstVal
	}
	return v, 0
}

func (check *Checker) invalidConversion(code errorCode, x *operand, target Type) {
	msg := "cannot convert %s to %s"
	switch code {
	case _TruncatedFloat:
		msg = "%s truncated to %s"
	case _NumericOverflow:
		msg = "%s overflows %s"
	}
	check.errorf(x, msg, x, target)
}

// updateExprType updates the type of x to typ and invokes itself
// recursively for the operands of x, depending on expression kind.
// If typ is still an untyped and not the final type, updateExprType
// only updates the recorded untyped type for x and possibly its
// operands. Otherwise (i.e., typ is not an untyped type anymore,
// or it is the final type for x), the type and value are recorded.
// Also, if x is a constant, it must be representable as a value of typ,
// and if x is the (formerly untyped) lhs operand of a non-constant
// shift, it must be an integer value.
//
func (check *Checker) updateExprType(x syntax.Expr, typ Type, final bool) {
	old, found := check.untyped[x]
	if !found {
		return // nothing to do
	}

	// update operands of x if necessary
	switch x := x.(type) {
	case *syntax.BadExpr,
		*syntax.FuncLit,
		*syntax.CompositeLit,
		*syntax.IndexExpr,
		*syntax.SliceExpr,
		*syntax.AssertExpr,
		*syntax.ListExpr,
		//*syntax.StarExpr,
		*syntax.KeyValueExpr,
		*syntax.ArrayType,
		*syntax.StructType,
		*syntax.FuncType,
		*syntax.InterfaceType,
		*syntax.MapType,
		*syntax.ChanType:
		// These expression are never untyped - nothing to do.
		// The respective sub-expressions got their final types
		// upon assignment or use.
		if debug {
			check.dump("%v: found old type(%s): %s (new: %s)", posFor(x), x, old.typ, typ)
			unreachable()
		}
		return

	case *syntax.CallExpr:
		// Resulting in an untyped constant (e.g., built-in complex).
		// The respective calls take care of calling updateExprType
		// for the arguments if necessary.

	case *syntax.Name, *syntax.BasicLit, *syntax.SelectorExpr:
		// An identifier denoting a constant, a constant literal,
		// or a qualified identifier (imported untyped constant).
		// No operands to take care of.

	case *syntax.ParenExpr:
		check.updateExprType(x.X, typ, final)

	// case *syntax.UnaryExpr:
	// 	// If x is a constant, the operands were constants.
	// 	// The operands don't need to be updated since they
	// 	// never get "materialized" into a typed value. If
	// 	// left in the untyped map, they will be processed
	// 	// at the end of the type check.
	// 	if old.val != nil {
	// 		break
	// 	}
	// 	check.updateExprType(x.X, typ, final)

	case *syntax.Operation:
		if x.Y == nil {
			// unary expression
			if x.Op == syntax.Mul {
				// see commented out code for StarExpr above
				// TODO(gri) needs cleanup
				if debug {
					unimplemented()
				}
				return
			}
			// If x is a constant, the operands were constants.
			// The operands don't need to be updated since they
			// never get "materialized" into a typed value. If
			// left in the untyped map, they will be processed
			// at the end of the type check.
			if old.val != nil {
				break
			}
			check.updateExprType(x.X, typ, final)
			break
		}

		// binary expression
		if old.val != nil {
			break // see comment for unary expressions
		}
		if isComparison(x.Op) {
			// The result type is independent of operand types
			// and the operand types must have final types.
		} else if isShift(x.Op) {
			// The result type depends only on lhs operand.
			// The rhs type was updated when checking the shift.
			check.updateExprType(x.X, typ, final)
		} else {
			// The operand types match the result type.
			check.updateExprType(x.X, typ, final)
			check.updateExprType(x.Y, typ, final)
		}

	default:
		unreachable()
	}

	// If the new type is not final and still untyped, just
	// update the recorded type.
	if !final && isUntyped(typ) {
		old.typ = asBasic(typ)
		check.untyped[x] = old
		return
	}

	// Otherwise we have the final (typed or untyped type).
	// Remove it from the map of yet untyped expressions.
	delete(check.untyped, x)

	if old.isLhs {
		// If x is the lhs of a shift, its final type must be integer.
		// We already know from the shift check that it is representable
		// as an integer if it is a constant.
		if !isInteger(typ) {
			check.errorf(x, invalidOp+"shifted operand %s (type %s) must be integer", x, typ)
			return
		}
		// Even if we have an integer, if the value is a constant we
		// still must check that it is representable as the specific
		// int type requested (was issue #22969). Fall through here.
	}
	if old.val != nil {
		// If x is a constant, it must be representable as a value of typ.
		c := operand{old.mode, x, old.typ, old.val, 0}
		check.convertUntyped(&c, typ)
		if c.mode == invalid {
			return
		}
	}

	// Everything's fine, record final type and value for x.
	check.recordTypeAndValue(x, old.mode, typ, old.val)
}

// updateExprVal updates the value of x to val.
func (check *Checker) updateExprVal(x syntax.Expr, val constant.Value) {
	if info, ok := check.untyped[x]; ok {
		info.val = val
		check.untyped[x] = info
	}
}

// convertUntyped attempts to set the type of an untyped value to the target type.
func (check *Checker) convertUntyped(x *operand, target Type) {
	newType, val, code := check.implicitTypeAndValue(x, target)
	if code != 0 {
		check.invalidConversion(code, x, target.Underlying())
		x.mode = invalid
		return
	}
	if val != nil {
		x.val = val
		check.updateExprVal(x.expr, val)
	}
	if newType != x.typ {
		x.typ = newType
		check.updateExprType(x.expr, newType, false)
	}
}

// implicitTypeAndValue returns the implicit type of x when used in a context
// where the target type is expected. If no such implicit conversion is
// possible, it returns a nil Type and non-zero error code.
//
// If x is a constant operand, the returned constant.Value will be the
// representation of x in this context.
func (check *Checker) implicitTypeAndValue(x *operand, target Type) (Type, constant.Value, errorCode) {
	target = expand(target)
	if x.mode == invalid || isTyped(x.typ) || target == Typ[Invalid] {
		return x.typ, nil, 0
	}

	if isUntyped(target) {
		// both x and target are untyped
		xkind := x.typ.(*Basic).kind
		tkind := target.(*Basic).kind
		if isNumeric(x.typ) && isNumeric(target) {
			if xkind < tkind {
				return target, nil, 0
			}
		} else if xkind != tkind {
			return nil, nil, _InvalidUntypedConversion
		}
		return x.typ, nil, 0
	}

	if x.isNil() {
		assert(isUntyped(x.typ))
		if hasNil(target) {
			return target, nil, 0
		}
		return nil, nil, _InvalidUntypedConversion
	}

	switch t := optype(target).(type) {
	case *Basic:
		if x.mode == constant_ {
			v, code := check.representation(x, t)
			if code != 0 {
				return nil, nil, code
			}
			return target, v, code
		}
		// Non-constant untyped values may appear as the
		// result of comparisons (untyped bool), intermediate
		// (delayed-checked) rhs operands of shifts, and as
		// the value nil.
		switch x.typ.(*Basic).kind {
		case UntypedBool:
			if !isBoolean(target) {
				return nil, nil, _InvalidUntypedConversion
			}
		case UntypedInt, UntypedRune, UntypedFloat, UntypedComplex:
			if !isNumeric(target) {
				return nil, nil, _InvalidUntypedConversion
			}
		case UntypedString:
			// Non-constant untyped string values are not permitted by the spec and
			// should not occur during normal typechecking passes, but this path is
			// reachable via the AssignableTo API.
			if !isString(target) {
				return nil, nil, _InvalidUntypedConversion
			}
		default:
			return nil, nil, _InvalidUntypedConversion
		}
	case *Sum:
		ok := t.is(func(t Type) bool {
			target, _, _ := check.implicitTypeAndValue(x, t)
			return target != nil
		})
		if !ok {
			return nil, nil, _InvalidUntypedConversion
		}
	case *Interface:
		// Update operand types to the default type rather than the target
		// (interface) type: values must have concrete dynamic types.
		// Untyped nil was handled upfront.
		check.completeInterface(nopos, t)
		if !t.Empty() {
			return nil, nil, _InvalidUntypedConversion // cannot assign untyped values to non-empty interfaces
		}
		return Default(x.typ), nil, 0 // default type for nil is nil
	default:
		return nil, nil, _InvalidUntypedConversion
	}
	return target, nil, 0
}

func (check *Checker) comparison(x, y *operand, op syntax.Operator) {
	// spec: "In any comparison, the first operand must be assignable
	// to the type of the second operand, or vice versa."
	err := ""
	xok, _ := x.assignableTo(check, y.typ, nil)
	yok, _ := y.assignableTo(check, x.typ, nil)
	if xok || yok {
		defined := false
		switch op {
		case syntax.Eql, syntax.Neq:
			// spec: "The equality operators == and != apply to operands that are comparable."
			defined = Comparable(x.typ) && Comparable(y.typ) || x.isNil() && hasNil(y.typ) || y.isNil() && hasNil(x.typ)
		case syntax.Lss, syntax.Leq, syntax.Gtr, syntax.Geq:
			// spec: The ordering operators <, <=, >, and >= apply to operands that are ordered."
			defined = isOrdered(x.typ) && isOrdered(y.typ)
		default:
			unreachable()
		}
		if !defined {
			typ := x.typ
			if x.isNil() {
				typ = y.typ
			}
			if check.conf.CompilerErrorMessages {
				err = check.sprintf("operator %s not defined on %s", op, typ)
			} else {
				err = check.sprintf("operator %s not defined for %s", op, typ)
			}
		}
	} else {
		err = check.sprintf("mismatched types %s and %s", x.typ, y.typ)
	}

	if err != "" {
		// TODO(gri) better error message for cases where one can only compare against nil
		check.errorf(x, invalidOp+"cannot compare %s %s %s (%s)", x.expr, op, y.expr, err)
		x.mode = invalid
		return
	}

	if x.mode == constant_ && y.mode == constant_ {
		x.val = constant.MakeBool(constant.Compare(x.val, op2tok[op], y.val))
		// The operands are never materialized; no need to update
		// their types.
	} else {
		x.mode = value
		// The operands have now their final types, which at run-
		// time will be materialized. Update the expression trees.
		// If the current types are untyped, the materialized type
		// is the respective default type.
		check.updateExprType(x.expr, Default(x.typ), true)
		check.updateExprType(y.expr, Default(y.typ), true)
	}

	// spec: "Comparison operators compare two operands and yield
	//        an untyped boolean value."
	x.typ = Typ[UntypedBool]
}

// If e != nil, it must be the shift expression; it may be nil for non-constant shifts.
func (check *Checker) shift(x, y *operand, e syntax.Expr, op syntax.Operator) {
	// TODO(gri) This function seems overly complex. Revisit.

	var xval constant.Value
	if x.mode == constant_ {
		xval = constant.ToInt(x.val)
	}

	if isInteger(x.typ) || isUntyped(x.typ) && xval != nil && xval.Kind() == constant.Int {
		// The lhs is of integer type or an untyped constant representable
		// as an integer. Nothing to do.
	} else {
		// shift has no chance
		check.errorf(x, invalidOp+"shifted operand %s must be integer", x)
		x.mode = invalid
		return
	}

	// spec: "The right operand in a shift expression must have integer type
	// or be an untyped constant representable by a value of type uint."

	// Provide a good error message for negative shift counts.
	if y.mode == constant_ {
		yval := constant.ToInt(y.val) // consider -1, 1.0, but not -1.1
		if yval.Kind() == constant.Int && constant.Sign(yval) < 0 {
			check.errorf(y, invalidOp+"negative shift count %s", y)
			x.mode = invalid
			return
		}
	}

	// Caution: Check for isUntyped first because isInteger includes untyped
	//          integers (was bug #43697).
	if isUntyped(y.typ) {
		check.convertUntyped(y, Typ[Uint])
		if y.mode == invalid {
			x.mode = invalid
			return
		}
	} else if !isInteger(y.typ) {
		check.errorf(y, invalidOp+"shift count %s must be integer", y)
		x.mode = invalid
		return
	} else if !isUnsigned(y.typ) && !check.allowVersion(check.pkg, 1, 13) {
		check.errorf(y, invalidOp+"signed shift count %s requires go1.13 or later", y)
		x.mode = invalid
		return
	}

	if x.mode == constant_ {
		if y.mode == constant_ {
			// if either x or y has an unknown value, the result is unknown
			if x.val.Kind() == constant.Unknown || y.val.Kind() == constant.Unknown {
				x.val = constant.MakeUnknown()
				// ensure the correct type - see comment below
				if !isInteger(x.typ) {
					x.typ = Typ[UntypedInt]
				}
				return
			}
			// rhs must be within reasonable bounds in constant shifts
			const shiftBound = 1023 - 1 + 52 // so we can express smallestFloat64 (see issue #44057)
			s, ok := constant.Uint64Val(y.val)
			if !ok || s > shiftBound {
				check.errorf(y, invalidOp+"invalid shift count %s", y)
				x.mode = invalid
				return
			}
			// The lhs is representable as an integer but may not be an integer
			// (e.g., 2.0, an untyped float) - this can only happen for untyped
			// non-integer numeric constants. Correct the type so that the shift
			// result is of integer type.
			if !isInteger(x.typ) {
				x.typ = Typ[UntypedInt]
			}
			// x is a constant so xval != nil and it must be of Int kind.
			x.val = constant.Shift(xval, op2tok[op], uint(s))
			x.expr = e
			check.overflow(x)
			return
		}

		// non-constant shift with constant lhs
		if isUntyped(x.typ) {
			// spec: "If the left operand of a non-constant shift
			// expression is an untyped constant, the type of the
			// constant is what it would be if the shift expression
			// were replaced by its left operand alone.".
			//
			// Delay operand checking until we know the final type
			// by marking the lhs expression as lhs shift operand.
			//
			// Usually (in correct programs), the lhs expression
			// is in the untyped map. However, it is possible to
			// create incorrect programs where the same expression
			// is evaluated twice (via a declaration cycle) such
			// that the lhs expression type is determined in the
			// first round and thus deleted from the map, and then
			// not found in the second round (double insertion of
			// the same expr node still just leads to one entry for
			// that node, and it can only be deleted once).
			// Be cautious and check for presence of entry.
			// Example: var e, f = int(1<<""[f]) // issue 11347
			if info, found := check.untyped[x.expr]; found {
				info.isLhs = true
				check.untyped[x.expr] = info
			}
			// keep x's type
			x.mode = value
			return
		}
	}

	// non-constant shift - lhs must be an integer
	if !isInteger(x.typ) {
		check.errorf(x, invalidOp+"shifted operand %s must be integer", x)
		x.mode = invalid
		return
	}

	x.mode = value
}

var binaryOpPredicates opPredicates

func init() {
	// Setting binaryOpPredicates in init avoids declaration cycles.
	binaryOpPredicates = opPredicates{
		syntax.Add: isNumericOrString,
		syntax.Sub: isNumeric,
		syntax.Mul: isNumeric,
		syntax.Div: isNumeric,
		syntax.Rem: isInteger,

		syntax.And:    isInteger,
		syntax.Or:     isInteger,
		syntax.Xor:    isInteger,
		syntax.AndNot: isInteger,

		syntax.AndAnd: isBoolean,
		syntax.OrOr:   isBoolean,
	}
}

// If e != nil, it must be the binary expression; it may be nil for non-constant expressions
// (when invoked for an assignment operation where the binary expression is implicit).
func (check *Checker) binary(x *operand, e syntax.Expr, lhs, rhs syntax.Expr, op syntax.Operator) {
	var y operand

	check.expr(x, lhs)
	check.expr(&y, rhs)

	if x.mode == invalid {
		return
	}
	if y.mode == invalid {
		x.mode = invalid
		x.expr = y.expr
		return
	}

	if isShift(op) {
		check.shift(x, &y, e, op)
		return
	}

	check.convertUntyped(x, y.typ)
	if x.mode == invalid {
		return
	}
	check.convertUntyped(&y, x.typ)
	if y.mode == invalid {
		x.mode = invalid
		return
	}

	if isComparison(op) {
		check.comparison(x, &y, op)
		return
	}

	if !check.identical(x.typ, y.typ) {
		// only report an error if we have valid types
		// (otherwise we had an error reported elsewhere already)
		if x.typ != Typ[Invalid] && y.typ != Typ[Invalid] {
			check.errorf(x, invalidOp+"mismatched types %s and %s", x.typ, y.typ)
		}
		x.mode = invalid
		return
	}

	if !check.op(binaryOpPredicates, x, op) {
		x.mode = invalid
		return
	}

	if op == syntax.Div || op == syntax.Rem {
		// check for zero divisor
		if (x.mode == constant_ || isInteger(x.typ)) && y.mode == constant_ && constant.Sign(y.val) == 0 {
			check.error(&y, invalidOp+"division by zero")
			x.mode = invalid
			return
		}

		// check for divisor underflow in complex division (see issue 20227)
		if x.mode == constant_ && y.mode == constant_ && isComplex(x.typ) {
			re, im := constant.Real(y.val), constant.Imag(y.val)
			re2, im2 := constant.BinaryOp(re, token.MUL, re), constant.BinaryOp(im, token.MUL, im)
			if constant.Sign(re2) == 0 && constant.Sign(im2) == 0 {
				check.error(&y, invalidOp+"division by zero")
				x.mode = invalid
				return
			}
		}
	}

	if x.mode == constant_ && y.mode == constant_ {
		// if either x or y has an unknown value, the result is unknown
		if x.val.Kind() == constant.Unknown || y.val.Kind() == constant.Unknown {
			x.val = constant.MakeUnknown()
			// x.typ is unchanged
			return
		}
		// force integer division for integer operands
		tok := op2tok[op]
		if op == syntax.Div && isInteger(x.typ) {
			tok = token.QUO_ASSIGN
		}
		x.val = constant.BinaryOp(x.val, tok, y.val)
		x.expr = e
		check.overflow(x)
		return
	}

	x.mode = value
	// x.typ is unchanged
}

// exprKind describes the kind of an expression; the kind
// determines if an expression is valid in 'statement context'.
type exprKind int

const (
	conversion exprKind = iota
	expression
	statement
)

// rawExpr typechecks expression e and initializes x with the expression
// value or type. If an error occurred, x.mode is set to invalid.
// If hint != nil, it is the type of a composite literal element.
//
func (check *Checker) rawExpr(x *operand, e syntax.Expr, hint Type) exprKind {
	if check.conf.Trace {
		check.trace(e.Pos(), "expr %s", e)
		check.indent++
		defer func() {
			check.indent--
			check.trace(e.Pos(), "=> %s", x)
		}()
	}

	kind := check.exprInternal(x, e, hint)
	check.record(x)

	return kind
}

// exprInternal contains the core of type checking of expressions.
// Must only be called by rawExpr.
//
func (check *Checker) exprInternal(x *operand, e syntax.Expr, hint Type) exprKind {
	// make sure x has a valid state in case of bailout
	// (was issue 5770)
	x.mode = invalid
	x.typ = Typ[Invalid]

	switch e := e.(type) {
	case nil:
		unreachable()

	case *syntax.BadExpr:
		goto Error // error was reported before

	case *syntax.Name:
		check.ident(x, e, nil, false)

	case *syntax.DotsType:
		// dots are handled explicitly where they are legal
		// (array composite literals and parameter lists)
		check.error(e, "invalid use of '...'")
		goto Error

	case *syntax.BasicLit:
		if e.Bad {
			goto Error // error reported during parsing
		}
		switch e.Kind {
		case syntax.IntLit, syntax.FloatLit, syntax.ImagLit:
			check.langCompat(e)
			// The max. mantissa precision for untyped numeric values
			// is 512 bits, or 4048 bits for each of the two integer
			// parts of a fraction for floating-point numbers that are
			// represented accurately in the go/constant package.
			// Constant literals that are longer than this many bits
			// are not meaningful; and excessively long constants may
			// consume a lot of space and time for a useless conversion.
			// Cap constant length with a generous upper limit that also
			// allows for separators between all digits.
			const limit = 10000
			if len(e.Value) > limit {
				check.errorf(e, "excessively long constant: %s... (%d chars)", e.Value[:10], len(e.Value))
				goto Error
			}
		}
		x.setConst(e.Kind, e.Value)
		if x.mode == invalid {
			// The parser already establishes syntactic correctness.
			// If we reach here it's because of number under-/overflow.
			// TODO(gri) setConst (and in turn the go/constant package)
			// should return an error describing the issue.
			check.errorf(e, "malformed constant: %s", e.Value)
			goto Error
		}

	case *syntax.FuncLit:
		if sig, ok := check.typ(e.Type).(*Signature); ok {
			if !check.conf.IgnoreFuncBodies && e.Body != nil {
				// Anonymous functions are considered part of the
				// init expression/func declaration which contains
				// them: use existing package-level declaration info.
				decl := check.decl // capture for use in closure below
				iota := check.iota // capture for use in closure below (#22345)
				// Don't type-check right away because the function may
				// be part of a type definition to which the function
				// body refers. Instead, type-check as soon as possible,
				// but before the enclosing scope contents changes (#22992).
				check.later(func() {
					check.funcBody(decl, "<function literal>", sig, e.Body, iota)
				})
			}
			x.mode = value
			x.typ = sig
		} else {
			check.errorf(e, invalidAST+"invalid function literal %v", e)
			goto Error
		}

	case *syntax.CompositeLit:
		var typ, base Type

		switch {
		case e.Type != nil:
			// composite literal type present - use it
			// [...]T array types may only appear with composite literals.
			// Check for them here so we don't have to handle ... in general.
			if atyp, _ := e.Type.(*syntax.ArrayType); atyp != nil && atyp.Len == nil {
				// We have an "open" [...]T array type.
				// Create a new ArrayType with unknown length (-1)
				// and finish setting it up after analyzing the literal.
				typ = &Array{len: -1, elem: check.varType(atyp.Elem)}
				base = typ
				break
			}
			typ = check.typ(e.Type)
			base = typ

		case hint != nil:
			// no composite literal type present - use hint (element type of enclosing type)
			typ = hint
			base, _ = deref(under(typ)) // *T implies &T{}

		default:
			// TODO(gri) provide better error messages depending on context
			check.error(e, "missing type in composite literal")
			goto Error
		}

		switch utyp := optype(base).(type) {
		case *Struct:
			if len(e.ElemList) == 0 {
				break
			}
			fields := utyp.fields
			if _, ok := e.ElemList[0].(*syntax.KeyValueExpr); ok {
				// all elements must have keys
				visited := make([]bool, len(fields))
				for _, e := range e.ElemList {
					kv, _ := e.(*syntax.KeyValueExpr)
					if kv == nil {
						check.error(e, "mixture of field:value and value elements in struct literal")
						continue
					}
					key, _ := kv.Key.(*syntax.Name)
					// do all possible checks early (before exiting due to errors)
					// so we don't drop information on the floor
					check.expr(x, kv.Value)
					if key == nil {
						check.errorf(kv, "invalid field name %s in struct literal", kv.Key)
						continue
					}
					i := fieldIndex(utyp.fields, check.pkg, key.Value)
					if i < 0 {
						if check.conf.CompilerErrorMessages {
							check.errorf(kv.Key, "unknown field '%s' in struct literal of type %s", key.Value, base)
						} else {
							check.errorf(kv.Key, "unknown field %s in struct literal", key.Value)
						}
						continue
					}
					fld := fields[i]
					check.recordUse(key, fld)
					etyp := fld.typ
					check.assignment(x, etyp, "struct literal")
					// 0 <= i < len(fields)
					if visited[i] {
						check.errorf(kv, "duplicate field name %s in struct literal", key.Value)
						continue
					}
					visited[i] = true
				}
			} else {
				// no element must have a key
				for i, e := range e.ElemList {
					if kv, _ := e.(*syntax.KeyValueExpr); kv != nil {
						check.error(kv, "mixture of field:value and value elements in struct literal")
						continue
					}
					check.expr(x, e)
					if i >= len(fields) {
						check.error(x, "too many values in struct literal")
						break // cannot continue
					}
					// i < len(fields)
					fld := fields[i]
					if !fld.Exported() && fld.pkg != check.pkg {
						check.errorf(x, "implicit assignment to unexported field %s in %s literal", fld.name, typ)
						continue
					}
					etyp := fld.typ
					check.assignment(x, etyp, "struct literal")
				}
				if len(e.ElemList) < len(fields) {
					check.error(e.Rbrace, "too few values in struct literal")
					// ok to continue
				}
			}

		case *Array:
			// Prevent crash if the array referred to is not yet set up. Was issue #18643.
			// This is a stop-gap solution. Should use Checker.objPath to report entire
			// path starting with earliest declaration in the source. TODO(gri) fix this.
			if utyp.elem == nil {
				check.error(e, "illegal cycle in type declaration")
				goto Error
			}
			n := check.indexedElts(e.ElemList, utyp.elem, utyp.len)
			// If we have an array of unknown length (usually [...]T arrays, but also
			// arrays [n]T where n is invalid) set the length now that we know it and
			// record the type for the array (usually done by check.typ which is not
			// called for [...]T). We handle [...]T arrays and arrays with invalid
			// length the same here because it makes sense to "guess" the length for
			// the latter if we have a composite literal; e.g. for [n]int{1, 2, 3}
			// where n is invalid for some reason, it seems fair to assume it should
			// be 3 (see also Checked.arrayLength and issue #27346).
			if utyp.len < 0 {
				utyp.len = n
				// e.Type is missing if we have a composite literal element
				// that is itself a composite literal with omitted type. In
				// that case there is nothing to record (there is no type in
				// the source at that point).
				if e.Type != nil {
					check.recordTypeAndValue(e.Type, typexpr, utyp, nil)
				}
			}

		case *Slice:
			// Prevent crash if the slice referred to is not yet set up.
			// See analogous comment for *Array.
			if utyp.elem == nil {
				check.error(e, "illegal cycle in type declaration")
				goto Error
			}
			check.indexedElts(e.ElemList, utyp.elem, -1)

		case *Map:
			// Prevent crash if the map referred to is not yet set up.
			// See analogous comment for *Array.
			if utyp.key == nil || utyp.elem == nil {
				check.error(e, "illegal cycle in type declaration")
				goto Error
			}
			visited := make(map[interface{}][]Type, len(e.ElemList))
			for _, e := range e.ElemList {
				kv, _ := e.(*syntax.KeyValueExpr)
				if kv == nil {
					check.error(e, "missing key in map literal")
					continue
				}
				check.exprWithHint(x, kv.Key, utyp.key)
				check.assignment(x, utyp.key, "map literal")
				if x.mode == invalid {
					continue
				}
				if x.mode == constant_ {
					duplicate := false
					// if the key is of interface type, the type is also significant when checking for duplicates
					xkey := keyVal(x.val)
					if asInterface(utyp.key) != nil {
						for _, vtyp := range visited[xkey] {
							if check.identical(vtyp, x.typ) {
								duplicate = true
								break
							}
						}
						visited[xkey] = append(visited[xkey], x.typ)
					} else {
						_, duplicate = visited[xkey]
						visited[xkey] = nil
					}
					if duplicate {
						check.errorf(x, "duplicate key %s in map literal", x.val)
						continue
					}
				}
				check.exprWithHint(x, kv.Value, utyp.elem)
				check.assignment(x, utyp.elem, "map literal")
			}

		default:
			// when "using" all elements unpack KeyValueExpr
			// explicitly because check.use doesn't accept them
			for _, e := range e.ElemList {
				if kv, _ := e.(*syntax.KeyValueExpr); kv != nil {
					// Ideally, we should also "use" kv.Key but we can't know
					// if it's an externally defined struct key or not. Going
					// forward anyway can lead to other errors. Give up instead.
					e = kv.Value
				}
				check.use(e)
			}
			// if utyp is invalid, an error was reported before
			if utyp != Typ[Invalid] {
				check.errorf(e, "invalid composite literal type %s", typ)
				goto Error
			}
		}

		x.mode = value
		x.typ = typ

	case *syntax.ParenExpr:
		kind := check.rawExpr(x, e.X, nil)
		x.expr = e
		return kind

	case *syntax.SelectorExpr:
		check.selector(x, e)

	case *syntax.IndexExpr:
		if check.indexExpr(x, e) {
			check.funcInst(x, e)
		}
		if x.mode == invalid {
			goto Error
		}

	case *syntax.SliceExpr:
		check.sliceExpr(x, e)
		if x.mode == invalid {
			goto Error
		}

	case *syntax.AssertExpr:
		check.expr(x, e.X)
		if x.mode == invalid {
			goto Error
		}
		xtyp, _ := under(x.typ).(*Interface)
		if xtyp == nil {
			check.errorf(x, "%s is not an interface type", x)
			goto Error
		}
		check.ordinaryType(x.Pos(), xtyp)
		// x.(type) expressions are encoded via TypeSwitchGuards
		if e.Type == nil {
			check.error(e, invalidAST+"invalid use of AssertExpr")
			goto Error
		}
		T := check.varType(e.Type)
		if T == Typ[Invalid] {
			goto Error
		}
		check.typeAssertion(posFor(x), x, xtyp, T)
		x.mode = commaok
		x.typ = T

	case *syntax.TypeSwitchGuard:
		// x.(type) expressions are handled explicitly in type switches
		check.error(e, invalidAST+"use of .(type) outside type switch")
		goto Error

	case *syntax.CallExpr:
		return check.callExpr(x, e)

	case *syntax.ListExpr:
		// catch-all for unexpected expression lists
		check.error(e, "unexpected list of expressions")
		goto Error

	// case *syntax.UnaryExpr:
	// 	check.expr(x, e.X)
	// 	if x.mode == invalid {
	// 		goto Error
	// 	}
	// 	check.unary(x, e, e.Op)
	// 	if x.mode == invalid {
	// 		goto Error
	// 	}
	// 	if e.Op == token.ARROW {
	// 		x.expr = e
	// 		return statement // receive operations may appear in statement context
	// 	}

	// case *syntax.BinaryExpr:
	// 	check.binary(x, e, e.X, e.Y, e.Op)
	// 	if x.mode == invalid {
	// 		goto Error
	// 	}

	case *syntax.Operation:
		if e.Y == nil {
			// unary expression
			if e.Op == syntax.Mul {
				// pointer indirection
				check.exprOrType(x, e.X)
				switch x.mode {
				case invalid:
					goto Error
				case typexpr:
					x.typ = &Pointer{base: x.typ}
				default:
					if typ := asPointer(x.typ); typ != nil {
						x.mode = variable
						x.typ = typ.base
					} else {
						check.errorf(x, invalidOp+"cannot indirect %s", x)
						goto Error
					}
				}
				break
			}

			check.unary(x, e)
			if x.mode == invalid {
				goto Error
			}
			if e.Op == syntax.Recv {
				x.expr = e
				return statement // receive operations may appear in statement context
			}
			break
		}

		// binary expression
		check.binary(x, e, e.X, e.Y, e.Op)
		if x.mode == invalid {
			goto Error
		}

	case *syntax.KeyValueExpr:
		// key:value expressions are handled in composite literals
		check.error(e, invalidAST+"no key:value expected")
		goto Error

	case *syntax.ArrayType, *syntax.SliceType, *syntax.StructType, *syntax.FuncType,
		*syntax.InterfaceType, *syntax.MapType, *syntax.ChanType:
		x.mode = typexpr
		x.typ = check.typ(e)
		// Note: rawExpr (caller of exprInternal) will call check.recordTypeAndValue
		// even though check.typ has already called it. This is fine as both
		// times the same expression and type are recorded. It is also not a
		// performance issue because we only reach here for composite literal
		// types, which are comparatively rare.

	default:
		panic(fmt.Sprintf("%s: unknown expression type %T", posFor(e), e))
	}

	// everything went well
	x.expr = e
	return expression

Error:
	x.mode = invalid
	x.expr = e
	return statement // avoid follow-up errors
}

func keyVal(x constant.Value) interface{} {
	switch x.Kind() {
	case constant.Bool:
		return constant.BoolVal(x)
	case constant.String:
		return constant.StringVal(x)
	case constant.Int:
		if v, ok := constant.Int64Val(x); ok {
			return v
		}
		if v, ok := constant.Uint64Val(x); ok {
			return v
		}
	case constant.Float:
		v, _ := constant.Float64Val(x)
		return v
	case constant.Complex:
		r, _ := constant.Float64Val(constant.Real(x))
		i, _ := constant.Float64Val(constant.Imag(x))
		return complex(r, i)
	}
	return x
}

// typeAssertion checks that x.(T) is legal; xtyp must be the type of x.
func (check *Checker) typeAssertion(pos syntax.Pos, x *operand, xtyp *Interface, T Type) {
	method, wrongType := check.assertableTo(xtyp, T)
	if method == nil {
		return
	}
	var msg string
	if wrongType != nil {
		if check.identical(method.typ, wrongType.typ) {
			msg = fmt.Sprintf("missing method %s (%s has pointer receiver)", method.name, method.name)
		} else {
			msg = fmt.Sprintf("wrong type for method %s (have %s, want %s)", method.name, wrongType.typ, method.typ)
		}
	} else {
		msg = "missing method " + method.name
	}
	if check.conf.CompilerErrorMessages {
		check.errorf(pos, "impossible type assertion: %s (%s)", x, msg)
	} else {
		check.errorf(pos, "%s cannot have dynamic type %s (%s)", x, T, msg)
	}
}

// expr typechecks expression e and initializes x with the expression value.
// The result must be a single value.
// If an error occurred, x.mode is set to invalid.
//
func (check *Checker) expr(x *operand, e syntax.Expr) {
	check.rawExpr(x, e, nil)
	check.exclude(x, 1<<novalue|1<<builtin|1<<typexpr)
	check.singleValue(x)
}

// multiExpr is like expr but the result may also be a multi-value.
func (check *Checker) multiExpr(x *operand, e syntax.Expr) {
	check.rawExpr(x, e, nil)
	check.exclude(x, 1<<novalue|1<<builtin|1<<typexpr)
}

// exprWithHint typechecks expression e and initializes x with the expression value;
// hint is the type of a composite literal element.
// If an error occurred, x.mode is set to invalid.
//
func (check *Checker) exprWithHint(x *operand, e syntax.Expr, hint Type) {
	assert(hint != nil)
	check.rawExpr(x, e, hint)
	check.exclude(x, 1<<novalue|1<<builtin|1<<typexpr)
	check.singleValue(x)
}

// exprOrType typechecks expression or type e and initializes x with the expression value or type.
// If an error occurred, x.mode is set to invalid.
//
func (check *Checker) exprOrType(x *operand, e syntax.Expr) {
	check.rawExpr(x, e, nil)
	check.exclude(x, 1<<novalue)
	check.singleValue(x)
}

// exclude reports an error if x.mode is in modeset and sets x.mode to invalid.
// The modeset may contain any of 1<<novalue, 1<<builtin, 1<<typexpr.
func (check *Checker) exclude(x *operand, modeset uint) {
	if modeset&(1<<x.mode) != 0 {
		var msg string
		switch x.mode {
		case novalue:
			if modeset&(1<<typexpr) != 0 {
				msg = "%s used as value"
			} else {
				msg = "%s used as value or type"
			}
		case builtin:
			msg = "%s must be called"
		case typexpr:
			msg = "%s is not an expression"
		default:
			unreachable()
		}
		check.errorf(x, msg, x)
		x.mode = invalid
	}
}

// singleValue reports an error if x describes a tuple and sets x.mode to invalid.
func (check *Checker) singleValue(x *operand) {
	if x.mode == value {
		// tuple types are never named - no need for underlying type below
		if t, ok := x.typ.(*Tuple); ok {
			assert(t.Len() != 1)
			check.errorf(x, "%d-valued %s where single value is expected", t.Len(), x)
			x.mode = invalid
		}
	}
}

// op2tok translates syntax.Operators into token.Tokens.
var op2tok = [...]token.Token{
	syntax.Def:  token.ILLEGAL,
	syntax.Not:  token.NOT,
	syntax.Recv: token.ILLEGAL,

	syntax.OrOr:   token.LOR,
	syntax.AndAnd: token.LAND,

	syntax.Eql: token.EQL,
	syntax.Neq: token.NEQ,
	syntax.Lss: token.LSS,
	syntax.Leq: token.LEQ,
	syntax.Gtr: token.GTR,
	syntax.Geq: token.GEQ,

	syntax.Add: token.ADD,
	syntax.Sub: token.SUB,
	syntax.Or:  token.OR,
	syntax.Xor: token.XOR,

	syntax.Mul:    token.MUL,
	syntax.Div:    token.QUO,
	syntax.Rem:    token.REM,
	syntax.And:    token.AND,
	syntax.AndNot: token.AND_NOT,
	syntax.Shl:    token.SHL,
	syntax.Shr:    token.SHR,
}
