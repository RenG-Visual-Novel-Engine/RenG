package obj

import (
	event "RenG/RVM/src/core/System/Game/Event"
	animation "RenG/RVM/src/core/System/Game/Graphic/Animation"
)

type Object interface {
	ScreenObject
	LabelObject
}

type ScreenObject interface {
	screenObj()
}

type LabelObject interface {
	labelObj()
}

type Show struct {
	Name  string
	T     Transform
	Anime *animation.Anime
}

func (s *Show) screenObj() {}
func (s *Show) labelObj()  {}

type Hide struct {
	Name string
}

func (h *Hide) labelObj() {}

type PlayMusic struct {
	Path string
	Loop bool
}

func (pa *PlayMusic) screenObj() {}
func (pa *PlayMusic) labelObj()  {}

type PlayVideo struct {
	Name  string
	T     Transform
	Anime *animation.Anime
}

func (pv *PlayVideo) screenObj() {}
func (pv *PlayVideo) labelObj()  {}

type Say struct {
	Character Character
	Text      string
	Color     Color
	TypingFX  bool
}

func (s *Say) labelObj() {}

type Key struct {
	Down, Up func(*event.EVENT_Key)
}

func (k *Key) screenObj() {}

type Text struct {
	Text     string
	FontName string
	T        Transform
	Color    Color
}

func (t *Text) screenObj() {}
func (t *Text) labelObj()  {}
