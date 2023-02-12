package obj

import (
	event "RenG/RVM/src/core/System/Game/Event"
)

type Key struct {
	Down, Up func(*event.EVENT_Key)
}

func (k *Key) screenObj() {}

type Button struct {
	MainImageName  string
	HoverImageName string
	T              Transform
	Anime          []*Anime
	Action         func()
}

func (b *Button) screenObj() {}
