package sdl

/*
#cgo LDFLAGS: -L./lib -lSDL2 -lSDL2main -lSDL2_image -lSDL2_ttf -lSDL2_mixer

#include <include/SDL.h>
#include <include/SDL_image.h>
#include <include/SDL_ttf.h>
#include <include/SDL_mixer.h>

SDL_Color colorSet(int r, int g, int b)
{
	SDL_Color textColor = { 0, 0, 0 };
	return textColor;
}
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

func Color(r, g, b int) SDL_Color {
	return (SDL_Color)(C.colorSet(C.int(r), C.int(g), C.int(b)))
}

func LoadFromRenderedText(text string, renderer *SDL_Renderer, f *TTF_Font, color SDL_Color) *SDL_Texture {
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

	texture := &SDL_Texture{Texture: t, Xpos: 300, Ypos: 550, Width: int(textSurface.w), Height: int(textSurface.h)}

	C.SDL_FreeSurface(textSurface)

	return texture
}
