package sdl

type LayerList struct {
	Layers []Layer
}

type Layer struct {
	Name   string
	Images []*SDL_Texture
}

func NewLayerList() LayerList {
	return LayerList{}
}

func (l *Layer) AddNewTexture(texture *SDL_Texture) {
	l.Images = append(l.Images, texture)
}
