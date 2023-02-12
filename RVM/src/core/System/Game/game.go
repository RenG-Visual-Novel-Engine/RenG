package game

import (
	audio "RenG/RVM/src/core/System/Game/Audio"
	event "RenG/RVM/src/core/System/Game/Event"
	graphic "RenG/RVM/src/core/System/Game/Graphic"
	"RenG/RVM/src/core/obj"
	"sync"
)

type Game struct {
	Graphic *graphic.Graphic

	Audio *audio.Audio

	// Private : 임의로 접근하지 마세요.
	Event *event.Event

	lock sync.Mutex

	width, height int
	path          string

	screens   map[string]*obj.Screen
	screenBps map[string]int
	labels    map[string]*obj.Label

	labelCallStack []string

	TextSpeed float64

	// Public : Say 스크린의 이름을 저장합니다.
	SayScreenName string

	// Private : Say 명령어로 변경된 캐릭터가 저장됩니다.
	nowName *string

	// Private : Say 명령어로 변경된 대사가 저장됩니다.
	nowText *string

	// Private : 현재 재생 중인 음악의 Path가 저장됩니다.
	nowMusic string
}

func Init(g *graphic.Graphic, a *audio.Audio, p string, w, h int, nN *string, nT *string) *Game {
	return &Game{
		Graphic:        g,
		Audio:          a,
		Event:          event.Init(),
		width:          w,
		height:         h,
		path:           p,
		screens:        make(map[string]*obj.Screen),
		screenBps:      make(map[string]int),
		labels:         make(map[string]*obj.Label),
		labelCallStack: []string{},
		TextSpeed:      40.0,
		nowName:        nN,
		nowText:        nT,
	}
}

func (g *Game) Close() {
	g.Graphic.Close()
	g.Audio.Close()
	g.Event.Close()
}

func (g *Game) GameStart(
	firstLabel string,
	sayLabel string,
) {
	g.SayScreenName = sayLabel
	go g.StartLabel(firstLabel)
}

func (g *Game) Register(
	screens map[string]*obj.Screen,
	labels map[string]*obj.Label,
	images map[string]string,
	videos map[string]string,
	fonts map[string]struct {
		Path        string
		Size        int
		LimitPixels int
	},
) {
	g.registerScreens(screens)
	g.registerLabels(labels)
	g.Graphic.RegisterImages(images)
	g.Graphic.RegisterVideos(videos)
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
