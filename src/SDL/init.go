package sdl

/*
#cgo LDFLAGS: -L./lib -lSDL2 -lSDL2main -lSDL2_image -lSDL2_ttf -lSDL2_mixer

#include <include/SDL.h>
#include <include/SDL_image.h>
#include <include/SDL_ttf.h>
#include <include/SDL_mixer.h>
*/
import "C"
import (
	"fmt"
	"unsafe"
)

func SDLInit(title string, width, height int) (*SDL_Window, *SDL_Renderer) {
	if int(C.SDL_Init(SDL_INIT_VIDEO|SDL_INIT_AUDIO)) < 0 {
		fmt.Println("SDL Error")
		return nil, nil
	}

	setHint := C.CString("1")
	C.SDL_SetHint(C.CString(SDL_HINT_RENDER_SCALE_QUALITY), setHint)

	cTitle := C.CString(title)
	window := C.SDL_CreateWindow(cTitle, SDL_WINDOWPOS_UNDEFINED, SDL_WINDOWPOS_UNDEFINED, C.int(width), C.int(height), SDL_WINDOW_SHOWN)

	renderer := C.SDL_CreateRenderer((*C.SDL_Window)(window), -1, SDL_RENDERER_ACCELERATED|SDL_RENDERER_PRESENTVSYNC)

	if (C.IMG_Init(IMG_INIT_PNG) & IMG_INIT_PNG) == 0 {
		fmt.Println("SDLImage Error")
		return nil, nil
	}

	if C.Mix_OpenAudio(44100, MIX_DEFAULT_FORMAT, 2, 2048) < 0 {
		fmt.Println("SDLMixer Error")
		return nil, nil
	}

	if C.TTF_Init() < 0 {
		fmt.Println("SDLTTF Error")
		return nil, nil
	}

	C.free(unsafe.Pointer(setHint))
	C.free(unsafe.Pointer(cTitle))

	return (*SDL_Window)(window), (*SDL_Renderer)(renderer)
}

func SDLInitRenderer(window *SDL_Window) *SDL_Renderer {
	return (*SDL_Renderer)(C.SDL_CreateRenderer((*C.SDL_Window)(window), -1, SDL_RENDERER_ACCELERATED|SDL_RENDERER_PRESENTVSYNC))
}

func Close(window *SDL_Window, renderer *SDL_Renderer) {
	C.SDL_DestroyWindow((*C.SDL_Window)(window))
	C.SDL_DestroyRenderer((*C.SDL_Renderer)(renderer))

	C.SDL_Quit()
	C.IMG_Quit()
	C.TTF_Quit()
	C.Mix_Quit()
}
