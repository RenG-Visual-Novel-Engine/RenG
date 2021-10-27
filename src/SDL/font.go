package sdl

/*
#cgo LDFLAGS: -L./lib -lSDL2 -lSDL2main -lSDL2_ttf

#include <include/SDL.h>
#include <include/SDL_ttf.h>
*/
import "C"
import (
	"fmt"
	"unsafe"
)

func OpenFont(path string) *TTF_Font {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	f := C.TTF_OpenFont(cPath, 28)
	if f == nil {
		fmt.Println("failed TTF OpenFont")
	}

	return (*TTF_Font)(f)
}

func (f *TTF_Font) LoadFromRenderedText(text string, renderer *SDL_Renderer, color SDL_Color) *SDL_Texture {
	cText := C.CString(text)
	defer C.free(unsafe.Pointer(cText))

	textSurface := C.TTF_RenderUTF8_Blended((*C.TTF_Font)(f), cText, (C.SDL_Color)(color))
	if textSurface == nil {
		fmt.Println("failed renderText")
	}

	t := C.SDL_CreateTextureFromSurface((*C.SDL_Renderer)(renderer), textSurface)
	if t == nil {
		fmt.Println("texture nil")
	}

	// TODO
	texture := &SDL_Texture{
		Texture: t,
		Xpos:    (1280 - int(textSurface.w)) / 2,
		Ypos:    550,
		Width:   int(textSurface.w),
		Height:  int(textSurface.h),
		Alpha:   255,
		Degree:  0,
	}

	C.SDL_FreeSurface(textSurface)

	return texture
}
