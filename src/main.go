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
	PATH *string
)

var (
	env      *object.Environment = object.NewEnvironment()
	errValue *object.Error
)

func init() {
	// r 플래그로 파일 주소를 받음
	PATH = flag.String("r", "", "root")
	flag.Parse()
	if flag.NFlag() == 0 {
		flag.Usage()
	}
	runtime.LockOSThread()
}

func main() {
	config.Path = *PATH

	dir, err := ioutil.ReadDir(config.Path)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range dir {
		if file.Name() == "config.rgo" {
			f, err := ioutil.ReadFile(config.Path + "\\config.rgo")
			if err != nil {
				panic(err)
			}
			interPretation(string(f))
		} else if file.Name()[len(file.Name())-3:] == "rgo" && file.Name() != "config.rgo" {
			f, err := ioutil.ReadFile(config.Path + "\\" + file.Name())
			if err != nil {
				panic(err)
			}
			config.Code += string(f) + "\n"
		}
	}

	setUp()

	interPretation(config.Code)

	mainLoop(errValue)
}

func interPretation(code string) {
	l := lexer.New(code)
	p := parser.New(l)
	program := p.ParseProgram()

	obj := evaluator.Eval(program, env)
	errValue, _ = obj.(*object.Error)
}

func setUp() {
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
			case sdl.SDL_KEYDOWN:
				//  default:
				//	EventChan <- sdl.Event{Event: event, Type: uint32(event.EventType())}
			}
		}

		config.Renderer.SetRenderDrawColor(0xFF, 0xFF, 0xFF, 0xFF)
		config.Renderer.RenderClear()

		for i := 0; i < len(config.LayerList.Layers); i++ {
			for j := 0; j < len(config.LayerList.Layers[i].Images); j++ {
				rengeval.LayerMutex.Lock()
				config.LayerList.Layers[i].Images[j].Render(config.Renderer, nil)
				rengeval.LayerMutex.Unlock()
			}
		}

		config.Renderer.RenderPresent()
	}

	config.TextureList.DestroyAll()
	config.MusicList.FreaAll()
	config.ChunkList.FreeAll()
	sdl.Close(config.Window, config.Renderer)
}

func run(env *object.Environment) {

	fontPath, _ := env.Get("gui_font")
	config.MainFont = sdl.OpenFont(config.Path + fontPath.(*object.String).Value)

	config.LayerList.Layers = append(config.LayerList.Layers, sdl.Layer{Name: "error"})
	config.LayerList.Layers = append(config.LayerList.Layers, sdl.Layer{Name: "main"})
	config.LayerList.Layers = append(config.LayerList.Layers, sdl.Layer{Name: "screen"})

	config.ChannelList.NewChannel("music", -1)
	config.ChannelList.NewChannel("sound", 0)
	config.ChannelList.NewChannel("voice", 1)

	start, ok := env.Get("start")

	if !ok {
		config.LayerList.Layers[0].AddNewTexture(config.MainFont.LoadFromRenderedText("Could not find the entry point for your code.", config.Renderer, sdl.CreateColor(0, 0, 0)))
		return
	}

	if errValue != nil {
		config.LayerList.Layers[0].AddNewTexture(config.MainFont.LoadFromRenderedText(errValue.Message, config.Renderer, sdl.CreateColor(0, 0, 0)))
		return
	}

	var (
		result    object.Object
		jumpLabel *object.JumpLabel
		label     object.Object
	)

	result = rengeval.RengEval(start.(*object.Label).Body, env)

	if result == nil {
		return
	}

	if jumpLabel, ok = result.(*object.JumpLabel); !ok {
		return
	}

R:
	if label, ok = env.Get(jumpLabel.Label.Value); ok {
		result = rengeval.RengEval(label.(*object.Label).Body, env)

		if result == nil {
			return
		}

		if jumpLabel, ok = result.(*object.JumpLabel); !ok {
			return
		}

		goto R
	}
}
