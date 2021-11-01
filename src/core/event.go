package core

/*
#cgo CFLAGS: -I./sdl/include
#cgo LDFLAGS: -L./sdl/lib -lSDL2 -lSDL2main -lSDL2_image -lSDL2_ttf -lSDL2_mixer

#include <SDL.h>
#include <SDL_image.h>
#include <SDL_ttf.h>
#include <SDL_mixer.h>

Uint32 eventType(SDL_Event event)
{
	return event.type;
}

Uint32 windowEventType(SDL_Event event)
{
	return event.window.event;
}

Uint8 mouseButtonType(SDL_Event event)
{
	return event.button.button;
}

Sint32 GetWheelY(SDL_Event e)
{
	return e.wheel.y;
}
*/
import "C"
import (
	"time"
)

type Event struct {
	Type  uint32
	Key   key
	Mouse mouse
}

type key struct {
}

type mouse struct {
	Motion buttonMotion
	Down   buttonDown
	Up     buttonUp
	Wheel  buttonWheel
}

type buttonMotion struct {
	X int
	Y int
}

type buttonDown struct {
	Button uint8
	X      int
	Y      int
}

type buttonUp struct {
	Button uint8
	X      int
	Y      int
}

type buttonWheel struct {
	Y int
}

func (event *SDL_Event) HandleEvent(eventType int, eventChan chan Event) {
	switch eventType {
	case SDL_KEYDOWN:
	case SDL_KEYUP:
	case SDL_MOUSEMOTION:
		go func() {
			var x, y C.int
			C.SDL_GetMouseState(&x, &y)
			for {
				eventChan <- Event{
					Type: SDL_MOUSEMOTION,
					Mouse: mouse{
						Motion: buttonMotion{
							X: int(x),
							Y: int(y),
						},
					},
				}

				time.Sleep(time.Microsecond)

				if len(eventChan) > 0 {
					<-eventChan
					break
				}
			}
		}()
	case SDL_MOUSEBUTTONDOWN:
		go func() {
			var x, y C.int
			C.SDL_GetMouseState(&x, &y)
			for {
				eventChan <- Event{
					Type: SDL_MOUSEBUTTONDOWN,
					Mouse: mouse{
						Down: buttonDown{
							Button: uint8(C.mouseButtonType((C.SDL_Event)(*event))),
							X:      int(x),
							Y:      int(y),
						},
					},
				}

				if len(eventChan) > 0 {
					<-eventChan
					break
				}
			}
			// fmt.Printf("ButtonDown x : %d  y : %d\n", x, y)
		}()
	case SDL_MOUSEBUTTONUP:
		go func() {
			var x, y C.int
			C.SDL_GetMouseState(&x, &y)
			for {
				eventChan <- Event{
					Type: SDL_MOUSEBUTTONUP,
					Mouse: mouse{
						Up: buttonUp{
							Button: uint8(C.mouseButtonType((C.SDL_Event)(*event))),
							X:      int(x),
							Y:      int(y),
						},
					},
				}

				time.Sleep(time.Microsecond)

				if len(eventChan) > 0 {
					<-eventChan
					break
				}
			}
			// fmt.Printf("ButtonUp   x : %d  y : %d\n", x, y)
		}()
	case SDL_MOUSEWHEEL:
		go func() {
			y := int(C.GetWheelY((C.SDL_Event)(*event)))
			// 100 이상의 마우스 휠 입력값은 잘못된 입력으로 처리합니다.
			if y >= 100 {
				return
			}

			for {
				eventChan <- Event{
					Type: SDL_MOUSEWHEEL,
					Mouse: mouse{
						Wheel: buttonWheel{
							Y: y,
						},
					},
				}

				time.Sleep(time.Microsecond)

				if len(eventChan) > 0 {
					<-eventChan
					break
				}
			}
		}()
	}
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
