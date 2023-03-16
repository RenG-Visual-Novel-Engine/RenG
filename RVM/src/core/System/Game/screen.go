package game

import (
	event "RenG/RVM/src/core/System/Game/Event"
	"RenG/RVM/src/core/obj"
	"time"
)

func (g *Game) screenEval(
	so []obj.ScreenObject,
	name string, bps int,
) {
	for _, object := range so {
		switch object := object.(type) {
		case *obj.Code:
			object.Func()
		case *obj.Show:
			g.evalShow(object, name, bps)
		case *obj.PlayMusic:
			g.NowMusic = object.Path
			g.Audio.PlayMusic(g.path+object.Path, object.Loop, object.Ms)
		case *obj.StopMusic:
			g.NowMusic = ""
			g.Audio.StopMusic(object.Ms)
		case *obj.PlayChannel:
			g.Audio.PlayChannel(object.ChanName, g.path+object.Path)
		case *obj.PlayVideo:
			g.evalPlayVideo(object, name, bps)
		case *obj.Key:
			g.Event.AddKeyEvent(name, event.KeyEvent{Down: object.Down, Up: object.Up})
		case *obj.Button:
			g.evalButton(object, name, bps)
		case *obj.Bar:
			g.evalBar(object, name, bps)
		case *obj.Text:
			g.evalText(object, name, bps)
		case *obj.TextPointer:
			g.evalTextPointer(object, name, bps)
		case *obj.Timer:
			go func() {
				time.Sleep(time.Duration(object.Time) * time.Second)
				object.Do()
			}()
		}
	}
}
