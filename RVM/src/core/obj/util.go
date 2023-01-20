package obj

type Transform struct {
	Pos    Vector2
	Size   Vector2
	Rotate int
}

type Vector2 struct {
	X, Y int
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
