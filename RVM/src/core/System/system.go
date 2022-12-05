package system

/*
#cgo CFLAGS: -I./../sdl/include
#cgo LDFLAGS: -L./../sdl/lib -lSDL2 -lSDL2main -lSDL2_image -lSDL2_ttf -lSDL2_mixer

#include <SDL.h>
#include <SDL_image.h>
#include <SDL_ttf.h>
#include <SDL_mixer.h>

Uint32 eventType(SDL_Event event)
{
	return event.type;
}

*/
import "C"
import (
	video "RenG/RVM/src/core/Graphic/Video"
	"RenG/RVM/src/core/System/vm"
	"RenG/RVM/src/core/t"
	"unsafe"
)

type System struct {
	window   *t.SDL_Window
	renderer *t.SDL_Renderer
	event    t.SDL_Event
	vm       *vm.VM
}

func Init(title string, width, height int) *System {
	system := &System{}

	if C.SDL_Init(t.SDL_INIT_EVERYTHING) < 0 {
		return nil
	}

	Ctitle := C.CString(title)
	defer C.free(unsafe.Pointer(Ctitle))

	window := C.SDL_CreateWindow(
		Ctitle, t.SDL_WINDOWPOS_CENTERED, t.SDL_WINDOWPOS_CENTERED,
		C.int(width), C.int(height),
		t.SDL_WINDOW_SHOWN|t.SDL_WINDOW_RESIZABLE|t.SDL_WINDOW_INPUT_FOCUS|t.SDL_WINDOW_MOUSE_FOCUS,
	)
	if window == nil {
		return nil
	}

	renderer := C.SDL_CreateRenderer(window, -1,
		t.SDL_RENDERER_ACCELERATED|t.SDL_RENDERER_PRESENTVSYNC|t.SDL_RENDERER_TARGETTEXTURE,
	)
	if renderer == nil {
		return nil
	}

	system.window = (*t.SDL_Window)(window)
	system.renderer = (*t.SDL_Renderer)(renderer)

	return system
}

func (s *System) Close() {
	C.SDL_DestroyWindow((*C.SDL_Window)(s.window))
	C.SDL_DestroyRenderer((*C.SDL_Renderer)(s.renderer))

	C.SDL_Quit()
}

// 임시
func (s *System) Render(v *video.Video) {
	var quit bool = false

	for !quit {
		for int(C.SDL_PollEvent((*C.SDL_Event)(&s.event))) != 0 {
			switch int(C.eventType((C.SDL_Event)(s.event))) {
			case 256:
				quit = true
			}
		}

		C.SDL_RenderClear((*C.SDL_Renderer)(s.renderer))
		C.SDL_SetRenderDrawColor((*C.SDL_Renderer)(s.renderer), C.uchar(0), C.uchar(0), C.uchar(0), C.uchar(255))

		v.Lock()
		C.SDL_RenderCopy((*C.SDL_Renderer)(s.renderer), (*C.SDL_Texture)(v.GetTexture()), nil, nil)
		v.Unlock()

		C.SDL_RenderPresent((*C.SDL_Renderer)(s.renderer))
	}
}

func (s *System) GetRenderer() *t.SDL_Renderer {
	return s.renderer
}
