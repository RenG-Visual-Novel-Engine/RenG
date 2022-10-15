package object

import (
	"fmt"
)

func NewTypeError(format string, a ...interface{}) *Error {
	return &Error{Message: "TypeError : " + fmt.Sprintf(format, a...)}
}

func NewFunctionArgsError(fnName string, format string, a ...interface{}) *Error {
	return &Error{Message: "FunctionArgsError (" + fnName + ") : " + fmt.Sprintf(format, a...)}
}
