package event

/*
#cgo CFLAGS: -I./../../../sdl/include
#cgo CFLAGS: -I./c
#cgo LDFLAGS: -L./../../../sdl/lib -lSDL2 -lSDL2main -lSDL2_image -lSDL2_ttf -lSDL2_mixer

#include <SDL.h>
*/
import "C"

/* -- Active -- */

func (e *Event) keyDown() {
	e.lock.Lock()
	events, ok := e.Key[e.TopScreenName]
	e.lock.Unlock()
	if !ok {
		return
	}
	for _, event := range events {
		event.Down(&EVENT_Key{
			KeyCode: int(e.getKeyEvent().keysym.sym),
		})
	}
}

func (e *Event) keyUp() {
	e.lock.Lock()
	events, ok := e.Key[e.TopScreenName]
	e.lock.Unlock()
	if !ok {
		return
	}
	for _, event := range events {
		event.Up(&EVENT_Key{
			KeyCode: int(e.getKeyEvent().keysym.sym),
		})
	}
}

/* -- Util -- */

func (e *Event) AddKeyEvent(screenName string, ke KeyEvent) {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.Key[screenName] = append(e.Key[screenName], ke)
}
