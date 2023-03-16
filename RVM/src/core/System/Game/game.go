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

	LabelManager *LabelManager

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
		Graphic:   g,
		Audio:     a,
		Event:     event.Init(),
		width:     w,
		height:    h,
		path:      p,
		screens:   make(map[string]*obj.Screen),
		screenBps: make(map[string]int),
		LabelManager: &LabelManager{
			labels: make(map[string]*obj.Label),
			labelCallStack: []struct {
				Name  string
				Index int
			}{},
		},
		TextSpeed: 40.0,
		nowName:   nN,
		nowText:   nT,
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
	data *struct {
		Time                  string
		Data                  string
		CurrentLabelName      string
		CurrentMusicName      string
		CurrentLabelIndex     int
		CurrentLabelCallStack []struct {
			Name  string
			Index int
		}
	},
) {
	if data != nil {
		g.loadData(data)
	}

	if data != nil {
		go g.startLabel(data.CurrentLabelName, data.CurrentLabelIndex, sayLabel)
	} else {
		go g.startLabel(firstLabel, 0, sayLabel)
	}
}

func (g *Game) loadData(
	data *struct {
		Time                  string
		Data                  string
		CurrentLabelName      string
		CurrentMusicName      string
		CurrentLabelIndex     int
		CurrentLabelCallStack []struct {
			Name  string
			Index int
		}
	},
) {
	g.LabelManager.SetCallStack(data.CurrentLabelCallStack[:len(data.CurrentLabelCallStack)-1])
	g.LabelManager.SetNowLabelName(data.CurrentLabelName)
	g.LabelManager.SetNowLabelIndex(data.CurrentLabelIndex)

	g.NowMusic = data.CurrentMusicName
	g.Audio.PlayMusic(g.path+data.CurrentMusicName, true, 1000)

	screenNames := strings.Split(data.Data, "|")

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

				param := strings.Split(data[1], "?")

				isLoop, _ := strconv.Atoi(param[1])
				xPos, _ := strconv.Atoi(param[2])
				yPos, _ := strconv.Atoi(param[3])
				xSize, _ := strconv.Atoi(param[4])
				ySize, _ := strconv.Atoi(param[5])
				rotate, _ := strconv.Atoi(param[6])

				g.Graphic.AddScreenTextureRenderBuffer(
					bps,
					g.Graphic.GetVideoTexture(strings.Split(data[1], "?")[0]),
					obj.Transform{
						Pos: obj.Vector2{
							X: xPos,
							Y: yPos,
						},
						Size: obj.Vector2{
							X: xSize,
							Y: ySize,
						},
						Rotate: rotate,
					},
				)

				g.Graphic.VideoStart(
					strings.Split(strings.Split(screenName, "-")[1], "&")[0],
					param[0],
					!(isLoop == 0),
				)
			case "I":
				// TODO : transform 저장
				param := strings.Split(data[1], "?")

				xPos, _ := strconv.Atoi(param[1])
				yPos, _ := strconv.Atoi(param[2])
				xSize, _ := strconv.Atoi(param[3])
				ySize, _ := strconv.Atoi(param[4])
				rotate, _ := strconv.Atoi(param[5])

				g.Graphic.AddScreenTextureRenderBuffer(
					bps,
					g.Graphic.Image.GetImageTexture(param[0]),
					obj.Transform{
						Pos: obj.Vector2{
							X: xPos,
							Y: yPos,
						},
						Size: obj.Vector2{
							X: xSize,
							Y: ySize,
						},
						Rotate: rotate,
					},
				)
			}
		}
	}
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
		g.LabelManager.labels[name] = label
	}
}
