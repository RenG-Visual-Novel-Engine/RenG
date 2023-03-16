package graphic

/*
#cgo CFLAGS: -I./../../../sdl/include
#cgo LDFLAGS: -L./../../../sdl/lib -lSDL2 -lSDL2main -lSDL2_image -lSDL2_ttf

#include <SDL.h>
#include <SDL_image.h>
#include <SDL_ttf.h>
*/
import "C"
import (
	image "RenG/RVM/src/core/System/Game/Graphic/Image"
	video "RenG/RVM/src/core/System/Game/Graphic/Video"
	"RenG/RVM/src/core/globaltype"
	"RenG/RVM/src/core/obj"
	"sync"
	"time"
	"unsafe"
)

type Graphic struct {
	window   *globaltype.SDL_Window
	renderer *globaltype.SDL_Renderer
	Cursor   *C.SDL_Surface

	path          string
	width, height int

	lock     sync.Mutex
	userLock sync.Mutex
	sayLock  sync.Mutex

	// screenBps -> targetScreentextures > texture
	renderBuffer [][]struct {
		texture   *globaltype.SDL_Texture
		transform obj.Transform
	}

	Image *image.Image
	Video *video.Video
	fonts map[string]struct {
		Font        *globaltype.TTF_Font
		Size        int
		LimitPixels int
	}
	textMemPool map[string][]*globaltype.SDL_Texture
	typingFXs   map[string][]struct {
		Data []struct {
			Texture   *globaltype.SDL_Texture
			Transform obj.Transform
		}
		Duration  float64
		StartTime time.Time
		Bps       int
		Index     int
	}
	animations map[string]map[int][]struct {
		Anime *obj.Anime
		Bps   int
	}
}

func Init(window *globaltype.SDL_Window, r *globaltype.SDL_Renderer, p string, w, h int) *Graphic {
	return &Graphic{
		window:   window,
		renderer: r,
		renderBuffer: [][]struct {
			texture   *globaltype.SDL_Texture
			transform obj.Transform
		}{},
		path:   p,
		width:  w,
		height: h,
		Image:  image.Init(r),
		Video:  video.Init(),
		fonts: make(map[string]struct {
			Font        *globaltype.TTF_Font
			Size        int
			LimitPixels int
		}),
		textMemPool: make(map[string][]*globaltype.SDL_Texture),
		typingFXs: make(map[string][]struct {
			Data []struct {
				Texture   *globaltype.SDL_Texture
				Transform obj.Transform
			}
			Duration  float64
			StartTime time.Time
			Bps       int
			Index     int
		}),
		animations: make(map[string]map[int][]struct {
			Anime *obj.Anime
			Bps   int
		}),
	}
}

func (g *Graphic) Close() {
	C.SDL_DestroyRenderer((*C.SDL_Renderer)(g.renderer))

	g.Image.Close()
	g.Video.Close()

	C.SDL_FreeSurface(g.Cursor)
}

func (g *Graphic) Update() {
	g.UpdateAnimation()
	g.UpdateTypingFX()
}

func (g *Graphic) RegisterCursor(path string) {
	cpath := C.CString(g.path + path)
	defer C.free(unsafe.Pointer(cpath))

	surface := C.IMG_Load(cpath)
	cursor := C.SDL_CreateColorCursor(surface, 0, 0)
	C.SDL_SetCursor(cursor)
	g.Cursor = surface
}

func (g *Graphic) RegisterImages(images map[string]string) {
	for name, path := range images {
		g.Image.RegisterImage(name, g.path+path)
	}
}

func (g *Graphic) RegisterVideos(videos map[string]string) {
	for name, path := range videos {
		g.Video.Register(name, g.path+path, g.renderer)
	}
}

func (g *Graphic) RegisterFonts(
	fonts map[string]struct {
		Path        string
		Size        int
		LimitPixels int
	},
) {
	for name, font := range fonts {
		cpath := C.CString(g.path + font.Path)
		defer C.free(unsafe.Pointer(cpath))
		g.fonts[name] = struct {
			Font        *globaltype.TTF_Font
			Size        int
			LimitPixels int
		}{
			(*globaltype.TTF_Font)(C.TTF_OpenFont(cpath, C.int(font.Size))),
			font.Size,
			font.LimitPixels,
		}
	}
}

func (g *Graphic) SayLock() {
	g.sayLock.Lock()
}

func (g *Graphic) SayUnlock() {
	g.sayLock.Unlock()
}

func (g *Graphic) RenderLock() {
	g.userLock.Lock()
}

func (g *Graphic) RenderUnlock() {
	g.userLock.Unlock()
}
