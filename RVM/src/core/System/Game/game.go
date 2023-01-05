package game

import (
	audio "RenG/RVM/src/core/System/Game/Audio"
	graphic "RenG/RVM/src/core/System/Game/Graphic"
	"RenG/RVM/src/core/obj"
	"sync"
)

type Game struct {
	lock sync.Mutex

	Graphic *graphic.Graphic
	Audio   *audio.Audio

	labels map[string]*obj.Label

	nowLabel string
}

func Init(g *graphic.Graphic, a *audio.Audio) *Game {
	return &Game{
		Graphic:  g,
		Audio:    a,
		labels:   make(map[string]*obj.Label),
		nowLabel: "",
	}
}

func (g *Game) Close() {
	g.Graphic.Close()
	g.Audio.Close()
}

func (g *Game) RegisterLabels(labels map[string]*obj.Label) {
	g.lock.Lock()
	defer g.lock.Unlock()

	for name, label := range labels {
		g.labels[name] = label
	}
}
