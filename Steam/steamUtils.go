package steam

type SteamUtils uintptr

type AppID uint32

func (s *Steam) SteamUtils() SteamUtils {
	ret1, _, _ := s.dll.NewProc("SteamAPI_SteamUtils_v010").Call()
	return SteamUtils(ret1)
}

func (s *Steam) GetAppID(su SteamUtils) AppID {
	ret1, _, _ := s.dll.NewProc("SteamAPI_ISteamUtils_GetAppID").Call(uintptr(su))
	return AppID(ret1)
}
