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
	animation "RenG/RVM/src/core/System/Game/Graphic/Animation"
	sprite "RenG/RVM/src/core/System/Game/Graphic/Sprite"
	texture "RenG/RVM/src/core/System/Game/Graphic/Texture"
	video "RenG/RVM/src/core/System/Game/Graphic/Video"
	"RenG/RVM/src/core/globaltype"
	"RenG/RVM/src/core/obj"
	"sync"
	"time"
	"unsafe"
)

type Graphic struct {
	renderer *globaltype.SDL_Renderer
	Cursor   *C.SDL_Surface

	path string

	lock sync.Mutex

	// screenBps -> targetScreentextures > texture
	renderBuffer [][]struct {
		texture   *globaltype.SDL_Texture
		transform obj.Transform
	}

	images map[string]struct {
		texture *globaltype.SDL_Texture
		width   int
		height  int
	}
	sprites    map[string]*sprite.Sprite
	videos     *video.Video
	animations []struct {
		Anime *animation.Anime
		Bps   int
		Index int
	}
	fonts map[string]*globaltype.TTF_Font
}

func Init(r *globaltype.SDL_Renderer, p string) *Graphic {
	return &Graphic{
		renderer: r,
		renderBuffer: [][]struct {
			texture   *globaltype.SDL_Texture
			transform obj.Transform
		}{},
		path: p,
		images: make(map[string]struct {
			texture *globaltype.SDL_Texture
			width   int
			height  int
		}),
		sprites: make(map[string]*sprite.Sprite),
		videos:  video.Init(),
		animations: []struct {
			Anime *animation.Anime
			Bps   int
			Index int
		}{},
		fonts: make(map[string]*globaltype.TTF_Font),
	}
}

func (g *Graphic) Close() {
	C.SDL_DestroyRenderer((*C.SDL_Renderer)(g.renderer))

	for _, i := range g.images {
		C.SDL_DestroyTexture((*C.SDL_Texture)(i.texture))
	}

	g.videos.Close()

	C.SDL_FreeSurface(g.Cursor)
}

func (g *Graphic) Update() {
	// Animation

	for n, anime := range g.animations {
		s := time.Since(anime.Anime.Time).Seconds()

		if s < anime.Anime.StartTime {
			continue
		}

		if s-anime.Anime.StartTime >= anime.Anime.Duration {
			if !anime.Anime.Loop {
				if anime.Anime.End != nil {
					anime.Anime.End()
				}
				g.animations = append(g.animations[:n], g.animations[n+1:]...)
				continue
			}
			anime.Anime.Time = time.Now()
			continue
		}

		switch anime.Anime.Type {
		case animation.ANIME_ALPHA:
			g.videos.Lock()
			texture.TextureAlphaChange(g.renderBuffer[anime.Bps][anime.Index].texture, anime.Anime.Curve((s-anime.Anime.StartTime)/anime.Anime.Duration))
			g.videos.Unlock()
			continue
		case animation.ANIME_ROTATE:
			g.videos.Lock()
			g.renderBuffer[anime.Bps][anime.Index].transform.Rotate = anime.Anime.Curve((s - anime.Anime.StartTime) / anime.Anime.Duration)
			g.videos.Unlock()
			continue
		}
	}
}

// key : name, value : path
func (g *Graphic) RegisterImages(images map[string]string) {
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

func (g *Graphic) RegisterCursor(path string) {
	cpath := C.CString(g.path + path)
	defer C.free(unsafe.Pointer(cpath))

	surface := C.IMG_Load(cpath)
	cursor := C.SDL_CreateColorCursor(surface, 0, 0)
	C.SDL_SetCursor(cursor)
	g.Cursor = surface
}

func (g *Graphic) loadImage(path string) (*globaltype.SDL_Texture, int, int) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))

	surface := C.IMG_Load(cpath)
	if surface == nil {
		panic("존재하지 않는 경로로 이미지를 불러왔습니다.")
	}
	defer C.SDL_FreeSurface(surface)

	t := C.SDL_CreateTextureFromSurface((*C.SDL_Renderer)(g.renderer), surface)

	C.SDL_SetTextureBlendMode(t, C.SDL_BLENDMODE_BLEND)
	return (*globaltype.SDL_Texture)(t), int(surface.w), int(surface.h)
}

func (g *Graphic) RegisterVideos(videos map[string]string) {
	for name, path := range videos {
		g.videos.VideoInit(name, g.path+path, g.renderer)
	}
}

func (g *Graphic) RegisterFonts(
	fonts map[string]struct {
		Path string
		Size int
	},
) {
	for name, font := range fonts {
		cpath := C.CString(font.Path)
		defer C.free(unsafe.Pointer(cpath))

		g.fonts[name] = (*globaltype.TTF_Font)(C.TTF_OpenFont(cpath, C.int(font.Size)))
	}
}
