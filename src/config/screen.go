package config

import (
	"RenG/src/core"
	"RenG/src/lang/object"
)

type Screen struct {
	First int
	Count int
}

var (
	Main_Menu *object.Screen
	Say       *object.Screen
	Choice    *object.Screen
)

var (
	ShowTextureIndex = make([]*core.SDL_Texture, 0)
	ShowIndex        = 0

	ScreenAllIndex     = make(map[string]Screen)
	ScreenTextureIndex = make([]*core.SDL_Texture, 0)
	ScreenIndex        = 0
	ScreenPriority     = make([]string, 0)

	SelectMenuIndex = make(chan int, 1)
)
