package game

import "RenG/RVM/src/core/obj"

func (g *Game) StartLabel(name string) {
	bps := g.Graphic.GetCurrentTopRenderBps() + 1
	g.Graphic.AddScreenRenderBuffer()

	g.screenBps[name] = bps

	g.labelCallStack = append(g.labelCallStack, name)
	jumpLabel := g.labelEval(g.labels[name], bps)

	for jumpLabel != "" {
		g.labelCallStack = nil
		g.labelCallStack = append(g.labelCallStack, name)
		jumpLabel = g.labelEval(g.labels[jumpLabel], bps)
	}

	bps = g.screenBps[name]
	delete(g.screenBps, name)

	g.Event.DeleteScreenAllKeyEvent(name)

	for target, target_bps := range g.screenBps {
		if target_bps > bps {
			g.screenBps[target] = target_bps - 1
		}
	}

	g.Graphic.DeleteScreenRenderBuffer(bps)
}

func (g *Game) labelEval(label *obj.Label, bps int) string {
	for _, object := range label.Obj {
		switch object := object.(type) {
		case *obj.Show:
			g.evalShow(object, bps)
		case *obj.Hide:
		case *obj.PlayMusic:
			g.Audio.PlayMusic(g.path+object.Path, object.Loop)
		case *obj.PlayVideo:
			g.evalPlayVideo(object, bps)
		case *obj.Say:
		case *obj.Text:
			g.evalText(object, bps)
		}
	}
	
	return ""
}
