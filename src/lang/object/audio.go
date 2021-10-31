package object

import (
	"RenG/src/core"
	"sync"
)

type MusicList struct {
	store map[string]*core.Mix_Music
	outer *MusicList
}

type ChunkList struct {
	store map[string]*core.Mix_Chunk
	outer *ChunkList
}

var (
	MusMutex = &sync.RWMutex{}
	ChuMutex = &sync.RWMutex{}
)

func NewMusicList() *MusicList {
	m := make(map[string]*core.Mix_Music)
	return &MusicList{store: m, outer: nil}
}

func (m *MusicList) Get(name string) (*core.Mix_Music, bool) {
	MusMutex.RLock()
	obj, ok := m.store[name]
	if !ok && m.outer != nil {
		MusMutex.RUnlock()
		obj, ok = m.outer.Get(name)
	}
	MusMutex.RUnlock()
	return obj, ok
}

func (m *MusicList) Set(name string, music *core.Mix_Music) *core.Mix_Music {
	MusMutex.Lock()
	m.store[name] = music
	MusMutex.Unlock()
	return music
}

func (m *MusicList) FreaAll() {
	MusMutex.Lock()
	for _, music := range m.store {
		music.FreeMusic()
	}
	MusMutex.Unlock()
}

func NewChunkList() *ChunkList {
	c := make(map[string]*core.Mix_Chunk)
	return &ChunkList{store: c, outer: nil}
}

func (c *ChunkList) Get(name string) (*core.Mix_Chunk, bool) {
	ChuMutex.RLock()
	obj, ok := c.store[name]
	if !ok && c.outer != nil {
		ChuMutex.RUnlock()
		obj, ok = c.outer.Get(name)
	}
	ChuMutex.RUnlock()
	return obj, ok
}

func (c *ChunkList) Set(name string, chunk *core.Mix_Chunk) *core.Mix_Chunk {
	ChuMutex.Lock()
	c.store[name] = chunk
	ChuMutex.Unlock()
	return chunk
}

func (c *ChunkList) FreeAll() {
	ChuMutex.Lock()
	for _, chunk := range c.store {
		chunk.FreeChunk()
	}
	ChuMutex.Unlock()
}
