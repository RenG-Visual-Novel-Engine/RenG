package core

/*
#cgo CFLAGS: -I./sdl/include
#cgo LDFLAGS: -L./sdl/lib -lSDL2 -lSDL2main -lSDL2_image -lSDL2_ttf

#include <SDL.h>
#include <SDL_ttf.h>

*/
import "C"
import (
	"unsafe"
)

func OpenFont(path string) *TTF_Font {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	f := C.TTF_OpenFont(cPath, 28)
	if f == nil {
		return nil
	}

	return (*TTF_Font)(f)
}

func (f *TTF_Font) LoadFromRenderedText(text string, renderer *SDL_Renderer, limitWidth uint, color SDL_Color, alpha uint8, degree float64) *SDL_Texture {
	cText := C.CString(text)
	defer C.free(unsafe.Pointer(cText))

	textSurface := C.TTF_RenderUTF8_Blended_Wrapped((*C.TTF_Font)(f), cText, (C.SDL_Color)(color), C.uint(limitWidth))
	if textSurface == nil {
		return nil
	}

	t := C.SDL_CreateTextureFromSurface((*C.SDL_Renderer)(renderer), textSurface)
	if t == nil {
		return nil
	}

	texture := &SDL_Texture{
		Texture:     t,
		TextureType: TEXTTEXTURE,
		TextTexture: Text{
			Text:   text,
			Width:  int(textSurface.w),
			Height: int(textSurface.h),
			Alpha:  alpha,
			Degree: degree,
		},
	}

	C.SDL_FreeSurface(textSurface)

	return texture
}

func (f *TTF_Font) ChangeTextColor(texture *SDL_Texture, renderer *SDL_Renderer, text string, color SDL_Color) {
	cText := C.CString(text)
	defer C.free(unsafe.Pointer(cText))

	textSurface := C.TTF_RenderUTF8_Blended_Wrapped((*C.TTF_Font)(f), cText, (C.SDL_Color)(color), C.uint(texture.TextTexture.Width))
	if textSurface == nil {
		return
	}

	t := C.SDL_CreateTextureFromSurface((*C.SDL_Renderer)(renderer), textSurface)
	if t == nil {
		return
	}

	texture.Texture = t

	C.SDL_FreeSurface(textSurface)
}
