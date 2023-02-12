package obj

type PlayMusic struct {
	Path string
	Loop bool
	Ms   int
}

func (pa *PlayMusic) screenObj() {}
func (pa *PlayMusic) labelObj()  {}

type StopMusic struct {
	Ms int
}

func (sm *StopMusic) screenObj() {}
func (sm *StopMusic) labelObj()  {}

type PlayChannel struct {
	Path     string
	ChanName string
	Ms       int
}

func (pc *PlayChannel) screenObj() {}
func (pc *PlayChannel) labelObj()  {}

type StopChannel struct {
	ChanName string
	Ms       int
}

func (sc *StopChannel) screenObj() {}
func (sc *StopChannel) labelObj()  {}
