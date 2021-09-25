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

	title, ok1 := env.Get("gui_title")
	width, ok2 := env.Get("gui_width")
	height, ok3 := env.Get("gui_height")
	sayImagePath, ok4 := env.Get("screen_say_image")
	fontPath, ok5 := env.Get("gui_font")
	testText, ok6 := env.Get("test_text")
	bgImagePath, ok7 := env.Get("gui_bg_image")

	if !ok1 || !ok2 || !ok3 || !ok4 || !ok5 || !ok6 || !ok7 {
		return false
	}

	ok, window, renderer := sdl.SDLInit(title.(*object.String).Value, int(width.(*object.Integer).Value), int(height.(*object.Integer).Value))
	if !ok {
		return false
	}

	LayerList := sdl.NewLayerList()
	LayerList.Layers = append(LayerList.Layers, sdl.Layer{Name: "main"})
	LayerList.Layers = append(LayerList.Layers, sdl.Layer{Name: "screen"})

	bgTexture, _ := sdl.LoadFromFile(*Root+bgImagePath.Inspect(), renderer)
	LayerList.Layers[0].AddNewTexture(bgTexture)

	sayTexture, _ := sdl.LoadFromFile(*Root+sayImagePath.Inspect(), renderer)
	LayerList.Layers[1].AddNewTexture(sayTexture)

	font := sdl.OpenFont(*Root + fontPath.(*object.String).Value)
	textTexture := sdl.LoadFromRenderedText(testText.Inspect(), renderer, font, sdl.Color(0, 0, 0))
	LayerList.Layers[1].AddNewTexture(textTexture)

	for !quit {
		for event.PollEvent() != 0 {
			if event.EventType() == sdl.SDL_QUIT {
				quit = true
			}
		}
		renderer.SetRenderDrawColor(0x00, 0x00, 0x00, 0xFF)
		renderer.RenderClear()

		for i := 0; i < len(LayerList.Layers); i++ {
			for j := 0; j < len(LayerList.Layers[i].Images); j++ {
				LayerList.Layers[i].Images[j].Render(renderer, nil, LayerList.Layers[i].Images[j].Xpos, LayerList.Layers[i].Images[j].Ypos)
			}
		}

		renderer.RenderPresent()
	}

	sdl.Close(window, renderer)
	return true
}
