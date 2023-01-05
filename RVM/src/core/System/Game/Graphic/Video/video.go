package video

/*
#cgo CFLAGS: -I./../../../../sdl/include
#cgo CFLAGS: -I./../../../../ffmpeg/include
#cgo CFLAGS: -I./c
#cgo LDFLAGS: -lwinmm
#cgo LDFLAGS: -L./../../../../sdl/lib -lSDL2 -lSDL2main
#cgo LDFLAGS: -L./../../../../ffmpeg/lib -lavcodec -lavformat -lavutil -lswscale

#include <ffvideo.h>
*/
import "C"
import (
	"RenG/RVM/src/core/globaltype"
	"unsafe"
)

type Video struct {
	V map[string]*C.VideoState
}

func Init() *Video {
	return &Video{
		V: make(map[string]*C.VideoState),
	}
}

func (v *Video) VideoInit(name, path string, r *globaltype.SDL_Renderer) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))

	v.V[name] = C.VideoInit(cpath, (*C.SDL_Renderer)(r))
}

func (v *Video) VideoStart(name string) {
	C.Start(v.V[name])
}

func (v *Video) GetTexture(name string) *C.SDL_Texture {
	v.Lock()
	defer v.Unlock()

	return v.V[name].texture
}

func (v *Video) IsNowPlaying(name string) bool {
	if _, ok := v.V[name]; ok {
		return v.V[name].nowPlaying == 1
	}
	return false
}

func (v *Video) Lock() {
	C.Lock()
}

func (v *Video) Unlock() {
	C.Unlock()
}

func (v *Video) Close() {

}
