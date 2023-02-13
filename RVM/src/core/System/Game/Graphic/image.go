package graphic

import (
	"RenG/RVM/src/core/globaltype"
	"log"
)

func (g *Graphic) GetCurrentTexturePosition(bps, index int) (x, y int) {
	g.lock.Lock()
	defer g.lock.Unlock()

	return g.renderBuffer[bps][index].transform.Pos.X, g.renderBuffer[bps][index].transform.Pos.Y
}

func (g *Graphic) GetCurrentTextureSize(bps, index int) (x, y int) {
	g.lock.Lock()
	defer g.lock.Unlock()

	return g.renderBuffer[bps][index].transform.Size.X, g.renderBuffer[bps][index].transform.Size.Y
}

func (g *Graphic) SetVideoAlphaByName(name string, alpha int) {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.Video.Lock()
	_, ok := g.Video.V[name]
	if !ok {
		g.Video.Unlock()
		log.Printf("Video Name Error : got - %s", name)
		return
	}
	g.Video.Unlock()
	g.Image.ChangeTextureAlpha(
		(*globaltype.SDL_Texture)(g.Video.GetVideoTexture(name)),
		alpha,
	)
}

func (g *Graphic) SetAlphaByBps(bps, index, alpha int) {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.Video.Lock()
	g.Image.ChangeTextureAlpha(
		g.renderBuffer[bps][index].texture,
		alpha,
	)
	g.Video.Unlock()
}

func (g *Graphic) SetRotateByBps(bps, index, alpha int) {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.Video.Lock()
	g.renderBuffer[bps][index].transform.Rotate = alpha
	g.Video.Unlock()
}

func (g *Graphic) ChangeTextureByBps(bps, index int, changeImageName string) {
	g.lock.Lock()
	defer g.lock.Unlock()

	if g.renderBuffer[bps][index].texture != g.Image.GetImageTexture(changeImageName) {
		g.renderBuffer[bps][index].texture = g.Image.GetImageTexture(changeImageName)
	}
}
