package event

/*
#cgo CFLAGS: -I./../../../sdl/include
#cgo LDFLAGS: -L./../../../sdl/lib -lSDL2 -lSDL2main -lSDL2_image -lSDL2_ttf -lSDL2_mixer

#include <SDL.h>
*/
import "C"

type MouseClickEvent struct {
	Down func(*EVENT_MouseButton)
	Up   func(*EVENT_MouseButton)
}

type BarEvent struct {
	IsNowDown bool
	Down      func(*EVENT_MouseButton) bool
	Up        func(*EVENT_MouseButton)
	Scroll    func(*EVENT_MouseMotion)
}

type ButtonEvent struct {
	IsNowDown bool
	Down      func(*EVENT_MouseButton) bool
	Up        func(*EVENT_MouseButton)
	Hover     func(*EVENT_MouseMotion)
}

type EVENT_MouseButton struct {
	X, Y, Button int
}

type EVENT_MouseMotion struct {
	X, Y, Xrel, Yrel int
}

type KeyEvent struct {
	Down, Up func(*EVENT_Key)
}

type EVENT_Key struct {
	KeyCode int
}

const (
	// Event Type
	RENG_QUIT    = C.SDL_QUIT
	RENG_KEYDOWN = C.SDL_KEYDOWN
	RENG_KEYUP   = C.SDL_KEYUP

	RENG_MOUSEMOTION     = C.SDL_MOUSEMOTION
	RENG_MOUSEBUTTONDOWN = C.SDL_MOUSEBUTTONDOWN
	RENG_MOUSEBUTTONUP   = C.SDL_MOUSEBUTTONUP
	RENG_MOUSEWHEEL      = C.SDL_MOUSEWHEEL

	RENG_BUTTON_LEFT   = C.SDL_BUTTON_LEFT
	RENG_BUTTON_RIGHT  = C.SDL_BUTTON_RIGHT
	RENG_BUTTON_MIDDLE = C.SDL_BUTTON_MIDDLE

	// SDL Key Code
	RENGK_0 = C.SDLK_0
	RENGK_1 = C.SDLK_1
	RENGK_2 = C.SDLK_2
	RENGK_3 = C.SDLK_3
	RENGK_4 = C.SDLK_4
	RENGK_5 = C.SDLK_5
	RENGK_6 = C.SDLK_6
	RENGK_7 = C.SDLK_7
	RENGK_8 = C.SDLK_8
	RENGK_9 = C.SDLK_9

	RENGK_a = C.SDLK_a
	RENGK_b = C.SDLK_b
	RENGK_c = C.SDLK_c
	RENGK_d = C.SDLK_d
	RENGK_e = C.SDLK_e
	RENGK_f = C.SDLK_f
	RENGK_g = C.SDLK_g
	RENGK_h = C.SDLK_h
	RENGK_i = C.SDLK_i
	RENGK_j = C.SDLK_j
	RENGK_k = C.SDLK_k
	RENGK_l = C.SDLK_l
	RENGK_m = C.SDLK_m
	RENGK_n = C.SDLK_n
	RENGK_o = C.SDLK_o
	RENGK_p = C.SDLK_p
	RENGK_q = C.SDLK_q
	RENGK_r = C.SDLK_r
	RENGK_s = C.SDLK_s
	RENGK_t = C.SDLK_t
	RENGK_u = C.SDLK_u
	RENGK_v = C.SDLK_v
	RENGK_w = C.SDLK_w
	RENGK_x = C.SDLK_x
	RENGK_y = C.SDLK_y
	RENGK_z = C.SDLK_z

	RENGK_ESCAPE = C.SDLK_ESCAPE
)
