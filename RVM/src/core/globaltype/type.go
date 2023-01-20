package globaltype

/*
#cgo CFLAGS: -I./../sdl/include
#cgo LDFLAGS: -L./../sdl/lib -lSDL2 -lSDL2main -lSDL2_image -lSDL2_ttf

#include <SDL.h>
#include <SDL_image.h>
#include <SDL_ttf.h>
*/
import "C"

type (
	SDL_Window   = C.SDL_Window
	SDL_Renderer = C.SDL_Renderer
	SDL_Event    = C.SDL_Event
	SDL_Texture  = C.SDL_Texture
	SDL_Surface  = C.SDL_Surface

	TTF_Font = C.TTF_Font
)
