package code

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
