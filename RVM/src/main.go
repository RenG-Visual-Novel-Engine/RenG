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
	// if len(os.Args) < 2 {
	// 	return
	//}

	// _, err := file.ReadRGOCDir(os.Args[1])
	//if err != nil {
	//	panic(err)
	//}

	s := system.Init("테스트 용도", 1280, 720)
	defer s.Close()

	a := audio.Init()
	defer a.Close()

	s.Render()

}
