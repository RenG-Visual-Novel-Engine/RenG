package obj

import (
	event "RenG/RVM/src/core/System/Game/Event"
)

type Key struct {
	Down, Up func(e *event.EVENT_Key)
}

func (k *Key) screenObj() {}

type Button struct {
	MainImageName  string
	HoverImageName string
	T              Transform
	Anime          []*Anime
	Down           func(e *event.EVENT_MouseButton)
	Up             func(e *event.EVENT_MouseButton)
}

func (b *Button) screenObj() {}

type Bar struct {
	FrameImageName       string
	CursorImageName      string
	CursorHoverImageName string
	GaugeImageName       string

	FrameImageT  Transform
	CursorSize   Vector2
	StartPadding int
	EndPadding   int
	SidePadding  int
	IsVertical   bool

	MaxValue  int
	MinValue  int
	InitValue int

	Down   func(e *event.EVENT_MouseButton, value int)
	Up     func(e *event.EVENT_MouseButton, value int)
	Scroll func(e *event.EVENT_MouseMotion, value int)
}

func (b *Bar) screenObj() {}
