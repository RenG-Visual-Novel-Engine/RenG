package main

import (
	audio "RenG/RVM/src/core/Audio"
	video "RenG/RVM/src/core/Graphic/Video"
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

	v := video.Init()
	defer v.Close()

	a.PlayMusic("D:\\program\\renpy\\SummerFlower_Mode\\game\\sounds\\ed.ogg", true)
	v.VideoInit("D:\\source\\video\\ed.webm")
	v.VideoStart(s.GetRenderer())

	s.Render(&v)

}
