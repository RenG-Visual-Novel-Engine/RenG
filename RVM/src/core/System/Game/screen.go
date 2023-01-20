package game

import (
	event "RenG/RVM/src/core/System/Game/Event"
	"RenG/RVM/src/core/obj"
	"log"
)

func (g *Game) ActiveScreen(name string) {
	screen, ok := g.screens[name]
	if !ok {
		log.Printf("Screen Name Error, got - %s\n", name)
	}

	bps := g.Graphic.GetCurrentTopRenderBps() + 1
	g.Graphic.AddScreenRenderBuffer()

	g.screenBps[name] = bps
	g.screenEval(screen.Obj, name, bps)
}

func (g *Game) InActiveScreen(name string) {
	bps := g.screenBps[name]
	delete(g.screenBps, name)

	g.Event.DeleteScreenAllKeyEvent(name)

	for target, target_bps := range g.screenBps {
		if target_bps > bps {
			g.screenBps[target] = target_bps - 1
		}
	}

	g.Graphic.DeleteScreenRenderBuffer(bps)
}

func (g *Game) screenEval(
	so []obj.ScreenObject,
	name string, bps int,
) {
	for _, object := range so {
		switch object := object.(type) {
		case *obj.Show:
			g.evalShow(object, bps)
		case *obj.PlayMusic:
			g.Audio.PlayMusic(object.Path, object.Loop)
		case *obj.PlayVideo:
			g.evalPlayVideo(object, bps)
		case *obj.Key:
			g.Event.AddKeyEvent(name, event.KeyEvent{Down: object.Down, Up: object.Up})
		case *obj.Text:
			g.evalText(object, bps)
		}
	}
}
