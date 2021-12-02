package evaluator

import (
	"RenG/src/config"
	"RenG/src/core"
	"RenG/src/lang/object"
	"fmt"
	"os/exec"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
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

			return NULL
		},
	},
	"Start": {
		Fn: func(args ...object.Object) object.Object {
			config.DeleteScreen("main_menu")

			config.StartChannel <- true

			return NULL
		},
	},
	/*------------os------------*/
	"OStype": {
		Fn: func(args ...object.Object) object.Object {
			return &object.String{Value: runtime.GOOS}
		},
	},
	"OSusername": {
		Fn: func(args ...object.Object) object.Object {
			user, err := user.Current()
			if err != nil {
				return newError("Not find OS user, got=%s", err)
			}

			switch username := user.Username; runtime.GOOS {
			case "windows":
				switch slice := strings.Split(username, "\\"); len(slice) {
				case 1:
					return &object.String{Value: slice[0]}
				case 2:
					return &object.String{Value: slice[1]}
				default:
					return newError("username is error, isn't your username? got=%s", username)
				}
			default:
				return newError("OSusername support only windows, your OS = %s", runtime.GOOS)
			}
		},
	},
	"OSbgChange": {
		Fn: func(args ...object.Object) object.Object {
			if runtime.GOOS != "windows" {
				return newError("OSbgChange func supports only windows")
			}

			if len(args) != 1 {
				return newError("OSbgChange func has 1 args, got=%d", len(args))
			}

			changePath := syscall.StringToUTF16Ptr(config.Path + args[0].(*object.String).Value)
			currentPath := make([]uint16, syscall.MAX_PATH)

			proc := syscall.NewLazyDLL("user32.dll").NewProc("SystemParametersInfoW")

			proc.Call(
				0x0073, // SPI_GETDESKWALLPAPER
				syscall.MAX_PATH,
				uintptr(unsafe.Pointer(&currentPath[0])),
				0,
			)

			proc.Call(
				20, // SPI_SETDESKWALLPAPER
				0,
				uintptr(unsafe.Pointer(changePath)),
				0x01,
			)

			var n int
			for n = 0; n < len(currentPath) && currentPath[n] != 0; n++ {
			}

			return &object.String{Value: string(utf16.Decode(currentPath[:n]))}
		},
	},
	"OSbgReturn": {
		Fn: func(args ...object.Object) object.Object {
			if runtime.GOOS != "windows" {
				return newError("OSbgReturn func supports only windows")
			}

			if len(args) != 1 {
				return newError("OSbgChange func has 1 args, got=%d", len(args))
			}

			proc := syscall.NewLazyDLL("user32.dll").NewProc("SystemParametersInfoW")

			changePath := syscall.StringToUTF16Ptr(args[0].(*object.String).Value)
			currentPath := make([]uint16, syscall.MAX_PATH)

			proc.Call(
				0x0073, // SPI_GETDESKWALLPAPER
				syscall.MAX_PATH,
				uintptr(unsafe.Pointer(&currentPath[0])),
				0,
			)

			var n int
			for n = 0; n < len(currentPath) && currentPath[n] != 0; n++ {
			}

			if string(utf16.Decode(currentPath[:n])) != args[0].(*object.String).Value {
				proc.Call(
					20, // SPI_SETDESKWALLPAPER
					0,
					uintptr(unsafe.Pointer(changePath)),
					0x01,
				)
			}

			return NULL
		},
	},
	/*-----------win32----------*/
	"MessageBox": {
		Fn: func(args ...object.Object) object.Object {
			if runtime.GOOS != "windows" {
				return newError("MessageBox func supports only windows")
			}

			if len(args) != 2 {
				return newError("MessageBox func has 2 args, got=%d", len(args))
			}

			title, ok := args[0].(*object.String)
			if !ok {
				return newError("MessageBox func Title args is not string")
			}

			contents, ok := args[1].(*object.String)
			if !ok {
				return newError("MessageBox func Contents args is not string")
			}

			syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW").Call(
				0,
				uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(contents.Value))),
				uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(title.Value))),
				0,
			)

			return NULL
		},
	},
	/*-----------util-----------*/
	"Character": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("Character func has 2 args, got=%d", len(args))
			}

			name, ok := args[0].(*object.String)
			if !ok {
				return newError("Character func name args is not string")
			}

			color, ok := args[1].(*object.String)
			if !ok {
				return newError("Character func color args is not string")
			}

			hex := make([]int64, 3)
			switch color.Value[:1] {
			case "#":
				hex[0], _ = strconv.ParseInt(color.Value[1:3], 16, 32)
				hex[1], _ = strconv.ParseInt(color.Value[3:5], 16, 32)
				hex[2], _ = strconv.ParseInt(color.Value[5:7], 16, 32)
			default:
				return newError("Color support hex code")
			}

			c := core.CreateColor(int(hex[0]), int(hex[1]), int(hex[2]))

			return &object.Character{
				Name: &object.String{
					Value: name.Value,
				},
				Color: &c,
			}
		},
	},
	"GoSite": {
		Fn: func(args ...object.Object) object.Object {
			if runtime.GOOS == "windows" {
				cmd := exec.Command("cmd", "/C", "start", "/max", args[0].(*object.String).Value)
				cmd.Run()
			}

			return NULL
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
