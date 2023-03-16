package game

import (
	"RenG/RVM/src/core/obj"
	"time"
)

func (g *Game) startLabel(name string, index int, sayScreenName string) {
	g.SayScreenName = sayScreenName

	// label을 screen처럼 등록
	bps, ok := g.screenBps[name]
	if !ok {
		g.screenBps[name] = g.Graphic.GetCurrentTopRenderBps() + 1
		bps = g.screenBps[name]
		g.Graphic.AddScreenRenderBuffer()
	}

	//config 정보 삽입
	g.LabelManager.SetNowLabelName(name)
	g.LabelManager.SetNowLabelIndex(index)
	g.LabelManager.AddCallStack(name, index)
	g.Event.TopScreenName = name

	for {
		g.labelEval(g.LabelManager.GetNowLabelObject(), g.LabelManager.GetNowLabelName(), bps)

		if !g.LabelManager.NextLabelObject() {
			break
		}
	}

	g.InActiveScreen(g.SayScreenName)
}

func (g *Game) labelEval(object obj.LabelObject, name string, bps int) {
	switch object := object.(type) {
	case *obj.Code:
		object.Func()
	case *obj.Jump:
		g.LabelManager.JumpLabel(object.LabelName)
	case *obj.Call:
		g.LabelManager.CallLabel(object.LabelName)
	case *obj.PlayChannel:
		g.Audio.PlayChannel(object.ChanName, g.path+object.Path)
	case *obj.Show:
		g.evalShow(object, name, bps)
	case *obj.Hide:
		g.evalHide(object, name, bps)
	case *obj.PlayMusic:
		g.NowMusic = object.Path
		g.Audio.PlayMusic(g.path+object.Path, object.Loop, object.Ms)
	case *obj.StopMusic:
		g.NowMusic = ""
		g.Audio.StopMusic(object.Ms)
	case *obj.PlayVideo:
		g.evalPlayVideo(object, name, bps)
	case *obj.Say:
		g.evalSay(object)
	case *obj.Pause:
		g.InActiveScreen(g.SayScreenName)
		time.Sleep(time.Duration(object.Time) * time.Second)
	}
}
