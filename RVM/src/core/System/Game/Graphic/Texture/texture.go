package texture

/*
#cgo CFLAGS: -I./../../../../sdl/include
#cgo LDFLAGS: -L./../../../../sdl/lib -lSDL2 -lSDL2main -lSDL2_image -lSDL2_ttf

#include <SDL.h>
#include <SDL_image.h>
#include <SDL_ttf.h>
*/
import "C"
import "RenG/RVM/src/core/globaltype"

func TextureAlphaChange(t *globaltype.SDL_Texture, value int) {
	C.SDL_SetTextureAlphaMod((*C.SDL_Texture)(t), C.uchar(value))
}
