package st

/*
#cgo CFLAGS: -I./../sdl/include
#cgo LDFLAGS: -L./../sdl/lib -lSDL2 -lSDL2main -lSDL2_image -lSDL2_ttf -lSDL2_mixer

#include <SDL.h>
#include <SDL_image.h>
#include <SDL_ttf.h>
#include <SDL_mixer.h>
*/
import "C"

const (
	MIX_DEFAULT_FORMAT = C.MIX_DEFAULT_FORMAT
)

const (
	SDL_AUDIO_ALLOW_FREQUENCY_CHANGE = C.SDL_AUDIO_ALLOW_FREQUENCY_CHANGE
	SDL_AUDIO_ALLOW_CHANNELS_CHANGE  = C.SDL_AUDIO_ALLOW_CHANNELS_CHANGE
)
