package event

/*
#cgo CFLAGS: -I./../../../sdl/include
#cgo LDFLAGS: -L./../../../sdl/lib -lSDL2 -lSDL2main -lSDL2_image -lSDL2_ttf -lSDL2_mixer

#include <SDL.h>
*/
import "C"

type ButtonEvent struct {
	Down  func(*EVENT_MouseButtonDown)
	Up    func(*EVENT_MouseButtonUp)
	Hover func(*EVENT_MouseMotion)
}

type EVENT_MouseButtonDown struct {
}

type EVENT_MouseButtonUp struct {
}

type EVENT_MouseMotion struct {
}

type KeyEvent struct {
	Down, Up func(*EVENT_Key)
}

type EVENT_Key struct {
	KeyCode int
}

const (
	// Event Type
	SDL_QUIT            = C.SDL_QUIT
	SDL_KEYDOWN         = C.SDL_KEYDOWN
	SDL_KEYUP           = C.SDL_KEYUP
	SDL_MOUSEMOTION     = C.SDL_MOUSEMOTION
	SDL_MOUSEBUTTONDOWN = C.SDL_MOUSEBUTTONDOWN
	SDL_MOUSEBUTTONUP   = C.SDL_MOUSEBUTTONUP
	SDL_MOUSEWHEEL      = C.SDL_MOUSEWHEEL

	// SDL Key Code
	SDLK_0 = C.SDLK_0
	SDLK_1 = C.SDLK_1
	SDLK_2 = C.SDLK_2
	SDLK_3 = C.SDLK_3
	SDLK_4 = C.SDLK_4
	SDLK_5 = C.SDLK_5
	SDLK_6 = C.SDLK_6
	SDLK_7 = C.SDLK_7
	SDLK_8 = C.SDLK_8
	SDLK_9 = C.SDLK_9

	SDLK_a = C.SDLK_a
	SDLK_b = C.SDLK_b
	SDLK_c = C.SDLK_c
	SDLK_d = C.SDLK_d
	SDLK_e = C.SDLK_e
	SDLK_f = C.SDLK_f
	SDLK_g = C.SDLK_g
	SDLK_h = C.SDLK_h
	SDLK_i = C.SDLK_i
	SDLK_j = C.SDLK_j
	SDLK_k = C.SDLK_k
	SDLK_l = C.SDLK_l
	SDLK_m = C.SDLK_m
	SDLK_n = C.SDLK_n
	SDLK_o = C.SDLK_o
	SDLK_p = C.SDLK_p
	SDLK_q = C.SDLK_q
	SDLK_r = C.SDLK_r
	SDLK_s = C.SDLK_s
	SDLK_t = C.SDLK_t
	SDLK_u = C.SDLK_u
	SDLK_v = C.SDLK_v
	SDLK_w = C.SDLK_w
	SDLK_x = C.SDLK_x
	SDLK_y = C.SDLK_y
	SDLK_z = C.SDLK_z
)
