package vm

import (
	"RenG/RVM/src/core/System/code"
	"RenG/RVM/src/core/System/object"
)

type Frame interface {
	Instructions() code.Instructions
	GetIp() int
	SetIp(int)
	GetBasePointer() int
}

func NewFrmae(obj object.Object, basePointer int) Frame {
	switch obj := obj.(type) {
	case *object.CompiledFunction:
		return &FunctionFrame{
			fn:          obj,
			ip:          -1,
			basePointer: basePointer,
		}
	default:
		return nil
	}
}

type FunctionFrame struct {
	fn          *object.CompiledFunction
	ip          int
	basePointer int
}

func (ff *FunctionFrame) Instructions() code.Instructions { return ff.fn.Instructions }
func (ff *FunctionFrame) GetIp() int                      { return ff.ip }
func (ff *FunctionFrame) SetIp(ip int)                    { ff.ip = ip }
func (ff *FunctionFrame) GetBasePointer() int             { return ff.basePointer }
