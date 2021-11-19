package evaluator

import (
	"RenG/src/config"
	"RenG/src/lang/object"
	"fmt"
	"os/exec"
	"runtime"
	"time"
)

var FunctionBuiltins = map[string]*object.Builtin{
	"len": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d", len(args))
			}

			switch arg := args[0].(type) {
			case *object.Array:
				return &object.Integer{Value: int64(len(arg.Elements))}
			case *object.String:
				return &object.Integer{Value: int64(len(arg.Value))}
			default:
				return newError("arguments to len not supported, got=%s", args[0].Type())
			}
		},
	},
	"append": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("wrong number of arguments. got=%d, want=2", len(args))
			}
			if args[0].Type() != object.ARRAY_OBJ {
				return newError("argument to push must be ARRAY, got %s", args[0].Type())
			}

			arr := args[0].(*object.Array)
			length := len(arr.Elements)

			newElements := make([]object.Object, length+1)
			copy(newElements, arr.Elements)
			newElements[length] = args[1]

			return &object.Array{Elements: newElements}
		},
	},
	"pause": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1", len(args))
			}

			switch args[0].Type() {
			case object.INTEGER_OBJ:
				time.Sleep(time.Second * time.Duration(args[0].(*object.Integer).Value))
			case object.FLOAT_OBJ:
				time.Sleep(time.Second * time.Duration(args[0].(*object.Float).Value))
			default:
				return newError("argument to pause must be INTEGER or FLOAT, got %s", args[0].Type())
			}

			return nil
		},
	},
	"Start": {
		Fn: func(args ...object.Object) object.Object {
			config.DeleteScreen("main_menu")

			config.StartChannel <- true

			return nil
		},
	},
	"GoSite": {
		Fn: func(args ...object.Object) object.Object {
			if runtime.GOOS == "windows" {
				cmd := exec.Command("cmd", "/C", "start", "/max", args[0].(*object.String).Value)
				cmd.Run()
			}

			return nil
		},
	},
	"print": { // 테스트 용으로 제작된 임시 출력 함수
		Fn: func(args ...object.Object) object.Object {
			for _, arg := range args {
				fmt.Println(arg.Inspect())
			}

			return NULL
		},
	},
}
