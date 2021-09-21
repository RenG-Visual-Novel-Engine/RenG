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

func SDLInit(title string, width, height int) (bool, *SDL_Window, *SDL_Renderer) {
	if int(C.SDL_Init(SDL_INIT_VIDEO|SDL_INIT_AUDIO)) < 0 {
		fmt.Println("SDL Error")
		return false, nil, nil
	}

	setHint := C.CString("1")
	C.SDL_SetHint(C.CString(SDL_HINT_RENDER_SCALE_QUALITY), setHint)
	C.free(unsafe.Pointer(setHint))

	cTitle := C.CString(title)
	window := C.SDL_CreateWindow(cTitle, SDL_WINDOWPOS_UNDEFINED, SDL_WINDOWPOS_UNDEFINED, C.int(width), C.int(height), SDL_WINDOW_SHOWN)
	C.free(unsafe.Pointer(cTitle))

	renderer := C.SDL_CreateRenderer(window, -1, SDL_RENDERER_ACCELERATED|SDL_RENDERER_PRESENTVSYNC)
	C.SDL_SetRenderDrawColor(renderer, 0x00, 0x00, 0x00, 0x00)

	if (C.IMG_Init(IMG_INIT_PNG) & IMG_INIT_PNG) == 0 {
		fmt.Println("SDLImage Error")
		return false, nil, nil
	}

	if C.Mix_OpenAudio(44100, MIX_DEFAULT_FORMAT, 2, 2048) < 0 {
		fmt.Println("SDLMixer Error")
		return false, nil, nil
	}

	return true, (*SDL_Window)(window), (*SDL_Renderer)(renderer)
}
