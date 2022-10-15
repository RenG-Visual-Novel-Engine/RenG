package object

var FunctionBuiltins = map[string]*Builtin{
	"len": &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return NewFunctionArgsError("len", "필요 인자 : 1 || 호출 인자 : %d", len(args))
			}

			switch arg := args[0].(type) {
			case *String:
				return &Integer{Value: int64(len(arg.Value))}
			case *Array:
				return &Integer{Value: int64(len(arg.Elements))}
			default:
				return NewTypeError("필요 타입 : INTEGER or ARRAY || 호출 타입 : %s", arg.Type())
			}
		},
	},
}
