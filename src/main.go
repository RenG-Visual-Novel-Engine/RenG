package main

import (
	"runtime"
)

func init() {
	runtime.LockOSThread()
}

func main() {
	if Init() {
		go Run()
	}
	MainLoop()
	Clear()
}
