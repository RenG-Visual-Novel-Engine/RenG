package object

import (
	sdl "RenG/src/SDL"
	"sync"
)

type MusicList struct {
	store map[string]*sdl.Mix_Music
	outer *MusicList
}

type ChunkList struct {
	store map[string]*sdl.Mix_Chunk
	outer *ChunkList
}

var (
	MusMutex = &sync.RWMutex{}
	ChuMutex = &sync.RWMutex{}
)

func NewMusicList() *MusicList {
	m := make(map[string]*sdl.Mix_Music)
	return &MusicList{store: m, outer: nil}
}

func (m *MusicList) Get(name string) (*sdl.Mix_Music, bool) {
	MusMutex.RLock()
	obj, ok := m.store[name]
	if !ok && m.outer != nil {
		MusMutex.RUnlock()
		obj, ok = m.outer.Get(name)
	}
	MusMutex.RUnlock()
	return obj, ok
}

func (m *MusicList) Set(name string, music *sdl.Mix_Music) *sdl.Mix_Music {
	MusMutex.Lock()
	m.store[name] = music
	MusMutex.Unlock()
	return music
}

func (m *MusicList) FreaAll() {
	MusMutex.Lock()
	for _, music := range m.store {
		sdl.FreeMusic(music)
	}
	MusMutex.Unlock()
}

func NewChunkList() *ChunkList {
	c := make(map[string]*sdl.Mix_Chunk)
	return &ChunkList{store: c, outer: nil}
}

func (c *ChunkList) Get(name string) (*sdl.Mix_Chunk, bool) {
	ChuMutex.RLock()
	obj, ok := c.store[name]
	if !ok && c.outer != nil {
		ChuMutex.RUnlock()
		obj, ok = c.outer.Get(name)
	}
	ChuMutex.RUnlock()
	return obj, ok
}

func (c *ChunkList) Set(name string, chunk *sdl.Mix_Chunk) *sdl.Mix_Chunk {
	ChuMutex.Lock()
	c.store[name] = chunk
	ChuMutex.Unlock()
	return chunk
}

func (c *ChunkList) FreeAll() {
	ChuMutex.Lock()
	for _, chunk := range c.store {
		sdl.FreeChunk(chunk)
	}
	ChuMutex.Unlock()
}
