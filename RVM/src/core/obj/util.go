package obj

type Code struct {
	Func func()
}

func (c *Code) labelObj()  {}
func (c *Code) screenObj() {}

type Vector2 struct {
	X, Y int
}

type Color struct {
	R int
	G int
	B int
	A int
}
