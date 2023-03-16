package video

/*
#cgo CFLAGS: -I./../../../../sdl/include
#cgo CFLAGS: -I./../../../../ffmpeg/include
#cgo CFLAGS: -I./c
#cgo LDFLAGS: -L./../../../../sdl/lib -lSDL2 -lSDL2main
#cgo LDFLAGS: -lwinmm
#cgo LDFLAGS: -L./../../../../ffmpeg/lib -lavcodec -lavformat -lavutil -lswscale






#include <ffvideo.h>
*/
import "C"
import (
	"RenG/RVM/src/core/globaltype"
)

type Video struct {
	ScreenVideo map[string][]string
	V           map[string]*C.VideoState
}

func Init() *Video {
	return &Video{
		ScreenVideo: make(map[string][]string),
		V:           make(map[string]*C.VideoState),
	}
}

func (v *Video) Close() {
	for _, v := range v.V {
		C.SDL_DestroyTexture(v.texture)
	}
}

func (v *Video) Register(name, path string, r *globaltype.SDL_Renderer) {
	v.V[name] = C.VideoInit(C.CString(path), (*C.SDL_Renderer)(r))
}

func (v *Video) VideoStart(ScreenName, VideoName string, loop bool) {
	v.Lock()
	defer v.Unlock()

	v.ScreenVideo[ScreenName] = append(v.ScreenVideo[ScreenName], VideoName)

	if v.V[VideoName].stop == 1 {
		v.V[VideoName].stop = 0
	}

	if loop {
		C.Start(v.V[VideoName], 1)
	} else {
		C.Start(v.V[VideoName], 0)
	}
}

func (v *Video) VideoStop(VideoName string) {
	v.Lock()
	defer v.Unlock()

	v.V[VideoName].stop = 1
}

func (v *Video) ScreenVideoAllStop(ScreenName string) {
	v.Lock()
	defer v.Unlock()

	for _, videoName := range v.ScreenVideo[ScreenName] {
		v.V[videoName].stop = 1
	}

	delete(v.ScreenVideo, ScreenName)
}

func (v *Video) Lock() {
	C.Lock()
}

func (v *Video) Unlock() {
	C.Unlock()
}
