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

func (s *Storage) SavaData(g *game.Game, key string) {
	screenData := g.GetShowScreenNamesWithoutSayScreen()

	var datas []string

	for _, screen := range screenData {
		var screenDataString string

		screenDataString += screen + "&"
		textures := g.GetScreenTextureNames(screen)
		screenDataString += strings.Join(textures, ",")

		datas = append(datas, screenDataString)
	}

	s.SetStringValue(
		key,
		time.Now().Format(time.RFC3339)+";"+
			g.NowlabelName+";"+
			g.NowMusic+";"+
			strconv.Itoa(g.NowlabelIndex)+";"+
			strings.Join(datas, "|"),
	)
}

func (s *Storage) LoadData(key string) *struct {
	Time              string
	Data              string
	CurrentLabelName  string
	CurrentMusicName  string
	CurrentLabelIndex int
} {
	v := s.GetStringValue(key)

	if v == "" {
		return nil
	}
	datas := strings.Split(v, ";")

	index, _ := strconv.Atoi(strings.Split(v, ";")[3])

	return &struct {
		Time              string
		Data              string
		CurrentLabelName  string
		CurrentMusicName  string
		CurrentLabelIndex int
	}{
		Time:              datas[0],
		Data:              datas[4],
		CurrentLabelName:  datas[1],
		CurrentMusicName:  datas[2],
		CurrentLabelIndex: index,
	}
}
