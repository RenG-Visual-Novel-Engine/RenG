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

	Key        map[string][]KeyEvent
	Button     map[string][]ButtonEvent
	Bar        map[string][]BarEvent
	MouseClick map[string][]MouseClickEvent
}

func Init() *Event {
	return &Event{
		Key:        make(map[string][]KeyEvent),
		Button:     make(map[string][]ButtonEvent),
		Bar:        make(map[string][]BarEvent),
		MouseClick: make(map[string][]MouseClickEvent),
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
			e.buttonDown()     // 버튼 이벤트 먼저 처리.
			e.mouseClickDown() // 그후 마우스 클릭 다운을 처리해야 버튼의 press 상태를 알 수 있음.
			e.barDown()
		case SDL_MOUSEBUTTONUP:
			e.mouseClickUp() // 마우스 이벤트 먼저 처리해야 버튼이 UP하기 전에, press 상태를 알 수 있음.
			e.buttonUp()     // 그후 버튼 이벤트 처리리
			e.barUp()
		case SDL_MOUSEMOTION:
			e.buttonHover()
			e.barScroll()
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
	delete(e.Bar, screenName)
	delete(e.MouseClick, screenName)
}
