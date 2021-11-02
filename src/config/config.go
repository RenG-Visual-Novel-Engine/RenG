package config

import (
	"RenG/src/core"
	"RenG/src/lang/object"
	"sync"
)

var (
	Path string
	Code string
)

var (
	Quit bool
)

var (
	Title  string
	Width  int
	Height int

	Icon *core.SDL_Surface
)

var (
	Window   *core.SDL_Window
	Renderer *core.SDL_Renderer
)

var (
	StartChannel         = make(chan bool)
	Event                core.SDL_Event
	MouseMotionEventChan = make(chan core.Event, 5)
	MouseDownEventChan   = make(chan core.Event, 5)
	MouseUpEventChan     = make(chan core.Event, 5)
	MouseWheelEventChan  = make(chan core.Event, 5)
)

var (
	Main_Menu *object.Screen
	Say       *object.Screen
	Choice    *object.Screen
)

var (
	Start *object.Label
)

var (
	Env = object.NewEnvironment()
)

var (
	MainFont *core.TTF_Font
)

var (
	LayerList   = core.NewLayerList()
	ChannelList = core.NewChannelList()
	MusicList   = object.NewMusicList()
	ChunkList   = object.NewChunkList()
	VideoList   = object.NewVideoList()
	TextureList = object.NewTextureList()
)

var (
	ShowTextureIndex = make([]*core.SDL_Texture, 0)
	ShowIndex        = 0

	ScreenHasIndex     = make(map[string][]int)
	ScreenTextureIndex = make([]*core.SDL_Texture, 0)
	ScreenIndex        = 0
)

var (
	LayerMutex = &sync.RWMutex{}
)
