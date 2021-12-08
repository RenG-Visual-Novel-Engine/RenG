package main

import (
	"RenG/src/config"
	"RenG/src/core"
)

func Clear() {
	config.TextureList.DestroyAll()
	config.MusicList.FreaAll()
	config.ChunkList.FreeAll()
	core.Close(config.Window, config.Renderer)
}
