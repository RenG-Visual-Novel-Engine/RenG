package vm

import (
	"RenG/Compiler/code"
	"RenG/Compiler/object"
)

type Frame struct {
	fn *object.CompiledFunction
	ip int
}

func NewFrmae(fn *object.CompiledFunction) *Frame {
	return &Frame{fn: fn, ip: -1}
}

func (f *Frame) Instructions() code.Instructions {
	return f.fn.Instructions
}
