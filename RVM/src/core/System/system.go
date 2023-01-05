package system

/*
#cgo CFLAGS: -I./../sdl/include
#cgo LDFLAGS: -L./../sdl/lib -lSDL2 -lSDL2main -lSDL2_image -lSDL2_ttf -lSDL2_mixer

#include <SDL.h>
#include <SDL_image.h>
#include <SDL_ttf.h>
#include <SDL_mixer.h>

SDL_Rect CreateRects(int x, int y, int w, int h)
{
	SDL_Rect Quad = { x, y, w, h };
	return Quad;
}

Uint32 eventType(SDL_Event event)
{
	return event.type;
}

*/
import "C"
import (
	audio "RenG/RVM/src/core/System/Game/Audio"
	graphic "RenG/RVM/src/core/System/Game/Graphic"
	"RenG/RVM/src/core/System/game"
	"RenG/RVM/src/core/globaltype"
	"os"
	"unsafe"
)

type System struct {
	window *globaltype.SDL_Window
	event  globaltype.SDL_Event

	quit  bool
	title string

	Game *game.Game
	// vm       *vm.VM
}

func Init(title string, width, height int) *System {

	if C.SDL_Init(C.SDL_INIT_EVERYTHING) < 0 {
		return nil
	}

	Ctitle := C.CString(title)
	defer C.free(unsafe.Pointer(Ctitle))

	window := C.SDL_CreateWindow(
		Ctitle, C.SDL_WINDOWPOS_CENTERED, C.SDL_WINDOWPOS_CENTERED,
		C.int(width), C.int(height),
		C.SDL_WINDOW_SHOWN|C.SDL_WINDOW_INPUT_FOCUS|C.SDL_WINDOW_MOUSE_FOCUS,
	)
	if window == nil {
		return nil
	}

	renderer := C.SDL_CreateRenderer(window, -1,
		C.SDL_RENDERER_ACCELERATED|C.SDL_RENDERER_PRESENTVSYNC|C.SDL_RENDERER_TARGETTEXTURE,
	)
	if renderer == nil {
		return nil
	}

	path, _ := os.Getwd()

	return &System{
		window: (*globaltype.SDL_Window)(window),
		quit:   false,
		title:  title,
		Game: game.Init(
			graphic.Init((*globaltype.SDL_Renderer)(renderer), path),
			audio.Init(),
		),
	}
}

func (s *System) Close() {
	C.SDL_DestroyWindow((*C.SDL_Window)(s.window))

	s.Game.Close()
	C.SDL_Quit()
}

func (s *System) Start(firstScreen string) {
	s.Game.Graphic.ActiveScreen(firstScreen)
	// s.Graphic.Videos.VideoStart(s.Graphic.Renderer)
	for !s.quit {
		for int(C.SDL_PollEvent((*C.SDL_Event)(&s.event))) != 0 {
			switch int(C.eventType((C.SDL_Event)(s.event))) {
			case 256:
				s.quit = true
			}
		}
		s.Game.Graphic.Render()
	}
}

// func (s *System)

/*
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

		C.SDL_RenderPresent((*C.SDL_Renderer)(s.renderer))

		v.Unlock()
	}
}
*/
