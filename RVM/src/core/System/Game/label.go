package game

import (
	event "RenG/RVM/src/core/System/Game/Event"
	"RenG/RVM/src/core/obj"
)

func (g *Game) labelEval(label *obj.Label, name string, bps, startIndex int) string {
	for n, object := range label.Obj {
		g.NowlabelIndex = n + startIndex
		switch object := object.(type) {
		case *obj.Show:
			g.evalShow(object, name, bps)
		case *obj.Hide:
			if object.Anime != nil {
				if object.Anime.End != nil {
					temp := object.Anime.End
					object.Anime.End = func() {
						temp()
						g.Graphic.DeleteAnimationByTextureIndex(name, object.TextureIndex)
						g.Graphic.DeleteScreenTextureRenderBuffer(bps, object.TextureIndex)
					}
				} else {
					object.Anime.End = func() {
						g.Graphic.DeleteAnimationByTextureIndex(name, object.TextureIndex)
						g.Graphic.DeleteScreenTextureRenderBuffer(bps, object.TextureIndex)
					}
				}
				g.Graphic.AddAnimation(name, object.Anime, bps, object.TextureIndex)
			} else {
				g.Graphic.DeleteAnimationByTextureIndex(name, object.TextureIndex)
				g.Graphic.DeleteScreenTextureRenderBuffer(bps, object.TextureIndex)
			}
		case *obj.PlayMusic:
			g.NowMusic = object.Path
			g.Audio.PlayMusic(g.path+object.Path, object.Loop, object.Ms)
		case *obj.StopMusic:
			g.NowMusic = ""
			g.Audio.StopMusic(object.Ms)
		case *obj.PlayVideo:
			g.evalPlayVideo(object, name, bps)
		case *obj.Say:
			*g.nowName = object.Character.Name
			*g.nowText = object.Text

			g.Graphic.SayLock()
			g.InActiveScreen(g.SayScreenName)
			g.ActiveScreen(g.SayScreenName)
			g.Graphic.SayUnlock()

			lock := make(chan int)
			g.Event.AddMouseClickEvent(g.SayScreenName, event.MouseClickEvent{
				Down: func(e *event.EVENT_MouseButton) {},
				Up: func(e *event.EVENT_MouseButton) {
					if buttons, ok := g.Event.Button[g.SayScreenName]; !ok {
						lock <- 0
					} else {
						for _, button := range buttons {
							if button.IsNowDown {
								return
							}
						}
						lock <- 0
					}
				},
			})
			<-lock
		}
	}

	g.InActiveScreen(g.SayScreenName)

	return ""
}
