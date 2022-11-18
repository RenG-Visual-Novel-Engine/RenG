package main

import (
	audio "RenG/RVM/src/core/Audio"
	system "RenG/RVM/src/core/System"
	"runtime"
)

func init() {
	runtime.LockOSThread()
}

func main() {
	s := system.Init("테스트 용도", 1280, 720)
	defer s.Close()

	a := audio.Init()
	defer a.Close()

	a.PlayMusic("D:\\program\\Go\\src\\RenG\\game\\music\\TrackTribe.mp3", true)

	s.Render()

}
