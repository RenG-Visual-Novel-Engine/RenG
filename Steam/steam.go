package steam

import (
	"syscall"
	"unsafe"
)

const (
	is32Bit = unsafe.Sizeof(int(0)) == 4
)

type Steam struct {
	dll *syscall.LazyDLL
}

func Init() *Steam {
	dllName := "steam_api.dll"
	if !is32Bit {
		dllName = "steam_api64.dll"
	}

	return &Steam{
		dll: syscall.NewLazyDLL(dllName),
	}
}

func (s *Steam) SteamAPI_Init() bool {
	ret1, _, _ := s.dll.NewProc("SteamAPI_Init").Call()
	return byte(ret1) != 0
}

func (s *Steam) SteamAPI_ShutDown() {
	s.dll.NewProc("SteamAPI_Shutdown").Call()
}
