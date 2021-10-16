package config

import (
	sdl "RenG/src/SDL"
	"RenG/src/object"
)

var (
	Code string
)

var (
	Quit bool
)

var (
	Title  string
	Width  int
	Height int
)

var (
	Window   *sdl.SDL_Window
	Renderer *sdl.SDL_Renderer
)

var (
	Event     sdl.SDL_Event
	EventChan = make(chan sdl.Event)
)

var (
	MainFont *sdl.TTF_Font
)

var (
	LayerList   = sdl.NewLayerList()
	TextureList = object.NewTextureList()
)

var (
	ShowTextureIndex = make([]*sdl.SDL_Texture, 20)
	ShowIndex        = 0
)
