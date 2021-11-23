package main

import (
	"RenG/src/config"
	"RenG/src/core"
	"RenG/src/lang/ast"
	"RenG/src/lang/evaluator"
	"RenG/src/lang/lexer"
	"RenG/src/lang/object"
	"RenG/src/lang/parser"
	"RenG/src/lang/token"
	"RenG/src/reng"
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
	initRenG()
	interPretation(config.Code)
	mainLoop(errValue)
	clear()
}

func initRenG() {
	config.Path = *PATH + "\\"

	dir, err := ioutil.ReadDir(config.Path)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range dir {
		if file.Name() == "config.rgo" {
			f, err := ioutil.ReadFile(config.Path + "config.rgo")
			if err != nil {
				panic(err)
			}
			interPretation(string(f))
		} else if file.Name()[len(file.Name())-3:] == "rgo" && file.Name() != "config.rgo" {
			f, err := ioutil.ReadFile(config.Path + file.Name())
			if err != nil {
				panic(err)
			}
			config.Code += string(f) + "\n"
		}
	}

	title, _ := env.Get("config_title")
	config.Title = title.(*object.String).Value

	width, _ := env.Get("config_width")
	config.Width = int(width.(*object.Integer).Value)

	height, _ := env.Get("config_height")
	config.Height = int(height.(*object.Integer).Value)

	config.ChangeWidth = config.Width
	config.ChangeHeight = config.Height

	icon, ok := env.Get("config_icon")
	if ok {
		config.Icon = core.IMGLoad(config.Path + icon.(*object.String).Value)
	}

	config.Window, config.Renderer = core.SDLInit(config.Title, config.Width, config.Height, config.Icon)
}

func interPretation(code string) {
	l := lexer.New(code)
	p := parser.New(l)
	program := p.ParseProgram()

	obj := evaluator.Eval(program, env)
	errValue, _ = obj.(*object.Error)
}

func mainLoop(errObject *object.Error) {

	go run(env)

	for !config.Quit {
		for config.Event.PollEvent() != 0 {
			switch config.Event.EventType() {
			case core.SDL_QUIT:
				config.Quit = true
			case core.SDL_WINDOWEVENT:
				switch config.Event.WindowEventType() {
				case core.SDL_WINDOWEVENT_SIZE_CHANGED:
					config.ChangeWidth, config.ChangeHeight = config.Event.ChangeWidthAndHeight()
				}
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
		config.Renderer.SetRenderDrawColor(0xFF, 0xFF, 0xFF, 255)

		config.LayerMutex.Lock()
		for i := 0; i < len(config.LayerList.Layers); i++ {
			for j := 0; j < len(config.LayerList.Layers[i].Images); j++ {
				config.LayerList.Layers[i].Images[j].Render(config.Renderer, config.Width, config.Height, config.ChangeWidth, config.ChangeHeight)
			}
		}
		config.LayerMutex.Unlock()

		config.Renderer.RenderPresent()
	}
}

func run(env *object.Environment) {
	config.LayerList.Layers = append(config.LayerList.Layers, core.Layer{Name: "error"})
	config.LayerList.Layers = append(config.LayerList.Layers, core.Layer{Name: "main"})
	config.LayerList.Layers = append(config.LayerList.Layers, core.Layer{Name: "screen"})

	config.ChannelList.NewChannel("music", -1)
	config.ChannelList.NewChannel("sound", 0)
	config.ChannelList.NewChannel("voice", 1)

	fontPath, _ := env.Get("config_font")
	config.MainFont = core.OpenFont(config.Path + fontPath.(*object.String).Value)

	if errValue != nil {
		config.LayerMutex.Lock()
		config.LayerList.Layers[0].AddNewTexture(
			config.MainFont.LoadFromRenderedText(errValue.Message,
				config.Renderer,
				config.Width, config.Height,
				core.CreateColor(0, 0, 0),
			),
		)
		config.LayerMutex.Unlock()
		return
	}

	main_menu, ok := env.Get("main_menu")
	if !ok {
		config.LayerMutex.Lock()
		config.LayerList.Layers[0].AddNewTexture(
			config.MainFont.LoadFromRenderedText("Could not find the screen main_menu for your code.",
				config.Renderer,
				config.Width, config.Height,
				core.CreateColor(0, 0, 0),
			),
		)
		config.LayerMutex.Unlock()
		return
	}
	config.Main_Menu = main_menu.(*object.Screen)

	start, ok := env.Get("start")
	if !ok {
		config.LayerMutex.Lock()
		config.LayerList.Layers[0].AddNewTexture(
			config.MainFont.LoadFromRenderedText("Could not find the entry point for your code.",
				config.Renderer,
				config.Width, config.Height,
				core.CreateColor(0, 0, 0),
			),
		)
		config.LayerMutex.Unlock()
		return
	}

	config.Start = start.(*object.Label)

	var (
		result    object.Object
		jumpLabel *object.JumpLabel
		label     object.Object
	)

MAIN:

	reng.RengEval(&ast.ShowExpression{
		Token: token.Token{
			Type:    token.SHOW,
			Literal: "SHOW",
		},
		Name: config.Main_Menu.Name,
	}, env)

	<-config.StartChannel

	result = reng.RengEval(start.(*object.Label).Body, env)

	if result == nil {
		return
	} else if _, ok = result.(*object.ReturnValue); ok {
		config.StopAllChannel()
		config.DeleteAllLayerTexture()
		goto MAIN
	} else if jumpLabel, ok = result.(*object.JumpLabel); !ok {
		return
	}

	label, ok = env.Get(jumpLabel.Label.Value)

	for ok {
		result = reng.RengEval(label.(*object.Label).Body, env)

		if result == nil {
			return
		} else if _, ok = result.(*object.ReturnValue); ok {
			config.StopAllChannel()
			config.DeleteAllLayerTexture()
			goto MAIN
		} else if jumpLabel, ok = result.(*object.JumpLabel); !ok {
			return
		}

		label, ok = env.Get(jumpLabel.Label.Value)
	}
}

func clear() {
	config.TextureList.DestroyAll()
	config.MusicList.FreaAll()
	config.ChunkList.FreeAll()
	core.Close(config.Window, config.Renderer)
}
