package graphic

import (
	"RenG/RVM/src/core/globaltype"
	"log"
)

func (g *Graphic) GetTexture(bps, index int) *globaltype.SDL_Texture {
	g.lock.Lock()
	defer g.lock.Unlock()

	return g.renderBuffer[bps][index].texture
}

func (g *Graphic) GetCurrentTextureXPosition(bps, index int) (x int) {
	g.lock.Lock()
	defer g.lock.Unlock()
	return g.renderBuffer[bps][index].transform.Pos.X
}

func (g *Graphic) GetCurrentTextureYPosition(bps, index int) (x int) {
	g.lock.Lock()
	defer g.lock.Unlock()

	return g.renderBuffer[bps][index].transform.Pos.Y
}

func (g *Graphic) GetCurrentTextureXSize(bps, index int) (x int) {
	g.lock.Lock()
	defer g.lock.Unlock()
	return g.renderBuffer[bps][index].transform.Size.X
}

func (g *Graphic) GetCurrentTextureYSize(bps, index int) (y int) {
	g.lock.Lock()
	defer g.lock.Unlock()

	return g.renderBuffer[bps][index].transform.Size.Y
}

func (g *Graphic) GetCurrentTextureXFlip(bps, index int) (x int) {
	g.lock.Lock()
	defer g.lock.Unlock()
	return g.renderBuffer[bps][index].transform.Flip.X
}

func (g *Graphic) GetCurrentTextureYFlip(bps, index int) (y int) {
	g.lock.Lock()
	defer g.lock.Unlock()

	return g.renderBuffer[bps][index].transform.Flip.Y
}

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

func (g *Graphic) SetCurrentTextureXPosition(bps, index, value int) {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.renderBuffer[bps][index].transform.Pos.X = value
}

func (g *Graphic) SetCurrentTextureYPosition(bps, index, value int) {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.renderBuffer[bps][index].transform.Pos.Y = value
}

func (g *Graphic) SetCurrentTextureXSize(bps, index, value int) {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.renderBuffer[bps][index].transform.Size.X = value
}

func (g *Graphic) SetCurrentTextureYSize(bps, index, value int) {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.renderBuffer[bps][index].transform.Size.Y = value
}

func (g *Graphic) SetCurrentTextureXFlip(bps, index, value int) {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.renderBuffer[bps][index].transform.Flip.X = value
}

func (g *Graphic) SetCurrentTextureYFlip(bps, index, value int) {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.renderBuffer[bps][index].transform.Flip.Y = value
}

func (g *Graphic) SetVideoAlphaByName(name string, alpha int) {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.Video_Manager.Lock()
	_, ok := g.Video_Manager.V[name]
	if !ok {
		g.Video_Manager.Unlock()
		log.Printf("Video Name Error : got - %s", name)
		return
	}
	g.Video_Manager.Unlock()
	g.Image_Manager.ChangeTextureAlpha(
		(*globaltype.SDL_Texture)(g.Video_Manager.GetVideoTexture(name)),
		alpha,
	)
}

func (g *Graphic) SetAlphaByBps(bps, index, alpha int) {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.Video_Manager.Lock()
	g.Image_Manager.ChangeTextureAlpha(
		g.renderBuffer[bps][index].texture,
		alpha,
	)
	g.Video_Manager.Unlock()
}

func (g *Graphic) SetRotateByBps(bps, index, alpha int) {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.Video_Manager.Lock()
	g.renderBuffer[bps][index].transform.Rotate = alpha
	g.Video_Manager.Unlock()
}

func (g *Graphic) ChangeTextureByBpsUseImageManager(bps, index int, changeImageName string) {
	g.lock.Lock()
	defer g.lock.Unlock()

	if g.renderBuffer[bps][index].texture != g.Image_Manager.GetImageTexture(changeImageName) {
		g.renderBuffer[bps][index].texture = g.Image_Manager.GetImageTexture(changeImageName)
	}
}

func (g *Graphic) ChangeTextureByBps(bps, index int, texture *globaltype.SDL_Texture) {
	g.lock.Lock()
	defer g.lock.Unlock()

	if g.renderBuffer[bps][index].texture != texture {
		g.renderBuffer[bps][index].texture = texture
	}
}

func (g *Graphic) ChangeTextureTransformByBps(bps, index int, xpos, ypos, xsize, ysize int) {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.renderBuffer[bps][index].transform.Pos.X = xpos
	g.renderBuffer[bps][index].transform.Pos.Y = ypos
	g.renderBuffer[bps][index].transform.Size.X = xsize
	g.renderBuffer[bps][index].transform.Size.Y = ysize
}

// Real -> Change
func (g *Graphic) GetFixedChangeXSize(i int) int {
	xsize, _ := g.GetCurrentWindowSize()
	return int(float32(i) * float32(xsize) / float32(g.width))
}

// Real -> Change
func (g *Graphic) GetFixedChangeYSize(i int) int {
	_, ysize := g.GetCurrentWindowSize()
	return int(float32(i) * float32(ysize) / float32(g.height))
}

// Change -> Real
func (g *Graphic) GetFixedRealXSize(i int) int {
	xsize, _ := g.GetCurrentWindowSize()
	return int(float32(i) * float32(g.width) / float32(xsize))
}

// Change -> Real
func (g *Graphic) GetFixedRealYSize(i int) int {
	_, ysize := g.GetCurrentWindowSize()
	return int(float32(i) * float32(g.height) / float32(ysize))
}
