package object

import (
	sdl "RenG/src/SDL"
	"RenG/src/ffmpeg"
	"sync"
)

type VideoList struct {
	store map[string]*VideoObject
	outer *VideoList
}

type VideoObject struct {
	Video   *ffmpeg.Video
	Texture *sdl.SDL_Texture
}

var (
	VidMutex = &sync.RWMutex{}
)

func NewVideoList() *VideoList {
	v := make(map[string]*VideoObject)
	return &VideoList{store: v, outer: nil}
}

func (m *VideoList) Get(name string) (*VideoObject, bool) {
	VidMutex.RLock()
	obj, ok := m.store[name]
	if !ok && m.outer != nil {
		VidMutex.RUnlock()
		obj, ok = m.outer.Get(name)
	}
	VidMutex.RUnlock()
	return obj, ok
}

func (m *VideoList) Set(name string, video *VideoObject) *VideoObject {
	VidMutex.Lock()
	m.store[name] = video
	VidMutex.Unlock()
	return video
}
