package object

import (
	"RenG/src/core"
	"sync"
)

type TextureList struct {
	store map[string]*core.SDL_Texture
	outer *TextureList
}

var (
	TexMutex = &sync.RWMutex{}
)

func NewTextureList() *TextureList {
	t := make(map[string]*core.SDL_Texture)
	return &TextureList{store: t, outer: nil}
}

func (t *TextureList) Get(name string) (*core.SDL_Texture, bool) {
	TexMutex.RLock()
	obj, ok := t.store[name]
	if !ok && t.outer != nil {
		TexMutex.RUnlock()
		obj, ok = t.outer.Get(name)
	}
	TexMutex.RUnlock()
	return obj, ok
}

func (t *TextureList) Set(name string, texture *core.SDL_Texture) *core.SDL_Texture {
	TexMutex.Lock()
	t.store[name] = texture
	TexMutex.Unlock()
	return texture
}

func (t *TextureList) DestroyAll() {
	TexMutex.Lock()
	for _, texture := range t.store {
		texture.DestroyTexture()
	}
	TexMutex.Unlock()
}
