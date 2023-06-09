package video

/*
#cgo CFLAGS: -I./../../../../sdl/include
#cgo CFLAGS: -I./../../../../ffmpeg/include
#cgo CFLAGS: -I./c
#cgo LDFLAGS: -L./../../../../sdl/lib -lSDL2 -lSDL2main
#cgo LDFLAGS: -lwinmm
#cgo LDFLAGS: -L./../../../../ffmpeg/lib -lavcodec -lavformat -lavutil -lswscale

#include <libavcodec/avcodec.h>
#include <libavformat/avformat.h>
#include <libswscale/swscale.h>
*/
import "C"
import "unsafe"

func (v *Video) GetFrameData(frame *C.AVFrame) ([8]*uint8, [8]int32) {
	return *(*[8]*uint8)(unsafe.Pointer(&frame.data)), *(*[8]int32)(unsafe.Pointer(&frame.linesize))
}
