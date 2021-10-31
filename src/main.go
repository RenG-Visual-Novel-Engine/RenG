package main

import (
	"RenG/src/config"
	"RenG/src/core"
	"RenG/src/lang/evaluator"
	"RenG/src/lang/lexer"
	"RenG/src/lang/object"
	"RenG/src/lang/parser"
	"RenG/src/reng/screen"
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

	config.Window, config.Renderer = core.SDLInit(config.Title, config.Width, config.Height)
}

func mainLoop(errObject *object.Error) {

	go run(env)

	for !config.Quit {
		for config.Event.PollEvent() != 0 {
			switch config.Event.EventType() {
			case core.SDL_QUIT:
				config.Quit = true
			case core.SDL_MOUSEMOTION:
				config.Event.HandleEvent(core.SDL_MOUSEMOTION, config.MouseMotionEventChan)
			case core.SDL_MOUSEBUTTONDOWN:
				config.Event.HandleEvent(core.SDL_MOUSEBUTTONDOWN, config.MouseDownEventChan)
			case core.SDL_MOUSEBUTTONUP:
				config.Event.HandleEvent(core.SDL_MOUSEBUTTONUP, config.MouseUpEventChan)
			case core.SDL_MOUSEWHEEL:
				config.Event.HandleEvent(core.SDL_MOUSEWHEEL, config.MouseWheelEventChan)
			}
		}

		config.Renderer.RenderClear()

		for i := 0; i < len(config.LayerList.Layers); i++ {
			for j := 0; j < len(config.LayerList.Layers[i].Images); j++ {
				config.LayerMutex.Lock()
				config.LayerList.Layers[i].Images[j].Render(config.Renderer, nil)
				config.LayerMutex.Unlock()
			}
		}

		config.Renderer.RenderPresent()
	}

	config.TextureList.DestroyAll()
	config.MusicList.FreaAll()
	config.ChunkList.FreeAll()
	core.Close(config.Window, config.Renderer)
}

func run(env *object.Environment) {

	main_menu, _ := env.Get("main_menu")
	config.Main_Menu = main_menu.(*object.Screen)

	start, _ := env.Get("start")
	config.Start = start.(*object.Label)

	fontPath, _ := env.Get("gui_font")
	config.MainFont = core.OpenFont(config.Path + fontPath.(*object.String).Value)

	config.LayerList.Layers = append(config.LayerList.Layers, core.Layer{Name: "error"})
	config.LayerList.Layers = append(config.LayerList.Layers, core.Layer{Name: "main"})

	config.ChannelList.NewChannel("music", -1)
	config.ChannelList.NewChannel("sound", 0)
	config.ChannelList.NewChannel("voice", 1)

	if errValue != nil {
		config.LayerList.Layers[0].AddNewTexture(config.MainFont.LoadFromRenderedText(errValue.Message, config.Renderer, core.CreateColor(0, 0, 0)))
		return
	}

	screen.ScreenEval(config.Main_Menu.Body, env)

	/*
			start, ok := env.Get("start")

			if !ok {
				config.LayerList.Layers[0].AddNewTexture(config.MainFont.LoadFromRenderedText("Could not find the entry point for your code.", config.Renderer, core.CreateColor(0, 0, 0)))
				return
			}

			var (
				result    object.Object
				jumpLabel *object.JumpLabel
				label     object.Object
			)

			result = reng.RengEval(start.(*object.Label).Body, env)

			if result == nil {
				return
			}

			if jumpLabel, ok = result.(*object.JumpLabel); !ok {
				return
			}

		R:
			if label, ok = env.Get(jumpLabel.Label.Value); ok {
				result = reng.RengEval(label.(*object.Label).Body, env)

				if result == nil {
					return
				}

				if jumpLabel, ok = result.(*object.JumpLabel); !ok {
					return
				}

				goto R
			}
	*/
}
