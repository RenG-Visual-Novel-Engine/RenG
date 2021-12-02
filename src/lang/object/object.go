package object

import (
	"RenG/src/core"
	"RenG/src/lang/ast"
	"bytes"
	"fmt"
	"strings"
)

const (
	NULL_OBJ = "NULL"

	BOOLEAN_OBJ = "BOOLEAN"
	INTEGER_OBJ = "INTEGER"
	FLOAT_OBJ   = "FLOAT"
	STRING_OBJ  = "STRING"
	ARRAY_OBJ   = "ARRAY"

	FUNCTION_OBJ = "FUNCTION"
	BUILTIN_OBJ  = "BUILTIN"

	RETURN_VALUE_OBJ = "RETURN_VALUE"
	ERROR_OBJ        = "ERROR_OBJ"

	SCREEN_OBJ     = "SCREEN_OBJ"
	LABEL_OBJ      = "LABEL_OBJ"
	CHARACTER_OBJ  = "CHARACTER_OBJ"
	JUMP_LABEL_OBJ = "JUMP_LABEL"
	TRANSFORM_OBJ  = "TRANSFORM_OBJ"
)

type ObjectType string

type Object interface {
	Type() ObjectType
	Inspect() string
}

type Null struct{}

func (n *Null) Type() ObjectType { return NULL_OBJ }
func (n *Null) Inspect() string  { return "null" }

type Boolean struct {
	Value bool
}

func (b *Boolean) Type() ObjectType { return BOOLEAN_OBJ }
func (b *Boolean) Inspect() string  { return fmt.Sprintf("%t", b.Value) }

type Integer struct {
	Value int64
}

func (i *Integer) Inspect() string  { return fmt.Sprintf("%d", i.Value) }
func (i *Integer) Type() ObjectType { return INTEGER_OBJ }

type Float struct {
	Value float64
}

func (f *Float) Inspect() string  { return fmt.Sprintf("%f", f.Value) }
func (f *Float) Type() ObjectType { return FLOAT_OBJ }

type String struct {
	Value string
}

func (s *String) Type() ObjectType { return STRING_OBJ }
func (s *String) Inspect() string  { return s.Value }

type Array struct {
	Elements []Object
}

func (ao *Array) Type() ObjectType { return ARRAY_OBJ }
func (ao *Array) Inspect() string {
	var out bytes.Buffer

	elements := []string{}
	for _, e := range ao.Elements {
		elements = append(elements, e.Inspect())
	}

	out.WriteString("[")
	out.WriteString(strings.Join(elements, ", "))
	out.WriteString("]")

	return out.String()
}

type Function struct {
	Parameters []*ast.Identifier
	Body       *ast.BlockStatement
	Name       *ast.Identifier
	Env        *Environment
}

func (f *Function) Type() ObjectType { return FUNCTION_OBJ }
func (f *Function) Inspect() string {
	var out bytes.Buffer

	params := []string{}
	for _, p := range f.Parameters {
		params = append(params, p.String())
	}

	out.WriteString("def ")
	out.WriteString(f.Name.String() + " (")
	out.WriteString(strings.Join(params, ","))
	out.WriteString(") {\n")
	out.WriteString(f.Body.String() + "\n}")

	return out.String()
}

type BuiltinFunction func(args ...Object) Object

type Builtin struct {
	Fn BuiltinFunction
}

func (b *Builtin) Type() ObjectType { return BUILTIN_OBJ }
func (b *Builtin) Inspect() string  { return "builtin function" }

type ReturnValue struct {
	Value Object
}

func (rv *ReturnValue) Type() ObjectType { return RETURN_VALUE_OBJ }
func (rv *ReturnValue) Inspect() string  { return rv.Value.Inspect() }

type Error struct {
	Message string
}

func (e *Error) Type() ObjectType { return ERROR_OBJ }
func (e *Error) Inspect() string  { return "ERROR: " + e.Message }

type Screen struct {
	Name *ast.Identifier
	Body *ast.BlockStatement
}

func (s *Screen) Type() ObjectType { return SCREEN_OBJ }
func (s *Screen) Inspect() string  { return "{ " + s.Body.String() + " }" }

type Label struct {
	Name *ast.Identifier
	Body *ast.BlockStatement
}

func (l *Label) Type() ObjectType { return LABEL_OBJ }
func (l *Label) Inspect() string  { return "{ " + l.Body.String() + " }" }

type Character struct {
	Name  *String
	Color *core.SDL_Color
}

func (c *Character) Type() ObjectType { return CHARACTER_OBJ }
func (c *Character) Inspect() string  { return c.Name.Value }

type JumpLabel struct {
	Label *ast.Identifier
}

func (jl *JumpLabel) Type() ObjectType { return JUMP_LABEL_OBJ }
func (jl *JumpLabel) Inspect() string  { return jl.Label.Value }

type Transform struct {
	Name *ast.Identifier
	Body *ast.BlockStatement
}

func (t *Transform) Type() ObjectType { return TRANSFORM_OBJ }
func (t *Transform) Inspect() string  { return t.Name.String() }
