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
	sprite "RenG/RVM/src/core/System/Game/Graphic/Sprite"
	video "RenG/RVM/src/core/System/Game/Graphic/Video"
	"RenG/RVM/src/core/globaltype"
	"RenG/RVM/src/core/obj"
	"fmt"
	"sync"
	"unsafe"
)

type Graphic struct {
	renderer *globaltype.SDL_Renderer

	path string

	// screenBps -> targetScreentextures > texture
	renderBuffer [][]struct {
		texture   *globaltype.SDL_Texture
		transform obj.Transform
	}
	screenBps map[string]int

	lock sync.Mutex

	screens map[string]*obj.Screen
	images  map[string]struct {
		texture *globaltype.SDL_Texture
		width   int
		height  int
	}
	sprites map[string]*sprite.Sprite
	videos  *video.Video
}

func Init(r *globaltype.SDL_Renderer, p string) *Graphic {
	return &Graphic{
		renderer: r,
		path:     p,
		renderBuffer: [][]struct {
			texture   *globaltype.SDL_Texture
			transform obj.Transform
		}{},
		screenBps: make(map[string]int),
		screens:   make(map[string]*obj.Screen),
		images: make(map[string]struct {
			texture *globaltype.SDL_Texture
			width   int
			height  int
		}),
		sprites: make(map[string]*sprite.Sprite),
		videos:  video.Init(),
	}
}

func (g *Graphic) Close() {
	C.SDL_DestroyRenderer((*C.SDL_Renderer)(g.renderer))

	for _, i := range g.images {
		C.SDL_DestroyTexture((*C.SDL_Texture)(i.texture))
	}
}

// key : name, value : ui.Screen
func (g *Graphic) RegisterScreens(screens map[string]*obj.Screen) {
	g.lock.Lock()
	defer g.lock.Unlock()

	for name, screen := range screens {
		g.screens[name] = screen
	}
}

// key : name, value : path
func (g *Graphic) RegisterImages(images map[string]string) {
	g.lock.Lock()
	defer g.lock.Unlock()

	for name, path := range images {
		t, w, h := g.loadImage(g.path + path)
		g.images[name] = struct {
			texture *globaltype.SDL_Texture
			width   int
			height  int
		}{
			t, w, h,
		}
	}
}

func (g *Graphic) loadImage(path string) (*globaltype.SDL_Texture, int, int) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))

	surface := C.IMG_Load(cpath)
	if surface == nil {
		fmt.Println(path)
		panic("존재하지 않는 경로로 이미지를 불러왔습니다.")
	}
	defer C.SDL_FreeSurface(surface)

	t := C.SDL_CreateTextureFromSurface((*C.SDL_Renderer)(g.renderer), surface)

	C.SDL_SetTextureBlendMode(t, C.SDL_BLENDMODE_BLEND)
	return (*globaltype.SDL_Texture)(t), int(surface.w), int(surface.h)
}

func (g *Graphic) RegisterVideos(videos map[string]string) {
	g.lock.Lock()
	defer g.lock.Unlock()

	for name, path := range videos {
		g.videos.VideoInit(name, g.path+path, g.renderer)
	}
}
