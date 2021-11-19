package core

/*
#cgo CFLAGS: -I./sdl/include
#cgo LDFLAGS: -L./sdl/lib -lSDL2 -lSDL2main -lSDL2_image

#include <SDL.h>
#include <SDL_image.h>


SDL_PixelFormat* surfaceFormat(SDL_Surface* surface)
{
	return surface->format;
}
*/
import "C"
import (
	"unsafe"
)

func (t *SDL_Texture) Render(renderer *SDL_Renderer, w, h, cw, ch int) {
	renderQuad := CreateRect(ResizeInt(w, cw, t.Xpos), ResizeInt(h, ch, t.Ypos), ResizeInt(w, cw, t.Width), ResizeInt(w, cw, t.Height))

	t.SetAlpha(t.Alpha)

	C.SDL_RenderCopyEx((*C.SDL_Renderer)(renderer), t.Texture, nil, &renderQuad, C.double(t.Degree), nil, SDL_FLIP_NONE)
}

func (renderer *SDL_Renderer) SetRenderDrawColor(r, g, b, a uint8) {
	C.SDL_SetRenderDrawColor((*C.SDL_Renderer)(renderer), C.uchar(r), C.uchar(g), C.uchar(b), C.uchar(a))
}

func (r *SDL_Renderer) RenderClear() {
	C.SDL_RenderClear((*C.SDL_Renderer)(r))
}

func (r *SDL_Renderer) RenderPresent() {
	C.SDL_RenderPresent((*C.SDL_Renderer)(r))
}

func (r *SDL_Renderer) LoadFromFile(root string) (*SDL_Texture, bool) {
	cRoot := C.CString(root)
	defer C.free(unsafe.Pointer(cRoot))

	loadedSurface := C.IMG_Load(cRoot)

	if loadedSurface == nil {
		return nil, false
	}

	C.SDL_SetColorKey(loadedSurface, SDL_TRUE, C.SDL_MapRGB(C.surfaceFormat(loadedSurface), 0, 0xFF, 0xFF))

	newTexture := &SDL_Texture{
		Texture: C.SDL_CreateTextureFromSurface((*C.SDL_Renderer)(r), loadedSurface),
		Width:   int(loadedSurface.w),
		Height:  int(loadedSurface.h),
		Alpha:   255,
		Degree:  0,
	}

	newTexture.SetBlendMode()
	C.SDL_FreeSurface(loadedSurface)

	return newTexture, true
}

func (r *SDL_Renderer) CreateTexture(width, height int) *SDL_Texture {
	var texture SDL_Texture

	texture.Alpha = 255
	texture.Degree = 0
	texture.Width = width
	texture.Height = height
	texture.Texture = C.SDL_CreateTexture(
		(*C.SDL_Renderer)(r),
		SDL_PIXELFORMAT_YV12,
		SDL_TEXTUREACCESS_STREAMING,
		C.int(width),
		C.int(height),
	)

	return &texture
}
