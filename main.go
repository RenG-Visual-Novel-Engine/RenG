package main

import (
	steam "RenG/Steam"
)

// Start Point
//     역할
//      1.실행시 메인 게임 엔진 메인화면 코드가 있는 파일 주소를 인터프리터한테 넘김
//      2.파일이 제대로 작동할 준비가 되었는지 확인함
//

/*
func main() {
	if runtime.GOOS == "windows" {
		root, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		cmd := exec.Command("core\\RenG", "-r", fmt.Sprintf("%s\\RenGLauncher", root))
		cmd.Run()
	}
}
*/

/*
func main() {
	if runtime.GOOS == "windows" {
		root, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		cmd := exec.Command("core\\RenG", "-r", fmt.Sprintf("%s\\game", root))
		cmd.Run()
	}
}
*/

func main() {
	s := steam.Init()
	if s.SteamAPI_Init() {
		s.GetAppID(s.SteamUtils())
		s.SteamAPI_ShutDown()
	}
}
