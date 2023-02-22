package obj

type Show struct {
	Name  string
	T     Transform
	Anime []*Anime
}

func (s *Show) screenObj() {}
func (s *Show) labelObj()  {}

type Hide struct {
	TextureIndex int
	Anime        *Anime
}

func (h *Hide) labelObj() {}
