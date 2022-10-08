package main

import (
	system "RenG/RVM/src/core/System"
	"runtime"
)

func init() {
	runtime.LockOSThread()
}

func main() {
	system.Init("테스트 용도", 1280, 720)
}
