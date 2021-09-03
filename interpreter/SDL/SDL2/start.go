package main

/*
#cgo LDFLAGS: -L./lib -lSDL2 -lSDL2main

#include <include/SDL.h>

#define SDL_LoadBMP(file)   SDL_LoadBMP_RW(SDL_RWFromFile(file, "rb"), 1)
#define SDL_BlitSurface SDL_UpperBlit

Uint32 eventType(SDL_Event event)
{
	return event.type;
}

SDL_Keycode key(SDL_Event event)
{
	return event.key.keysym.sym;
}
*/
import "C"
import (
	"fmt"
	"runtime"
	"unsafe"
)

const (
	KEY_PRESS_SURFACE_DEFAULT = 0
	KEY_PRESS_SURFACE_UP      = 1
	KEY_PRESS_SURFACE_DOWN    = 2
	KEY_PRESS_SURFACE_LEFT    = 3
	KEY_PRESS_SURFACE_RIGHT   = 4
	KEY_PRESS_SURFACE_TOTAL   = 5
)

var (
	window           *C.SDL_Window
	screenSurface    *C.SDL_Surface
	keyPressSurfaces [KEY_PRESS_SURFACE_TOTAL]*C.SDL_Surface
	currentSurface   *C.SDL_Surface
)

const (
	SCREEN_WIDTH  = 640
	SCREEN_HEIGHT = 480
)

func init() {
	runtime.LockOSThread()
}

func SDLInit() bool {
	success := true

	title := C.CString("SDL2")
	defer C.free(unsafe.Pointer(title))

	if int(C.SDL_Init(C.SDL_INIT_VIDEO)) < 0 {
		fmt.Println("error")
		success = false
	} else {
		window = C.SDL_CreateWindow(title, C.SDL_WINDOWPOS_UNDEFINED, C.SDL_WINDOWPOS_UNDEFINED, SCREEN_WIDTH, SCREEN_HEIGHT, C.SDL_WINDOW_SHOWN)
		if window == nil {
			fmt.Println("error")
			success = false
		} else {
			screenSurface = C.SDL_GetWindowSurface(window)
		}
	}

	return success
}

func loadSurface(path string) *C.SDL_Surface {
	c_path := C.CString(path)
	defer C.free(unsafe.Pointer(c_path))

	typ := C.CString("rb")
	defer C.free(unsafe.Pointer(typ))

	loadedSurface := C.SDL_LoadBMP_RW(C.SDL_RWFromFile(c_path, typ), 1)
	if loadedSurface == nil {
		fmt.Println("error")
	}
	return loadedSurface
}

func LoadMedia() bool {
	success := true

	keyPressSurfaces[KEY_PRESS_SURFACE_DEFAULT] = loadSurface("press.bmp")
	keyPressSurfaces[KEY_PRESS_SURFACE_UP] = loadSurface("up.bmp")
	keyPressSurfaces[KEY_PRESS_SURFACE_DOWN] = loadSurface("down.bmp")
	keyPressSurfaces[KEY_PRESS_SURFACE_LEFT] = loadSurface("left.bmp")
	keyPressSurfaces[KEY_PRESS_SURFACE_RIGHT] = loadSurface("right.bmp")

	return success
}

func close() {
	C.SDL_DestroyWindow(window)
	C.SDL_Quit()
}

func Start() {
	defer close()

	if !SDLInit() {
		fmt.Println("error")
	} else {
		if !LoadMedia() {
			fmt.Println("error")
		} else {
			quit := false

			var e C.SDL_Event
			currentSurface = keyPressSurfaces[KEY_PRESS_SURFACE_DEFAULT]

			for !quit {
				for int(C.SDL_PollEvent(&e)) != 0 {
					if C.eventType(e) == C.SDL_QUIT {
						quit = true
					} else if C.eventType(e) == C.SDL_KEYDOWN {
						switch C.key(e) {
						case C.SDLK_UP:
							currentSurface = keyPressSurfaces[KEY_PRESS_SURFACE_UP]

						case C.SDLK_DOWN:
							currentSurface = keyPressSurfaces[KEY_PRESS_SURFACE_DOWN]

						case C.SDLK_LEFT:
							currentSurface = keyPressSurfaces[KEY_PRESS_SURFACE_LEFT]

						case C.SDLK_RIGHT:
							currentSurface = keyPressSurfaces[KEY_PRESS_SURFACE_RIGHT]

						default:
							currentSurface = keyPressSurfaces[KEY_PRESS_SURFACE_DEFAULT]
						}
					}
				}
				C.SDL_BlitSurface(currentSurface, nil, screenSurface, nil)
				C.SDL_UpdateWindowSurface(window)
			}
		}
	}
}
