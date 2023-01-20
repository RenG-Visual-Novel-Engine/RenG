package event

/*
#cgo CFLAGS: -I./../../../sdl/include
#cgo CFLAGS: -I./c
#cgo LDFLAGS: -L./../../../sdl/lib -lSDL2 -lSDL2main -lSDL2_image -lSDL2_ttf -lSDL2_mixer

#include <event.h>
*/
import "C"
import (
	"RenG/RVM/src/core/globaltype"
	"sync"
)

type Event struct {
	e globaltype.SDL_Event

	lock sync.Mutex

	Key map[string][]KeyEvent
}

func Init() *Event {
	return &Event{
		Key: make(map[string][]KeyEvent),
	}
}

func (e *Event) Close() {}

func (e *Event) Update() bool {
	for int(C.SDL_PollEvent((*C.SDL_Event)(&e.e))) != 0 {
		switch int(C.eventType((C.SDL_Event)(e.e))) {
		case SDL_QUIT:
			return true
		case SDL_KEYDOWN:
			for _, events := range e.Key {
				for _, event := range events {
					event.Down(&EVENT_Key{
						KeyCode: int(C.eventKey((C.SDL_Event)(e.e))),
					})
				}
			}
		case SDL_KEYUP:
			for _, events := range e.Key {
				for _, event := range events {
					event.Up(&EVENT_Key{
						KeyCode: int(C.eventKey((C.SDL_Event)(e.e))),
					})
				}
			}
		}
	}
	return false
}

func (e *Event) AddKeyEvent(screenName string, ke KeyEvent) {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.Key[screenName] = append(e.Key[screenName], ke)
}

func (e *Event) DeleteScreenAllKeyEvent(screenName string) {
	e.lock.Lock()
	defer e.lock.Unlock()

	delete(e.Key, screenName)
}
