package game

import "RenG/RVM/src/core/obj"

func (g *Game) AddAnimation(a *obj.Anime, screenName string, textureIndex int) {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.Graphic.AddAnimation(screenName, a, g.screenBps[screenName], textureIndex)
}
