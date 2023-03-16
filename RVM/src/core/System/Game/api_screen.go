package game

import (
	"RenG/RVM/src/core/obj"
	"log"
	"strconv"
)

func (g *Game) IsActiveScreen(name string) bool {
	g.lock.Lock()
	defer g.lock.Unlock()

	_, ok := g.screens[name]
	return ok
}

func (g *Game) ActiveScreen(name string) {
	g.lock.Lock()
	defer g.lock.Unlock()

	screen, ok := g.screens[name]
	if !ok {
		log.Printf("Screen Name Error, got - %s\n", name)
	}

	bps := g.Graphic.GetCurrentTopRenderBps() + 1
	g.Graphic.AddScreenRenderBuffer()

	g.Event.TopScreenName = name

	g.screenBps[name] = bps
	g.screenEval(screen.Obj, name, bps)
}

func (g *Game) InActiveScreen(name string) {
	g.lock.Lock()
	defer g.lock.Unlock()

	bps, ok := g.screenBps[name]
	if !ok {
		return
	}
	delete(g.screenBps, name)

	g.Event.DeleteAllScreenEvent(name)
	g.Graphic.DeleteAnimationByScreenName(name)
	g.Graphic.DeleteTypingFXByScreenName(name)
	g.Graphic.DestroyScreenTextTexture(name)
	g.Graphic.ScreenVideoAllStop(name)

	for target, target_bps := range g.screenBps {
		if target_bps > bps {
			g.screenBps[target] = target_bps - 1
			g.Graphic.UpdateTypingFXScreenBPS(target, target_bps-1)
			g.Graphic.UpdateAnimationScreenBPS(target, target_bps-1)
		}
		if bps == 0 && target_bps == 1 {
			g.Event.TopScreenName = target
		}
	}

	g.Graphic.DeleteScreenRenderBuffer(bps)
}

func (g *Game) GetScreenBps(screenName string) int {
	g.lock.Lock()
	defer g.lock.Unlock()

	return g.screenBps[screenName]
}

// Random
func (g *Game) GetShowScreenNamesWithoutSayScreen() []string {
	g.lock.Lock()
	defer g.lock.Unlock()

	var ret []string
	for name, bps := range g.screenBps {
		if name == g.SayScreenName {
			continue
		}
		ret = append(ret, strconv.Itoa(bps)+"-"+name)
	}
	return ret
}

func (g *Game) GetScreenTextureNamesANDTransform(screenName string) []string {
	g.lock.Lock()
	defer g.lock.Unlock()

	return g.Graphic.GetCurrentRenderBufferTextureNameANDTransformByBPS(g.screenBps[screenName])
}

func (g *Game) ShowTexture(textureName, screenName string, T obj.Transform) (textureIndex int) {
	g.lock.Lock()
	defer g.lock.Unlock()

	if T.Size.X != 0 && T.Size.Y != 0 {
		T = g.echoTransform(T, T.Size.X, T.Size.Y)
	} else {
		T = g.echoTransform(T, g.Graphic.Image.GetImageWidth(textureName), g.Graphic.Image.GetImageHeight(textureName))
	}

	g.Graphic.AddScreenTextureRenderBuffer(
		g.screenBps[screenName],
		g.Graphic.Image.GetImageTexture(textureName),
		T,
	)

	return g.Graphic.GetCurrentTopScreenIndexByBps(g.screenBps[screenName])
}

func (g *Game) HideTexture(textureIndex int, screenName string) {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.Graphic.DeleteAnimationByTextureIndex(screenName, textureIndex)
	g.Graphic.DeleteScreenTextureRenderBuffer(g.screenBps[screenName], textureIndex)
}
