package bytecode

import (
	"RenG/RVM/src/core/System/code"
	"RenG/RVM/src/core/System/object"
)

type Bytecode struct {
	Instructions code.Instructions
	Constants    []object.Object
}
