package sdl

/*
#cgo LDFLAGS: -L./lib -lSDL2 -lSDL2main -lSDL2_image -lSDL2_ttf -lSDL2_mixer

#include <include/SDL.h>
#include <include/SDL_image.h>
#include <include/SDL_ttf.h>
#include <include/SDL_mixer.h>

SDL_PixelFormat* surfaceFormat(SDL_Surface* surface)
{
	return surface->format;
}
*/
import "C"
import (
	"unsafe"
)

func SetRenderDrawColor(renderer *SDL_Renderer, r, g, b, a uint8) {
	C.SDL_SetRenderDrawColor((*C.SDL_Renderer)(renderer), C.uchar(r), C.uchar(g), C.uchar(b), C.uchar(a))
}

func RenderClear(renderer *SDL_Renderer) {
	C.SDL_RenderClear((*C.SDL_Renderer)(renderer))
}

func RenderPresent(renderer *SDL_Renderer) {
	C.SDL_RenderPresent((*C.SDL_Renderer)(renderer))
}

func RenderCopy(renderer *SDL_Renderer, texture *SDL_Texture) {
	C.SDL_RenderCopy((*C.SDL_Renderer)(renderer), (*C.SDL_Texture)(texture), nil, nil)
}

func LoadFromFile(root string, renderer *SDL_Renderer) (*SDL_Texture, bool) {
	cRoot := C.CString(root)
	loadedSurface := C.IMG_Load(cRoot)

	if loadedSurface == nil {
		return nil, false
	}

	C.SDL_SetColorKey(loadedSurface, SDL_TRUE, C.SDL_MapRGB(C.surfaceFormat(loadedSurface), 0, 0xFF, 0xFF))

	newTexture := C.SDL_CreateTextureFromSurface((*C.SDL_Renderer)(renderer), loadedSurface)

	if newTexture == nil {
		return nil, false
	}

	C.SDL_FreeSurface(loadedSurface)

	C.free(unsafe.Pointer(cRoot))
	return (*SDL_Texture)(newTexture), true
}
