package main

import (
	"RenG/src/config"
	"RenG/src/core"
	"RenG/src/lang/ast"
	"RenG/src/lang/object"
	"RenG/src/lang/token"
	"RenG/src/reng"
)

func Run() {
	main_menu, ok := env.Get("main_menu")
	if !ok {
		config.LayerMutex.Lock()
		config.LayerList.Layers[0].AddNewTexture(
			config.MainFont.LoadFromRenderedText("Could not find the screen main_menu for your code.",
				config.Renderer,
				uint(config.Width),
				core.CreateColor(0xFF, 0xFF, 0xFF),
				255,
				0,
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
				uint(config.Width),
				core.CreateColor(0xFF, 0xFF, 0xFF),
				255,
				0,
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
	} else if _, ok := result.(*object.ReturnValue); ok {
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
