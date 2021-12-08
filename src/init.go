package main

import (
	"RenG/src/config"
	"RenG/src/core"
	"RenG/src/lang/evaluator"
	"RenG/src/lang/lexer"
	"RenG/src/lang/object"
	"RenG/src/lang/parser"
	"flag"
	"io/ioutil"
	"log"
)

var (
	env      *object.Environment = object.NewEnvironment()
	errValue *object.Error
)

func Init() bool {
	f := flag.String("r", "", "root")
	flag.Parse()

	config.Path = *f + "\\"

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
			if err, ok := InterPretation(string(f)); ok {
				errValue = err
				return false
			}
		} else if file.Name()[len(file.Name())-3:] == "rgo" && file.Name() != "config.rgo" {
			f, err := ioutil.ReadFile(config.Path + file.Name())
			if err != nil {
				panic(err)
			}
			config.Code += string(f) + "\n"
		}
	}

	title, ok := env.Get("config_title")
	if !ok {
		return false
	}

	width, ok := env.Get("config_width")
	if !ok {
		return false
	}

	height, ok := env.Get("config_height")
	if !ok {
		return false
	}

	config.Title = title.(*object.String).Value
	config.Width = int(width.(*object.Integer).Value)
	config.Height = int(height.(*object.Integer).Value)

	config.ChangeWidth = config.Width
	config.ChangeHeight = config.Height

	icon, ok := env.Get("config_icon")
	if ok {
		config.Icon = core.IMGLoad(config.Path + icon.(*object.String).Value)
	}

	config.Window, config.Renderer = core.SDLInit(config.Title, config.Width, config.Height, config.Icon)
	if config.Window == nil || config.Renderer == nil {
		return false
	}

	if err, ok := InterPretation(config.Code); ok {
		errValue = err
		return false
	}

	config.LayerList.Layers = append(config.LayerList.Layers, core.Layer{Name: "error"})
	config.LayerList.Layers = append(config.LayerList.Layers, core.Layer{Name: "main"})
	config.LayerList.Layers = append(config.LayerList.Layers, core.Layer{Name: "screen"})

	config.ChannelList.NewChannel("music", -1)
	config.ChannelList.NewChannel("sound", 0)
	config.ChannelList.NewChannel("voice", 1)

	core.SetVolume(-1, 32)

	fontPath, _ := env.Get("config_font")
	config.MainFont = core.OpenFont(config.Path + fontPath.(*object.String).Value)

	if errValue != nil {
		config.LayerMutex.Lock()
		config.LayerList.Layers[0].AddNewTexture(
			config.MainFont.LoadFromRenderedText(errValue.Message,
				config.Renderer,
				uint(config.Width),
				core.CreateColor(0xFF, 0xFF, 0xFF),
				255,
				0,
			),
		)
		config.LayerMutex.Unlock()
	}

	return true
}

func InterPretation(code string) (*object.Error, bool) {
	l := lexer.New(code)
	p := parser.New(l)
	program := p.ParseProgram()

	obj := evaluator.Eval(program, env)
	err, ok := obj.(*object.Error)

	return err, ok
}
