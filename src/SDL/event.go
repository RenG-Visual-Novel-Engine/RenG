package sdl

/*
#cgo LDFLAGS: -L./lib -lSDL2 -lSDL2main -lSDL2_image -lSDL2_ttf -lSDL2_mixer

#include <include/SDL.h>
#include <include/SDL_image.h>
#include <include/SDL_ttf.h>
#include <include/SDL_mixer.h>

Uint32 eventType(SDL_Event event)
{
	return event.type;
}
*/
import "C"

func PollEvent(event *SDL_Event) int {
	return int(C.SDL_PollEvent((*C.SDL_Event)(event)))
}

func EventType(event SDL_Event) C.Uint32 {
	return C.eventType((C.SDL_Event)(event))
}
