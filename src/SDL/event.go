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

Uint32 windowEventType(SDL_Event event)
{
	return event.window.event;
}
*/
import "C"

type Event struct {
	Event SDL_Event
	Type  uint32
}

func (e *SDL_Event) PollEvent() int {
	return int(C.SDL_PollEvent((*C.SDL_Event)(e)))
}

func (e *SDL_Event) EventType() C.Uint32 {
	return C.eventType((C.SDL_Event)(*e))
}

func (e *SDL_Event) WindowEventType() C.Uint32 {
	return C.windowEventType((C.SDL_Event)(*e))
}
