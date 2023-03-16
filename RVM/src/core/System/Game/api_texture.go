package game

func (g *Game) ChangeTextureAlpha(name string, alpha int) {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.Graphic.Image.ChangeTextureAlpha(g.Graphic.Image.GetImageTexture(name), alpha)
}
