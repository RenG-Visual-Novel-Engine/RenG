package core

/*
#cgo CFLAGS: -I./sdl/include
#cgo LDFLAGS: -L./sdl/lib -lSDL2 -lSDL2main -lSDL2_image -lSDL2_ttf -lSDL2_mixer

#include <SDL.h>
#include <SDL_image.h>
#include <SDL_ttf.h>
#include <SDL_mixer.h>
*/
import "C"
import (
	"unsafe"
)

func SDLInit(title string, width, height int, icon *SDL_Surface) (*SDL_Window, *SDL_Renderer) {
	if C.SDL_Init(SDL_INIT_VIDEO|SDL_INIT_AUDIO|SDL_INIT_TIMER) < 0 {
		return nil, nil
	}

	setHint := C.CString("1")
	defer C.free(unsafe.Pointer(setHint))

	C.SDL_SetHint(C.CString(SDL_HINT_RENDER_SCALE_QUALITY), setHint)

	cTitle := C.CString(title)
	defer C.free(unsafe.Pointer(cTitle))

	window := C.SDL_CreateWindow(cTitle, SDL_WINDOWPOS_UNDEFINED, SDL_WINDOWPOS_UNDEFINED, C.int(width), C.int(height), SDL_WINDOW_SHOWN|SDL_WINDOW_RESIZABLE)
	if icon != nil {
		C.SDL_SetWindowIcon((*C.SDL_Window)(window), (*C.SDL_Surface)(icon))
	}
	renderer := C.SDL_CreateRenderer((*C.SDL_Window)(window), -1, SDL_RENDERER_ACCELERATED|SDL_RENDERER_PRESENTVSYNC|SDL_RENDERER_TARGETTEXTURE)

	if C.IMG_Init(IMG_INIT_PNG) < 0 {
		return nil, nil
	}

	if C.Mix_OpenAudio(44100, MIX_DEFAULT_FORMAT, 2, 2048) < 0 {
		return nil, nil
	}

	if C.TTF_Init() < 0 {
		return nil, nil
	}

	return (*SDL_Window)(window), (*SDL_Renderer)(renderer)
}

func Close(window *SDL_Window, renderer *SDL_Renderer) {
	C.SDL_DestroyWindow((*C.SDL_Window)(window))
	C.SDL_DestroyRenderer((*C.SDL_Renderer)(renderer))

	C.SDL_Quit()
	C.IMG_Quit()
	C.TTF_Quit()
	C.Mix_Quit()
}
