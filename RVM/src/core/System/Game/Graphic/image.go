package graphic

import (
	texture "RenG/RVM/src/core/System/Game/Graphic/Texture"
	"RenG/RVM/src/core/globaltype"
	"log"
)

func (g *Graphic) GetImageWidth(name string) int {
	g.lock.Lock()
	defer g.lock.Unlock()

	return g.images[name].width
}

func (g *Graphic) GetImageHeight(name string) int {
	g.lock.Lock()
	defer g.lock.Unlock()

	return g.images[name].height
}

func (g *Graphic) GetImageTexture(name string) *globaltype.SDL_Texture {
	g.lock.Lock()
	defer g.lock.Unlock()

	return g.images[name].texture
}

func (g *Graphic) SetImageAlphaByName(name string, alpha int) {
	g.lock.Lock()
	defer g.lock.Unlock()

	image, ok := g.images[name]
	if !ok {
		log.Printf("Image Name Error : got - %s", name)
		return
	}
	texture.TextureAlphaChange(
		image.texture,
		alpha,
	)
}

func (g *Graphic) SetVideoAlphaByName(name string, alpha int) {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.videos.Lock()
	_, ok := g.videos.V[name]
	if !ok {
		g.videos.Unlock()
		log.Printf("Video Name Error : got - %s", name)
		return
	}
	g.videos.Unlock()
	texture.TextureAlphaChange(
		(*globaltype.SDL_Texture)(g.videos.GetTexture(name)),
		alpha,
	)
}

func (g *Graphic) SetAlphaByBps(bps, index, alpha int) {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.videos.Lock()
	texture.TextureAlphaChange(
		g.renderBuffer[bps][index].texture,
		alpha,
	)
	g.videos.Unlock()
}

func (g *Graphic) SetRotateByBps(bps, index, alpha int) {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.videos.Lock()
	g.renderBuffer[bps][index].transform.Rotate = alpha
	g.videos.Unlock()
}
