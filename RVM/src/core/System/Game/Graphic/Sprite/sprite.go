package sprite

/*
#cgo CFLAGS: -I./../../../../sdl/include
#cgo LDFLAGS: -L./../../../../sdl/lib -lSDL2 -lSDL2main -lSDL2_image

#include <SDL.h>
#include <SDL_image.h>

SDL_Rect* Sprite_CreateRect(int x, int y, int w, int h)
{
	SDL_Rect* Quad = (SDL_Rect*)malloc(sizeof(SDL_Rect));
	Quad->x = x;
	Quad->y = y;
	Quad->w = w;
	Quad->h = h;
	return Quad;
}

void Sprite_FreeRect(SDL_Rect* r)
{
	free(r);
}
*/
import "C"
import (
	"RenG/RVM/src/core/globaltype"
	"log"
	"sync"
	"unsafe"
)

type Sprite struct {
	lock     sync.Mutex
	renderer *globaltype.SDL_Renderer
	data     map[string]struct {
		surface *C.SDL_Surface
		width   int
		height  int
	}
	member map[string][]*globaltype.SDL_Texture
}

func Init(r *globaltype.SDL_Renderer) *Sprite {
	return &Sprite{
		renderer: r,
		data: make(map[string]struct {
			surface *C.SDL_Surface
			width   int
			height  int
		}),
		member: make(map[string][]*globaltype.SDL_Texture),
	}
}

func (s *Sprite) Close() {
	for _, d := range s.data {
		C.SDL_FreeSurface((*C.SDL_Surface)(d.surface))
	}
}

func (s *Sprite) CreateSpriteImages(name string, count, xsize, ysize int) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.data[name].width%xsize != 0 || s.data[name].height%ysize != 0 {
		log.Fatalln("Create Sprite Images Error")
		return
	}

	xoffset, yoffset := 0, 0

	for i := 0; i < count; i++ {
		new := C.SDL_CreateRGBSurface(C.uint(0), C.int(xsize), C.int(ysize), C.int(32), C.uint(0), C.uint(0), C.uint(0), C.uint(0))
		defer C.SDL_FreeSurface(new)

		convert_new := C.SDL_ConvertSurfaceFormat(new, C.SDL_PIXELFORMAT_ARGB8888, 0)

		C.SDL_SetSurfaceBlendMode(convert_new, C.SDL_BLENDMODE_BLEND)

		src_rect := C.Sprite_CreateRect(C.int(xoffset), C.int(yoffset), C.int(xsize), C.int(ysize))
		dst_rect := C.Sprite_CreateRect(C.int(0), C.int(0), C.int(0), C.int(0))

		defer C.Sprite_FreeRect(src_rect)
		defer C.Sprite_FreeRect(dst_rect)

		C.SDL_BlitSurface(s.data[name].surface, src_rect, convert_new, dst_rect)

		t := C.SDL_CreateTextureFromSurface((*C.SDL_Renderer)(s.renderer), convert_new)
		C.SDL_SetTextureBlendMode(t, C.SDL_BLENDMODE_BLEND)
		s.member[name] = append(s.member[name], (*globaltype.SDL_Texture)(t))

		xoffset += xsize
		if s.data[name].width < xoffset+xsize {
			xoffset = 0
			yoffset += ysize
		}
	}
}

func (s *Sprite) DeleteSpriteImages(name string) {
	for _, t := range s.member[name] {
		C.SDL_DestroyTexture((*C.SDL_Texture)(t))
	}
}

func (s *Sprite) RegisterSprite(name, path string) {
	t, x, y := s.loadImage(path)
	s.data[name] = struct {
		surface *C.SDL_Surface
		width   int
		height  int
	}{t, x, y}
}

func (s *Sprite) loadImage(path string) (*C.SDL_Surface, int, int) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))

	surface := C.IMG_Load(cpath)
	if surface == nil {
		panic("존재하지 않는 경로로 이미지를 불러왔습니다.")
	}

	C.SDL_SetSurfaceBlendMode(surface, C.SDL_BLENDMODE_NONE)

	return surface, int(surface.w), int(surface.h)
}

func (s *Sprite) GetSpriteSize(name string) int {
	s.lock.Lock()
	defer s.lock.Unlock()

	return len(s.member[name])
}

func (s *Sprite) GetSpriteImage(name string, Index int) *globaltype.SDL_Texture {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.member[name][Index]
}
