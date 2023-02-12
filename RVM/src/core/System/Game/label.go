package game

import (
	event "RenG/RVM/src/core/System/Game/Event"
	"RenG/RVM/src/core/obj"
)

func (g *Game) StartLabel(name string) {
	bps := g.Graphic.GetCurrentTopRenderBps() + 1
	g.Graphic.AddScreenRenderBuffer()

	g.Event.TopScreenName = name

	g.screenBps[name] = bps

	g.labelCallStack = append(g.labelCallStack, name)
	jumpLabel := g.labelEval(g.labels[name], name, bps)

	for jumpLabel != "" {
		g.Event.TopScreenName = name
		g.labelCallStack = nil
		g.labelCallStack = append(g.labelCallStack, jumpLabel)
		jumpLabel = g.labelEval(g.labels[jumpLabel], jumpLabel, bps)
	}

	bps = g.screenBps[name]
	delete(g.screenBps, name)

	g.Event.DeleteAllScreenEvent(name)

	for target, target_bps := range g.screenBps {
		if target_bps > bps {
			g.screenBps[target] = target_bps - 1
		}
		if bps == 0 && target_bps == 1 {
			g.Event.TopScreenName = target
		}
	}

	g.Graphic.DeleteScreenRenderBuffer(bps)
}

func (g *Game) labelEval(label *obj.Label, name string, bps int) string {
	for _, object := range label.Obj {
		switch object := object.(type) {
		case *obj.Show:
			g.evalShow(object, name, bps)
		case *obj.Hide:
		case *obj.PlayMusic:
			g.nowMusic = object.Path
			g.Audio.PlayMusic(g.path+object.Path, object.Loop, object.Ms)
		case *obj.StopMusic:
			g.nowMusic = ""
			g.Audio.StopMusic(object.Ms)
		case *obj.PlayVideo:
			g.evalPlayVideo(object, name, bps)
		case *obj.Say:
			g.setNowName(object.Character.Name)
			g.setNowText(object.Text)
			g.ActiveScreen(g.SayScreenName)
			ch := make(chan int, 1)
			g.Event.AddKeyEvent(g.SayScreenName, event.KeyEvent{
				Up: func(e *event.EVENT_Key) {},
				Down: func(e *event.EVENT_Key) {
					if e.KeyCode == event.SDLK_n {
						ch <- 0
					}
				},
			})
			<-ch
			g.InActiveScreen(g.SayScreenName)
		}
	}

	return ""
}
