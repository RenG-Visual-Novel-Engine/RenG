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

Sint32 GetWheelY(SDL_Event e)
{
	return e.wheel.y;
}
*/
import "C"

type Event struct {
	Type  uint32
	Key   key
	Mouse mouse
}

type key struct {
}

type mouse struct {
	X int
	Y int
}

func HandleEvent(eventType int, event SDL_Event, eventChan chan Event) {
	switch eventType {
	case SDL_KEYDOWN:
	case SDL_KEYUP:
	case SDL_MOUSEMOTION:
	case SDL_MOUSEBUTTONDOWN:
	L:
		for {
			select {
			case <-eventChan:
			default:
				break L
			}
		}

		var x, y C.int
		C.SDL_GetMouseState(&x, &y)

		eventChan <- Event{
			Type: SDL_MOUSEBUTTONDOWN,
			Mouse: mouse{
				X: int(x),
				Y: int(y),
			},
		}

	case SDL_MOUSEBUTTONUP:
	case SDL_MOUSEWHEEL:
	}
}

func (e *SDL_Event) GetY() int {
	return int(C.GetWheelY((C.SDL_Event)(*e)))
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
