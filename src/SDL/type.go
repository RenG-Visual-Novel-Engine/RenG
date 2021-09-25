package sdl

/*
#cgo LDFLAGS: -L./lib -lSDL2 -lSDL2main -lSDL2_image -lSDL2_ttf -lSDL2_mixer

#include <include/SDL.h>
#include <include/SDL_image.h>
#include <include/SDL_ttf.h>
#include <include/SDL_mixer.h>
*/
import "C"

type SDL_Window C.SDL_Window
type SDL_Renderer C.SDL_Renderer
type SDL_Event C.SDL_Event

type SDL_Texture struct {
	Texture *C.SDL_Texture
	Xpos    int
	Ypos    int
	Width   int
	Height  int
}
type SDL_Rect C.SDL_Rect
type SDL_Color C.SDL_Color

type Mix_Music C.Mix_Music
type Mix_Chunk C.Mix_Chunk

type TTF_Font C.TTF_Font

const (
	SDL_INIT_TIMER = C.SDL_INIT_TIMER
	SDL_INIT_AUDIO = C.SDL_INIT_AUDIO
	SDL_INIT_VIDEO = C.SDL_INIT_VIDEO
)

const (
	SDL_WINDOW_SHOWN        = C.SDL_WINDOW_SHOWN
	SDL_WINDOWPOS_UNDEFINED = C.SDL_WINDOWPOS_UNDEFINED
)

const (
	SDL_HINT_RENDER_SCALE_QUALITY = C.SDL_HINT_RENDER_SCALE_QUALITY
)

const (
	SDL_RENDERER_ACCELERATED  = C.SDL_RENDERER_ACCELERATED
	SDL_RENDERER_PRESENTVSYNC = C.SDL_RENDERER_PRESENTVSYNC
)

const (
	IMG_INIT_JPG  = C.IMG_INIT_JPG
	IMG_INIT_PNG  = C.IMG_INIT_PNG
	IMG_INIT_TIF  = C.IMG_INIT_TIF
	IMG_INIT_WEBP = C.IMG_INIT_WEBP
)

const (
	MIX_DEFAULT_FORMAT = C.MIX_DEFAULT_FORMAT
)

const (
	SDL_QUIT    = C.SDL_QUIT
	SDL_KEYDOWN = C.SDL_KEYDOWN
)

const (
	SDL_TRUE  = C.SDL_TRUE
	SDL_FALSE = C.SDL_FALSE
)
