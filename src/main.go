package main

import (
	sdl "RenG/src/SDL"
	"RenG/src/config"
	"RenG/src/evaluator"
	"RenG/src/lexer"
	"RenG/src/object"
	"RenG/src/parser"
	"RenG/src/rengeval"
	"flag"
	"io/ioutil"
	"log"
	"runtime"
)

var (
	Root *string
)

var (
	env      *object.Environment = object.NewEnvironment()
	errValue *object.Error
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
			config.Code += string(tem) + "\n"
		}
	}

	go interPretation(config.Code)

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

	title, _ := env.Get("gui_title")
	config.Title = title.(*object.String).Value

	width, _ := env.Get("gui_width")
	config.Width = int(width.(*object.Integer).Value)

	height, _ := env.Get("gui_height")
	config.Height = int(height.(*object.Integer).Value)

	config.Window, config.Renderer = sdl.SDLInit(config.Title, config.Width, config.Height)
}

func mainLoop(errObject *object.Error) {

	go run(env)

	for !config.Quit {
		for config.Event.PollEvent() != 0 {
			switch config.Event.EventType() {
			case sdl.SDL_QUIT:
				config.Quit = true
			case sdl.SDL_WINDOWEVENT:
			case sdl.SDL_KEYDOWN:
				// default:
				//	EventChan <- sdl.Event{Event: event, Type: uint32(event.EventType())}
			}
		}

		config.Renderer.SetRenderDrawColor(0xFF, 0xFF, 0xFF, 0xFF)
		config.Renderer.RenderClear()

		for i := 0; i < len(config.LayerList.Layers); i++ {
			for j := 0; j < len(config.LayerList.Layers[i].Images); j++ {
				rengeval.LayerMutex.Lock()
				config.LayerList.Layers[i].Images[j].Render(config.Renderer, nil, config.LayerList.Layers[i].Images[j].Xpos, config.LayerList.Layers[i].Images[j].Ypos)
				rengeval.LayerMutex.Unlock()
			}
		}

		config.Renderer.RenderPresent()
	}

	config.TextureList.DestroyAll()
	sdl.Close(config.Window, config.Renderer)
}

func run(env *object.Environment) {

	fontPath, _ := env.Get("gui_font")
	config.MainFont = sdl.OpenFont(*Root + fontPath.(*object.String).Value)

	config.LayerList.Layers = append(config.LayerList.Layers, sdl.Layer{Name: "error"})
	config.LayerList.Layers = append(config.LayerList.Layers, sdl.Layer{Name: "main"})
	config.LayerList.Layers = append(config.LayerList.Layers, sdl.Layer{Name: "screen"})

	start, ok := env.Get("start")

	if !ok {
		config.LayerList.Layers[0].AddNewTexture(config.MainFont.LoadFromRenderedText("Could not find the entry point for your code.", config.Renderer, sdl.Color(0, 0, 0)))
		return
	}

	if errValue != nil {
		config.LayerList.Layers[0].AddNewTexture(config.MainFont.LoadFromRenderedText(errValue.Message, config.Renderer, sdl.Color(0, 0, 0)))
		return
	}

	var (
		result    object.Object
		jumpLabel *object.JumpLabel
		label     object.Object
	)

	result = rengeval.RengEval(start.(*object.Label).Body, *Root, env)

	if result == nil {
		return
	}

	if jumpLabel, ok = result.(*object.JumpLabel); !ok {
		return
	}

R:
	if label, ok = env.Get(jumpLabel.Label.Value); ok {
		result = rengeval.RengEval(label.(*object.Label).Body, *Root, env)

		if result == nil {
			return
		}

		if jumpLabel, ok = result.(*object.JumpLabel); !ok {
			return
		}

		goto R
	}
}
