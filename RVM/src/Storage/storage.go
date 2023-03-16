package storage

import (
	game "RenG/RVM/src/core/System/Game"
	"os"
	"strconv"
	"strings"
	"time"
)

type Storage struct {
	path string
	f    *os.File
}

/*
저장소를 불러옵니다.
만약 존재하지 않다면 새로 생성하여 불러옵니다.
*/
func LoadStorage(path string) *Storage {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, os.FileMode(0666))
	if err != nil {
		panic(err)
	}

	return &Storage{
		path: path,
		f:    file,
	}
}

func (s *Storage) CloseStorage() {
	s.f.Close()
}

// 2023-02-19T02:29:31+09:00;L_chapter1;\audio\Moonlight Stage 10th Remix.mp3;2;0-L_chapter1&V#test?XPOS?YPOS?XSIZE?YSIZE?ROTATE

// [TIME];[CurrentLabelName];[LabelStack];[CurrentPlayingMusic];[CurrentLabelBPS];[[BPS]-[SCREEN[TEXTURE[XPOS?YPOS?XSIZE?YSIZE?ROTATE]...]...]...]

func (s *Storage) SavaData(g *game.Game, key string) {
	screenData := g.GetShowScreenNamesWithoutSayScreen()

	var datas []string

	for _, screen := range screenData {
		var screenDataString string

		screenDataString += screen + "&"
		textures := g.GetScreenTextureNamesANDTransform(screen)
		screenDataString += strings.Join(textures, ",")

		datas = append(datas, screenDataString)
	}

	var stack []string

	for _, data := range g.LabelManager.GetCallStack() {
		stack = append(stack, data.Name+"^"+strconv.Itoa(data.Index))
	}

	s.SetStringValue(
		key,
		time.Now().Format(time.RFC3339)+";"+
			g.LabelManager.GetNowLabelName()+";"+
			g.NowMusic+";"+
			strconv.Itoa(g.LabelManager.GetNowLabelIndex())+";"+
			strings.Join(datas, "|")+";"+
			strings.Join(stack, "@"),
	)
}

func (s *Storage) LoadData(key string) *struct {
	Time                  string
	Data                  string
	CurrentLabelName      string
	CurrentMusicName      string
	CurrentLabelIndex     int
	CurrentLabelCallStack []struct {
		Name  string
		Index int
	}
} {
	v := s.GetStringValue(key)

	if v == "" {
		return nil
	}
	datas := strings.Split(v, ";")

	index, _ := strconv.Atoi(strings.Split(v, ";")[3])

	var stackData []struct {
		Name  string
		Index int
	}

	for _, str := range strings.Split(datas[5], "@") {
		s, _ := strconv.Atoi(strings.Split(str, "^")[1])
		stackData = append(stackData, struct {
			Name  string
			Index int
		}{
			strings.Split(str, "^")[0], s,
		})
	}

	return &struct {
		Time                  string
		Data                  string
		CurrentLabelName      string
		CurrentMusicName      string
		CurrentLabelIndex     int
		CurrentLabelCallStack []struct {
			Name  string
			Index int
		}
	}{
		Time:                  datas[0],
		Data:                  datas[4],
		CurrentLabelName:      datas[1],
		CurrentMusicName:      datas[2],
		CurrentLabelIndex:     index,
		CurrentLabelCallStack: stackData,
	}
}
