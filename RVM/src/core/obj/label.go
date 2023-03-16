package obj

type Label struct {
	Obj []LabelObject
}

type Character struct {
	Name string
}

type Say struct {
	Character Character
	Text      string
}

func (s *Say) labelObj() {}

type Pause struct {
	Time float32
}

func (p *Pause) labelObj() {}

type Call struct {
	LabelName string
}

func (c *Call) labelObj() {}

type Jump struct {
	LabelName string
}

func (j *Jump) labelObj() {}
