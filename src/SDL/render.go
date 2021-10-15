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

SDL_Rect createRect(int x, int y, int w, int h)
{
	SDL_Rect Quad = { x, y, w, h };
	return Quad;
}
*/
import "C"
import (
	"unsafe"
)

func (renderer *SDL_Renderer) SetRenderDrawColor(r, g, b, a uint8) {
	C.SDL_SetRenderDrawColor((*C.SDL_Renderer)(renderer), C.uchar(r), C.uchar(g), C.uchar(b), C.uchar(a))
}

func (r *SDL_Renderer) RenderClear() {
	C.SDL_RenderClear((*C.SDL_Renderer)(r))
}

func (r *SDL_Renderer) RenderPresent() {
	C.SDL_RenderPresent((*C.SDL_Renderer)(r))
}

func (t *SDL_Texture) Render(renderer *SDL_Renderer, clip *SDL_Rect, x, y int) {
	renderQuad := C.createRect(C.int(x), C.int(y), C.int(t.Width), C.int(t.Height))

	if clip != nil {
		renderQuad.w = clip.w
		renderQuad.h = clip.h
	}

	C.SDL_RenderCopy((*C.SDL_Renderer)(renderer), t.Texture, (*C.SDL_Rect)(clip), &renderQuad)
}

func (r *SDL_Renderer) LoadFromFile(root string) (*SDL_Texture, bool) {
	cRoot := C.CString(root)
	loadedSurface := C.IMG_Load(cRoot)

	if loadedSurface == nil {
		return nil, false
	}

	C.SDL_SetColorKey(loadedSurface, SDL_TRUE, C.SDL_MapRGB(C.surfaceFormat(loadedSurface), 0, 0xFF, 0xFF))

	newTexure := &SDL_Texture{Texture: C.SDL_CreateTextureFromSurface((*C.SDL_Renderer)(r), loadedSurface), Width: int(loadedSurface.w), Height: int(loadedSurface.h)}

	C.SDL_FreeSurface(loadedSurface)

	C.free(unsafe.Pointer(cRoot))
	return newTexure, true
}
