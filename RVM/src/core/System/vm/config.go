package vm

import "RenG/RVM/src/core/System/object"

const StackSize = 4096

var True = &object.Boolean{Value: true}
var False = &object.Boolean{Value: false}
var Null = &object.Null{}
