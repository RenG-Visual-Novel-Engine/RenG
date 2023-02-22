package game

import (
	audio "RenG/RVM/src/core/System/Game/Audio"
	event "RenG/RVM/src/core/System/Game/Event"
	graphic "RenG/RVM/src/core/System/Game/Graphic"
	"RenG/RVM/src/core/obj"
	"strconv"
	"strings"
	"sync"
)

type Game struct {
	Graphic *graphic.Graphic
	Audio   *audio.Audio
	Event   *event.Event

	lock sync.Mutex

	width, height int
	path          string

	screens   map[string]*obj.Screen
	screenBps map[string]int
	labels    map[string]*obj.Label

	NowlabelName   string
	NowlabelIndex  int
	labelCallStack []string

	TextSpeed float64

	// Public : Say 스크린의 이름을 저장합니다.
	SayScreenName string

	// Private : Say 명령어로 변경된 캐릭터가 저장됩니다.
	nowName *string

	// Private : Say 명령어로 변경된 대사가 저장됩니다.
	nowText *string

	// Public : 현재 재생 중인 음악의 Path가 저장됩니다.
	NowMusic string
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
	firstLabel,
	sayLabel string,
) {
	g.SayScreenName = sayLabel
	go g.StartLabel(firstLabel, 0)
}

func (g *Game) GameLoad(
	currentLabel,
	sayLabel,
	textureData,
	currentMusicName string,
	currentIndex int,
) {
	g.SayScreenName = sayLabel
	g.NowMusic = currentMusicName
	screenNames := strings.Split(textureData, "|")

	// Loop 저장
	g.Audio.PlayMusic(g.path+currentMusicName, true, 1000)

	for _, screenName := range screenNames {
		bps, err := strconv.Atoi(strings.Split(screenName, "-")[0])
		if err != nil {
			panic(err)
		}
		g.screenBps[strings.Split(strings.Split(screenName, "-")[1], "&")[0]] = bps
		g.Graphic.AddScreenRenderBuffer()

		for _, texture := range strings.Split(strings.Split(strings.Split(screenName, "-")[1], "&")[1], ",") {
			data := strings.Split(texture, "#")
			switch data[0] {
			case "V":
				// TODO : transform, loop 저장
				g.Graphic.AddScreenTextureRenderBuffer(
					bps,
					g.Graphic.GetVideoTexture(data[1]),
					obj.Transform{
						Pos:  obj.Vector2{X: 0, Y: 0},
						Size: obj.Vector2{X: 1280, Y: 720},
					},
				)
				g.Graphic.VideoStart(strings.Split(strings.Split(screenName, "-")[1], "&")[0], data[1], true)
			case "I":
				// TODO : transform 저장
				g.Graphic.AddScreenTextureRenderBuffer(
					bps,
					g.Graphic.Image.GetImageTexture(data[1]),
					obj.Transform{
						Pos:  obj.Vector2{X: 0, Y: 0},
						Size: obj.Vector2{X: 1280, Y: 720},
					},
				)
			}
		}
	}

	go g.StartLabel(currentLabel, currentIndex)
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
