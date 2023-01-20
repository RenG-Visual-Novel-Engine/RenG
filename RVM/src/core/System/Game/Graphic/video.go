package graphic

import "RenG/RVM/src/core/globaltype"

func (g *Graphic) VideoStart(name string) {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.videos.VideoStart(name)
}

func (g *Graphic) GetVideoTexture(name string) *globaltype.SDL_Texture {
	g.lock.Lock()
	defer g.lock.Unlock()

	return (*globaltype.SDL_Texture)(g.videos.GetTexture(name))
}
