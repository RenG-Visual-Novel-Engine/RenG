package object

import (
	sdl "RenG/src/SDL"
	"sync"
)

type TextureList struct {
	store map[string]*sdl.SDL_Texture
	outer *TextureList
}

var (
	TexMutex = &sync.RWMutex{}
)

func NewTextureList() *TextureList {
	t := make(map[string]*sdl.SDL_Texture)
	return &TextureList{store: t, outer: nil}
}

func (t *TextureList) Get(name string) (*sdl.SDL_Texture, bool) {
	TexMutex.RLock()
	obj, ok := t.store[name]
	if !ok && t.outer != nil {
		TexMutex.RUnlock()
		obj, ok = t.outer.Get(name)
	}
	TexMutex.RUnlock()
	return obj, ok
}

func (t *TextureList) Set(name string, texture *sdl.SDL_Texture) *sdl.SDL_Texture {
	TexMutex.Lock()
	t.store[name] = texture
	TexMutex.Unlock()
	return texture
}

func (t *TextureList) DestroyAll() {
	TexMutex.Lock()
	for _, texture := range t.store {
		sdl.DestroyTexture(texture)
	}
}
