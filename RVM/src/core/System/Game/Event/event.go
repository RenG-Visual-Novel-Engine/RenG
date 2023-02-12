package event

/*
#cgo CFLAGS: -I./../../../sdl/include
#cgo CFLAGS: -I./c
#cgo LDFLAGS: -L./../../../sdl/lib -lSDL2 -lSDL2main -lSDL2_image -lSDL2_ttf -lSDL2_mixer

#include <SDL.h>
*/
import "C"
import (
	"sync"
)

type Event struct {
	e C.SDL_Event

	lock sync.Mutex

	TopScreenName string

	Key    map[string][]KeyEvent
	Button map[string][]ButtonEvent
}

func Init() *Event {
	return &Event{
		Key:    make(map[string][]KeyEvent),
		Button: make(map[string][]ButtonEvent),
	}
}

func (e *Event) Close() {}

func (e *Event) Update() bool {
	for e.pollEvent() != 0 {
		switch e.getEventType() {
		case SDL_QUIT:
			return true
		case SDL_KEYDOWN:
			e.keyDown()
		case SDL_KEYUP:
			e.keyUp()
		case SDL_MOUSEBUTTONDOWN:
		case SDL_MOUSEBUTTONUP:
			e.buttonUp()
		case SDL_MOUSEMOTION:
			e.buttonHover()
		case SDL_MOUSEWHEEL:
		}
	}
	return false
}

func (e *Event) DeleteAllScreenEvent(screenName string) {
	e.lock.Lock()
	defer e.lock.Unlock()

	delete(e.Key, screenName)
	delete(e.Button, screenName)
}
