package video

/*
#cgo CFLAGS: -I./../../sdl/include
#cgo CFLAGS: -I./../../ffmpeg/include
#cgo CFLAGS: -I./c
#cgo LDFLAGS: -L./../../sdl/lib -lSDL2 -lSDL2main
#cgo LDFLAGS: -L./../../ffmpeg/lib -lavcodec -lavformat -lavutil -lswscale

#include <ffvideo.h>
*/
import "C"
import (
	"RenG/RVM/src/core/t"
	"unsafe"
)

type Video struct {
	V *C.VideoState
}

func Init() Video {
	return Video{}
}

func (v *Video) VideoInit(path string) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))

	v.V = C.VideoInit(cpath)
}

func (v *Video) VideoStart(r *t.SDL_Renderer) {
	C.Start(v.V, (*C.SDL_Renderer)(r))
}

func (v *Video) GetTexture() *C.SDL_Texture {
	v.Lock()
	defer v.Unlock()

	return v.V.texture
}

func (v *Video) Lock() {
	C.Lock(v.V)
}

func (v *Video) Unlock() {
	C.Unlock(v.V)
}

func (v *Video) Close() {

}
