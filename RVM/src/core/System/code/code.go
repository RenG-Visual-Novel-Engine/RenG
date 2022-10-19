package code

import "encoding/binary"

type Instructions []byte

type Opcode byte

const (
	OpConstant Opcode = iota
	OpPop
	OpJumpNotTruthy
	OpJump
	OpAdd
	OpSub
	OpMul
	OpDiv
	OpRem
	OpTrue
	OpFalse
	OpNull
	OpEqual
	OpNotEqual
	OpGreaterThan
	OpGreaterThanOrEquel
	OpMinus
	OpBang
	OpGetGlobal
	OpSetGlobal
	OpArray
	OpIndex
	OpCall
	OpReturn
	OpReturnValue
	OpGetLocal
	OpSetLocal
	OpGetBuiltin
)

func ReadUint16(ins Instructions) uint16 {
	return binary.BigEndian.Uint16(ins)
}

func ReadUint32(ins Instructions) uint32 {
	return binary.BigEndian.Uint32(ins)
}
