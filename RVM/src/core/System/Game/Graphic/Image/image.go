package image

/*
#cgo CFLAGS: -I./../../../../sdl/include
#cgo LDFLAGS: -L./../../../../sdl/lib -lSDL2 -lSDL2main -lSDL2_image

#include <SDL.h>
#include <SDL_image.h>
*/
import "C"
import (
	"RenG/RVM/src/core/globaltype"
	"sync"
	"unsafe"
)

type Image struct {
	lock sync.Mutex

	// Private : 텍스쳐 생성에 필요
	renderer *globaltype.SDL_Renderer

	// Private : 한번 생성되면 변경은 불가능하다.
	images map[string]struct {
		texture *globaltype.SDL_Texture
		width   int
		height  int
	}
}

func Init(r *globaltype.SDL_Renderer) *Image {
	return &Image{
		renderer: r,
		images: make(map[string]struct {
			texture *globaltype.SDL_Texture
			width   int
			height  int
		}),
	}
}

func (i *Image) Close() {
	for _, i := range i.images {
		C.SDL_DestroyTexture((*C.SDL_Texture)(i.texture))
	}
}

func (i *Image) RegisterImage(name string, path string) {
	t, w, h := i.loadImage(path)
	i.images[name] = struct {
		texture *globaltype.SDL_Texture
		width   int
		height  int
	}{
		t, w, h,
	}
}

func (i *Image) loadImage(path string) (*globaltype.SDL_Texture, int, int) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))

	surface := C.IMG_Load(cpath)
	if surface == nil {
		panic("존재하지 않는 경로로 이미지를 불러왔습니다.")
	}
	defer C.SDL_FreeSurface(surface)

	t := C.SDL_CreateTextureFromSurface((*C.SDL_Renderer)(i.renderer), surface)

	C.SDL_SetTextureBlendMode(t, C.SDL_BLENDMODE_BLEND)
	return (*globaltype.SDL_Texture)(t), int(surface.w), int(surface.h)
}
