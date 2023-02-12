package obj

type Screen struct {
	Obj []ScreenObject
}

type Text struct {
	Text     string
	FontName string
	T        Transform
	Color    Color
	TypingFX bool
}

func (t *Text) screenObj() {}

type TextPointer struct {
	Text     *string
	FontName string
	T        Transform
	Color    Color
	TypingFX bool
}

func (t *TextPointer) screenObj() {}

type Timer struct {
	Time float64
	Do   func()
}

func (t *Timer) screenObj() {}
