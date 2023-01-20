package system

/*
#cgo CFLAGS: -I./../sdl/include
#cgo LDFLAGS: -L./../sdl/lib -lSDL2 -lSDL2main -lSDL2_image -lSDL2_ttf -lSDL2_mixer

#include <SDL.h>
#include <SDL_image.h>
#include <SDL_ttf.h>
#include <SDL_mixer.h>
*/
import "C"
import (
	audio "RenG/RVM/src/core/System/Game/Audio"
	graphic "RenG/RVM/src/core/System/Game/Graphic"
	"RenG/RVM/src/core/System/game"
	"RenG/RVM/src/core/globaltype"
	"log"
	"os"
	"unsafe"
)

type System struct {
	window *globaltype.SDL_Window

	title string

	Game *game.Game
	// vm       *vm.VM
}

func Init(title string,
	width, height int,
	CursorPath string,
) *System {

	if C.SDL_Init(C.SDL_INIT_EVERYTHING) < 0 {
		return nil
	}

	Ctitle := C.CString(title)
	defer C.free(unsafe.Pointer(Ctitle))

	window := C.SDL_CreateWindow(
		Ctitle, C.SDL_WINDOWPOS_CENTERED, C.SDL_WINDOWPOS_CENTERED,
		C.int(width), C.int(height),
		C.SDL_WINDOW_HIDDEN|C.SDL_WINDOW_INPUT_FOCUS|C.SDL_WINDOW_MOUSE_FOCUS,
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

	C.TTF_Init()

	hint1 := C.CString(C.SDL_HINT_RENDER_SCALE_QUALITY)
	defer C.free(unsafe.Pointer(hint1))

	hint2 := C.CString("1")
	defer C.free(unsafe.Pointer(hint2))

	if C.SDL_SetHint(hint1, hint2) == 0 {
		log.Println("Hint quality Error")
	}

	path, _ := os.Getwd()

	g := graphic.Init((*globaltype.SDL_Renderer)(renderer), path)
	g.RegisterCursor(CursorPath)

	return &System{
		window: (*globaltype.SDL_Window)(window),
		title:  title,
		Game: game.Init(
			g,
			audio.Init(),
			path,
		),
	}
}

func (s *System) Close() {
	C.SDL_DestroyWindow((*C.SDL_Window)(s.window))

	s.Game.Close()
	C.SDL_Quit()
}

func (s *System) WindowStart(firstScreen string) {
	s.Game.ActiveScreen(firstScreen)
	C.SDL_ShowWindow((*C.SDL_Window)(s.window))

	for {
		if s.Game.Event.Update() {
			break
		}
		s.Game.Graphic.Update()
		s.Game.Graphic.Render()
	}
}

func (s *System) GameStart(firstLabel string) {
	go s.Game.StartLabel(firstLabel)
}
