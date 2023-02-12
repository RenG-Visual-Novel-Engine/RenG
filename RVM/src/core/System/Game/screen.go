package game

import (
	event "RenG/RVM/src/core/System/Game/Event"
	"RenG/RVM/src/core/obj"
	"log"
	"time"
)

func (g *Game) ActiveScreen(name string) {
	g.lock.Lock()
	defer g.lock.Unlock()

	screen, ok := g.screens[name]
	if !ok {
		log.Printf("Screen Name Error, got - %s\n", name)
	}

	bps := g.Graphic.GetCurrentTopRenderBps() + 1
	g.Graphic.AddScreenRenderBuffer()

	g.Event.TopScreenName = name

	g.screenBps[name] = bps
	g.screenEval(screen.Obj, name, bps)
}

func (g *Game) InActiveScreen(name string) {
	g.lock.Lock()
	defer g.lock.Unlock()

	bps := g.screenBps[name]
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

	g.Graphic.DeleteAnimationByScreenName(name)
	g.Graphic.DeleteTypingFXByScreenName(name)
	g.Graphic.DestroyScreenTextTexture(name)
	g.Graphic.ScreenVideoAllStop(name)

	g.Graphic.DeleteScreenRenderBuffer(bps)
}

func (g *Game) GetScreenBps(screenName string) int {
	g.lock.Lock()
	defer g.lock.Unlock()
	
	return g.screenBps[screenName]
}

func (g *Game) screenEval(
	so []obj.ScreenObject,
	name string, bps int,
) {
	for _, object := range so {
		switch object := object.(type) {
		case *obj.Show:
			g.evalShow(object, name, bps)
		case *obj.PlayMusic:
			g.nowMusic = object.Path
			g.Audio.PlayMusic(g.path+object.Path, object.Loop, object.Ms)
		case *obj.StopMusic:
			g.nowMusic = ""
			g.Audio.StopMusic(object.Ms)
		case *obj.PlayChannel:
			g.Audio.PlayChannel(object.ChanName, g.path+object.Path)
		case *obj.PlayVideo:
			g.evalPlayVideo(object, name, bps)
		case *obj.Key:
			g.Event.AddKeyEvent(name, event.KeyEvent{Down: object.Down, Up: object.Up})
		case *obj.Button:
			g.evalButton(object, name, bps)
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
