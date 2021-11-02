package core

/*
#cgo CFLAGS: -I./sdl/include
#cgo LDFLAGS: -L./sdl/lib -lSDL2 -lSDL2main -lSDL2_image

#include <SDL.h>
#include <SDL_image.h>
*/
import "C"
import "unsafe"

func IMGLoad(path string) *SDL_Surface {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	surface := C.IMG_Load(cPath)

	return (*SDL_Surface)(surface)
}

func (t *SDL_Texture) SetBlendMode() {
	C.SDL_SetTextureBlendMode((*C.SDL_Texture)(t.Texture), SDL_BLENDMODE_BLEND)
}

func (t *SDL_Texture) SetAlpha(alpha uint8) {
	C.SDL_SetTextureAlphaMod((*C.SDL_Texture)(t.Texture), C.uchar(alpha))
}

func (t *SDL_Texture) DestroyTexture() {
	C.SDL_DestroyTexture(t.Texture)
}
