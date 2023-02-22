package graphic

import "RenG/RVM/src/core/globaltype"

func (g *Graphic) VideoStart(ScreenName, VideoName string, loop bool) {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.Video.VideoStart(ScreenName, VideoName, loop)
}

func (g *Graphic) ScreenVideoAllStop(ScreenName string) {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.Video.ScreenVideoAllStop(ScreenName)
}

func (g *Graphic) GetVideoTexture(name string) *globaltype.SDL_Texture {
	g.lock.Lock()
	defer g.lock.Unlock()

	return (*globaltype.SDL_Texture)(g.Video.GetVideoTexture(name))
}

func (g *Graphic) GetNowPlaying(name string) bool {
	g.lock.Lock()
	defer g.lock.Unlock()

	return g.Video.GetNowPlaying(name)
}
