package image

/*
#cgo CFLAGS: -I./../../../../sdl/include
#cgo LDFLAGS: -L./../../../../sdl/lib -lSDL2 -lSDL2main -lSDL2_image

#include <SDL.h>
#include <SDL_image.h>
*/
import "C"
import "RenG/RVM/src/core/globaltype"

func (i *Image) ChangeTextureAlpha(t *globaltype.SDL_Texture, alpha int) {
	C.SDL_SetTextureAlphaMod((*C.SDL_Texture)(t), C.uchar(alpha))
}
