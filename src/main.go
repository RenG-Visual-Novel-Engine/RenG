package main

import (
	sdl "RenG/src/SDL"
	"RenG/src/evaluator"
	"RenG/src/lexer"
	"RenG/src/object"
	"RenG/src/parser"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"runtime"
)

var (
	Root *string
)

var (
	code = ""
)

var (
	event sdl.SDL_Event
	quit  = false
)

func init() {
	// root 플래그로 파일 주소를 받음
	Root = flag.String("r", "", "root")
	flag.Parse()
	if flag.NFlag() == 0 {
		flag.Usage()
	}
	runtime.LockOSThread()
}

func main() {
	// 해당 경로에 있는 파일들을 가져옴
	dir, err := ioutil.ReadDir(*Root)
	if err != nil {
		log.Fatal(err)
	}

	// 파일 이름 뒤에 확장자가 rgo인 파일들을 읽어들이고 code에 집어넣음
	for _, file := range dir {
		if file.Name()[len(file.Name())-3:] == "rgo" {
			tem, err := ioutil.ReadFile(*Root + "\\" + file.Name())
			if err != nil {
				panic(err)
			}
			code += string(tem) + "\n"
		}
	}

	obj, env := interPretation(code)
	if errValue, ok := obj.(*object.Error); ok {
		fmt.Println(errValue.Message)
	}

	_, ok := env.Get("start")
	if !ok {
		fmt.Println("Code should have start label")
	}

	if !run(env) {
		fmt.Println("Fail")
	}
}

// 해석 단계입니다.
func interPretation(code string) (object.Object, *object.Environment) {
	l := lexer.New(code)
	p := parser.New(l)
	program := p.ParseProgram()
	env := object.NewEnvironment()

	return evaluator.Eval(program, env), env
}

// SDL Run
func run(env *object.Environment) bool {

	title, ok1 := env.Get("title")
	width, ok2 := env.Get("width")
	height, ok3 := env.Get("height")

	if !ok1 || !ok2 || !ok3 {
		return false
	}

	ok, window, renderer := sdl.SDLInit(title.(*object.String).Value, int(width.(*object.Integer).Value), int(height.(*object.Integer).Value))
	if !ok {
		return false
	}

	LayerList := sdl.NewLayerList()
	LayerList.Layers = append(LayerList.Layers, sdl.Layer{Name: "main"})
	texture, _ := sdl.LoadFromFile(*Root+"\\gui\\loading.png", renderer)
	LayerList.Layers[0].Images = append(LayerList.Layers[0].Images, texture)

	for !quit {
		for sdl.PollEvent(&event) != 0 {
			if sdl.EventType(event) == sdl.SDL_QUIT {
				quit = true
			}
		}
		sdl.SetRenderDrawColor(renderer, 0xFF, 0xFF, 0xFF, 0xFF)
		sdl.RenderClear(renderer)

		for i := 0; i < len(LayerList.Layers); i++ {
			if LayerList.Layers[i].Images[0] == nil {
				continue
			}
			for j := 0; j < len(LayerList.Layers[i].Images); j++ {
				sdl.Render(renderer, LayerList.Layers[i].Images[j])
			}
		}

		sdl.RenderPresent(renderer)
	}

	sdl.Close(window, renderer)
	return true
}
