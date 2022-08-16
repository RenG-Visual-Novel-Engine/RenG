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

	ChangeWidth  int
	ChangeHeight int

	Icon *core.SDL_Surface
)

var (
	Window   *core.SDL_Window
	Renderer *core.SDL_Renderer
)

var (
	StartChannel         = make(chan bool, 5)
	Event                core.SDL_Event
	MouseMotionEventChan = make(chan core.Event, 5)
	MouseDownEventChan   = make(chan core.Event, 5)
	MouseUpEventChan     = make(chan core.Event, 5)
	MouseWheelEventChan  = make(chan core.Event, 5)

	KeyDownEventChan = make(chan core.Event, 5)
	KeyUpEventChan   = make(chan core.Event, 5)
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
	LayerMutex = &sync.RWMutex{}
)

var (
	Who      string = ""
	What     string = ""
	WhoColor core.SDL_Color
	Items    *object.Array
)
