package core

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

func (l *Layer) DeleteTexture(index int) {
	l.Images = append(l.Images[:index], l.Images[index+1:]...)
}

func (l *Layer) DeleteAllTexture() {
	if len(l.Images) == 0 {
		return
	}

	length := len(l.Images)

	for i := 0; i < length; i++ {
		l.Images[0] = l.Images[len(l.Images)-1]
		l.Images = l.Images[:len(l.Images)-1]
	}
}

func (l *Layer) ChangeTexture(texture *SDL_Texture, index int) {
	if len(l.Images) <= index {
		l.Images = append(l.Images, texture)
	} else {
		l.Images[index] = texture
	}
}
