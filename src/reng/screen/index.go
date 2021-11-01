package screen

import "RenG/src/config"

func Set(name string, index int) {
	config.ScreenHasIndex[name] = append(config.ScreenHasIndex[name], index)
}
