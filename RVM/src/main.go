package main

import (
	system "RenG/RVM/src/core/System"
	"RenG/RVM/src/file"
	"os"
	"runtime"
)

func init() {
	runtime.LockOSThread()
}

func main() {
	if len(os.Args) < 2 {
		return
	}

	_, err := file.ReadRGOCDir(os.Args[1])
	if err != nil {
		panic(err)
	}

	s := system.Init("테스트 용도", 1280, 720)
	defer s.Close()

}
