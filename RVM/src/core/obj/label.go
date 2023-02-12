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
