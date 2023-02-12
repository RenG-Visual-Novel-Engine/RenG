package obj

type PlayVideo struct {
	Name  string
	T     Transform
	Loop  bool
	Anime []*Anime
}

func (pv *PlayVideo) screenObj() {}
func (pv *PlayVideo) labelObj()  {}

type StopVideo struct {
	Name  string
	Anime []*Anime
}

func (sv *StopVideo) labelObj() {}
