package event

/*
#cgo CFLAGS: -I./../../../sdl/include
#cgo CFLAGS: -I./c
#cgo LDFLAGS: -L./../../../sdl/lib -lSDL2 -lSDL2main -lSDL2_image -lSDL2_ttf -lSDL2_mixer

#include <SDL.h>
*/
import "C"
import (
	"unsafe"
)

func (e *Event) pollEvent() int {
	return int(C.SDL_PollEvent(&e.e))
}

func (e *Event) getEventType() int {
	return int(*(*uint32)(unsafe.Pointer(&e.e[0])))
}

func (e *Event) getKeyEvent() *C.SDL_KeyboardEvent {
	return (*C.SDL_KeyboardEvent)(unsafe.Pointer(&e.e[0]))
}

func (e *Event) getMouseMotionEvent() *C.SDL_MouseMotionEvent {
	return (*C.SDL_MouseMotionEvent)(unsafe.Pointer(&e.e[0]))
}

func (e *Event) getMouseButtonEvent() *C.SDL_MouseButtonEvent {
	return (*C.SDL_MouseButtonEvent)(unsafe.Pointer(&e.e[0]))
}

/*
typedef union SDL_Event
{
    Uint32 type;
    SDL_CommonEvent common;
    SDL_DisplayEvent display;
    SDL_WindowEvent window;
    SDL_KeyboardEvent key;
    SDL_TextEditingEvent edit;
    SDL_TextInputEvent text;
    SDL_MouseMotionEvent motion;
    SDL_MouseButtonEvent button;
    SDL_MouseWheelEvent wheel;
    SDL_JoyAxisEvent jaxis;
    SDL_JoyBallEvent jball;
    SDL_JoyHatEvent jhat;
    SDL_JoyButtonEvent jbutton;
    SDL_JoyDeviceEvent jdevice;
    SDL_ControllerAxisEvent caxis;
    SDL_ControllerButtonEvent cbutton;
    SDL_ControllerDeviceEvent cdevice;
    SDL_ControllerTouchpadEvent ctouchpad;
    SDL_ControllerSensorEvent csensor;
    SDL_AudioDeviceEvent adevice;
    SDL_SensorEvent sensor;
    SDL_SysWMEvent syswm;
    SDL_QuitEvent quit;
    SDL_UserEvent user;
    SDL_TouchFingerEvent tfinger;
    SDL_MultiGestureEvent mgesture;
    SDL_DollarGestureEvent dgesture;
    SDL_DropEvent drop;

    Uint8 padding[sizeof(void *) <= 8 ? 56 : sizeof(void *) == 16 ? 64 : 3 * sizeof(void *)];
} SDL_Event;
*/
