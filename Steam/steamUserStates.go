package steam

import (
	"unsafe"
)

type SteamUserStates uintptr

func (s *Steam) SteamUserStates() SteamUserStates {
	ret1, _, _ := s.dll.NewProc("SteamAPI_SteamUserStats_v012").Call()
	return SteamUserStates(ret1)
}

func (s *Steam) RequestCurrentStats(sus SteamUserStates) bool {
	ret1, _, _ := s.dll.NewProc("SteamAPI_ISteamUserStats_RequestCurrentStats").Call(uintptr(sus))
	return byte(ret1) != 0
}

func (s *Steam) SetAchievement(sus SteamUserStates, pchName string) bool {
	ret1, _, _ := s.dll.NewProc("SteamAPI_ISteamUserStats_SetAchievement").Call(uintptr(sus), uintptr(unsafe.Pointer(&[]byte(pchName)[0])))
	return byte(ret1) != 0
}
