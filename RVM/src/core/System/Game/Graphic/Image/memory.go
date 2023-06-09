package image

/*
#cgo CFLAGS: -I./../../../../sdl/include
#cgo LDFLAGS: -L./../../../../sdl/lib -lSDL2 -lSDL2main -lSDL2_image

#include <SDL.h>
#include <SDL_image.h>
*/
import "C"
import "RenG/RVM/src/core/globaltype"

func (i *Image) FreeSurface(sur *C.SDL_Surface) {
	C.SDL_FreeSurface(sur)
}

func (i *Image) FreeTexture(tex *globaltype.SDL_Texture) {
	C.SDL_DestroyTexture((*C.SDL_Texture)(tex))
}
