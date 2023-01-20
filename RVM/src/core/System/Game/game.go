package game

import (
	audio "RenG/RVM/src/core/System/Game/Audio"
	event "RenG/RVM/src/core/System/Game/Event"
	graphic "RenG/RVM/src/core/System/Game/Graphic"
	"RenG/RVM/src/core/obj"
)

type Game struct {
	Graphic *graphic.Graphic
	Audio   *audio.Audio
	Event   *event.Event

	path string

	screens   map[string]*obj.Screen
	screenBps map[string]int
	labels    map[string]*obj.Label

	labelCallStack []string
}

func Init(g *graphic.Graphic, a *audio.Audio, p string) *Game {
	return &Game{
		Graphic:        g,
		Audio:          a,
		Event:          event.Init(),
		path:           p,
		screens:        make(map[string]*obj.Screen),
		screenBps:      make(map[string]int),
		labels:         make(map[string]*obj.Label),
		labelCallStack: []string{},
	}
}

func (g *Game) Close() {
	g.Graphic.Close()
	g.Audio.Close()
	g.Event.Close()
}

func (g *Game) Register(
	screens map[string]*obj.Screen,
	labels map[string]*obj.Label,
	images map[string]string,
	videos map[string]string,
	fonts map[string]struct {
		Path string
		Size int
	},
) {
	g.registerScreens(screens)
	g.registerLabels(labels)
	g.Graphic.RegisterImages(images)
	g.Graphic.RegisterVideos(videos)
	g.Graphic.RegisterFonts(fonts)
}

// key : name, value : ui.Screen
func (g *Game) registerScreens(screens map[string]*obj.Screen) {
	for name, screen := range screens {
		g.screens[name] = screen
	}
}

func (g *Game) registerLabels(labels map[string]*obj.Label) {
	for name, label := range labels {
		g.labels[name] = label
	}
}
