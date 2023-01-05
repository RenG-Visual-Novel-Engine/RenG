package obj

type Transform struct {
	Xpos, Ypos   int
	Xsize, Ysize int
	Rotate       int
}

type Character struct {
	Name  string
	Color Color
}

type Color struct {
	R int
	G int
	B int
	A int
}
