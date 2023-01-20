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
	"RenG/RVM/src/core/globaltype"
	"RenG/RVM/src/core/obj"
)

func (g *Graphic) GetTextTexture(text, fontName string, color obj.Color) *globaltype.SDL_Texture {
	return nil
}
