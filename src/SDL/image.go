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

func Render(renderer *SDL_Renderer, texture *SDL_Texture) {
	RenderCopy(renderer, texture)
}
