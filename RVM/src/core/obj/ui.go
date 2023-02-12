package obj

type Show struct {
	Name  string
	T     Transform
	Anime []*Anime
}

func (s *Show) screenObj() {}
func (s *Show) labelObj()  {}

type Hide struct {
	Name  string
	Anime []*Anime
}

func (h *Hide) labelObj() {}
