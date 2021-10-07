package main

import (
	sdl "RenG/src/SDL"
	"RenG/src/evaluator"
	"RenG/src/lexer"
	"RenG/src/object"
	"RenG/src/parser"
	"flag"
	"io/ioutil"
	"log"
	"runtime"
)

var (
	Root *string
)

var (
	code   = ""
	title  object.Object
	width  object.Object
	height object.Object
)

var (
	window   *sdl.SDL_Window
	renderer *sdl.SDL_Renderer
	event    sdl.SDL_Event
)

var (
	env      *object.Environment = object.NewEnvironment()
	errValue *object.Error

	quit = false
)

func init() {
	// r 플래그로 파일 주소를 받음
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

	go interPretation(code)

	setUp(env)

	mainLoop(errValue)
}

// 해석 단계입니다.
func interPretation(code string) {
	l := lexer.New(code)
	p := parser.New(l)
	program := p.ParseProgram()

	obj := evaluator.Eval(program, env)
	errValue, _ = obj.(*object.Error)
}

func setUp(env *object.Environment) {
R1:
	if env == nil {
		goto R1
	}

R2:
	if !env.IsHere("gui_title") || !env.IsHere("gui_width") || !env.IsHere("gui_height") {
		goto R2
	}

	title, _ = env.Get("gui_title")
	width, _ = env.Get("gui_width")
	height, _ = env.Get("gui_height")

	window, renderer = sdl.SDLInit(title.(*object.String).Value, int(width.(*object.Integer).Value), int(height.(*object.Integer).Value))
}

func mainLoop(errObject *object.Error) {
	LayerList := sdl.NewLayerList()
	TextureList := object.NewTextureList()

	go run(renderer, env, &LayerList, TextureList)

	for !quit {
		for event.PollEvent() != 0 {
			if event.EventType() == sdl.SDL_QUIT {
				quit = true
			}
		}
		renderer.SetRenderDrawColor(0xFF, 0xFF, 0xFF, 0xFF)
		renderer.RenderClear()

		for i := 0; i < len(LayerList.Layers); i++ {
			for j := 0; j < len(LayerList.Layers[i].Images); j++ {
				evaluator.LayerMutex.Lock()
				LayerList.Layers[i].Images[j].Render(renderer, nil, LayerList.Layers[i].Images[j].Xpos, LayerList.Layers[i].Images[j].Ypos)
				evaluator.LayerMutex.Unlock()
			}
		}

		renderer.RenderPresent()
	}

	sdl.Close(window, renderer)
}

func run(renderer *sdl.SDL_Renderer, env *object.Environment, layerList *sdl.LayerList, textureList *object.TextureList) {

	fontPath, _ := env.Get("gui_font")

	layerList.Layers = append(layerList.Layers, sdl.Layer{Name: "main"})
	layerList.Layers = append(layerList.Layers, sdl.Layer{Name: "screen"})

	font := sdl.OpenFont(*Root + fontPath.(*object.String).Value)

	start, ok := env.Get("start")

	if !ok {
		layerList.Layers[1].AddNewTexture(sdl.LoadFromRenderedText("Code should have start label", renderer, font, sdl.Color(0, 0, 0)))
		return
	}

	if errValue != nil {
		layerList.Layers[1].AddNewTexture(sdl.LoadFromRenderedText(errValue.Message, renderer, font, sdl.Color(0, 0, 0)))
		return
	}

	evaluator.RengEval(start.(*object.Label).Body, *Root, env, renderer, layerList, textureList)
}
