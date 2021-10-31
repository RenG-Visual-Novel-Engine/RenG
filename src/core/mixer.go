package core

/*
#cgo CFLAGS: -I./sdl/include
#cgo LDFLAGS: -L./sdl/lib -lSDL2 -lSDL2main -lSDL2_mixer

#include <SDL.h>
#include <SDL_mixer.h>

*/
import "C"
import (
	"sync"
	"unsafe"
)

type ChannelList struct {
	store map[string]int
	outer *ChannelList
}

var (
	ChaMutex = &sync.RWMutex{}
)

func NewChannelList() *ChannelList {
	c := make(map[string]int)
	return &ChannelList{store: c, outer: nil}
}

func (c *ChannelList) NewChannel(name string, num int) int {
	ChaMutex.Lock()
	c.store[name] = num
	ChaMutex.Unlock()
	return num
}

func (c *ChannelList) GetChannel(name string) (int, bool) {
	ChaMutex.RLock()
	obj, ok := c.store[name]
	if !ok && c.outer != nil {
		ChaMutex.RUnlock()
		obj, ok = c.outer.GetChannel(name)
	}
	ChaMutex.RUnlock()
	return obj, ok
}

func LoadMUS(root string) *Mix_Music {
	cRoot := C.CString(root)
	defer C.free(unsafe.Pointer(cRoot))

	return (*Mix_Music)(C.Mix_LoadMUS(cRoot))
}

func LoadWAV(root string) *Mix_Chunk {
	cRoot := C.CString(root)
	rb := C.CString("rb")
	defer C.free(unsafe.Pointer(cRoot))
	defer C.free(unsafe.Pointer(rb))

	return (*Mix_Chunk)(C.Mix_LoadWAV_RW(C.SDL_RWFromFile(cRoot, rb), 1))
}

func (music *Mix_Music) PlayMusic(loop bool) {
	if loop {
		C.Mix_PlayMusic((*C.Mix_Music)(music), -1)
	} else {
		C.Mix_PlayMusic((*C.Mix_Music)(music), 1)
	}
}

func PlayingMusic() bool {
	return int(C.Mix_PlayingMusic()) != 0
}

func PlayingMusicChannel(channel int) bool {
	return int(C.Mix_Playing(C.int(channel))) != 0
}

func StopMusic(channel int) {
	switch channel {
	case -1:
		C.Mix_HaltMusic()
	default:
		C.Mix_HaltChannel(C.int(channel))
	}
}

func (chunk *Mix_Chunk) PlaySound(channel int) {
	C.Mix_PlayChannelTimed(C.int(channel), (*C.Mix_Chunk)(chunk), 0, -1)
}

func (music *Mix_Music) FreeMusic() {
	C.Mix_FreeMusic((*C.Mix_Music)(music))
}

func (chunk *Mix_Chunk) FreeChunk() {
	C.Mix_FreeChunk((*C.Mix_Chunk)(chunk))
}
