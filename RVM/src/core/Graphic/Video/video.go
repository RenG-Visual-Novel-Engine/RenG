package video

/*
#cgo CFLAGS: -I./../../sdl/include
#cgo CFLAGS: -I./../../ffmpeg/include
#cgo CFLAGS: -I./c
#cgo LDFLAGS: -L./../../sdl/lib -lSDL2 -lSDL2main
#cgo LDFLAGS: -L./../../ffmpeg/lib -lavcodec -lavformat -lswresample -lavutil -lswscale

#include <ffvideo.h>
*/
import "C"

func New() {
	C.VideoInit(C.CString("g"))
}
