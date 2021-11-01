package config

import (
	"RenG/src/core"
)

// TODO
func DeleteScreen(name string) {
	indexes := ScreenHasIndex[name]

	for i, index := range indexes {
		num := ScreenTextureIndex[index-i]
		DeleteScreenTextureIndex(ScreenTextureHasIndex(num))
	}

	delete(ScreenHasIndex, name)
}

func AddScreenTextureIndex(texture *core.SDL_Texture) {
	ScreenTextureIndex = append(ScreenTextureIndex, texture)
	ScreenIndex++
}

func DeleteScreenTextureIndex(index int) {
	ScreenTextureIndex = append(ScreenTextureIndex[:index], ScreenTextureIndex[index+1:]...)
}

func ScreenTextureHasIndex(texture *core.SDL_Texture) int {
	result := 0
	for _, t := range ScreenTextureIndex {
		if t == texture {
			break
		}
		result++
	}
	return result
}

func AddShowTextureIndex(texture *core.SDL_Texture) {
	ShowTextureIndex = append(ShowTextureIndex, texture)
	ShowIndex++
}

func ShowTextureHasIndex(texture *core.SDL_Texture) int {
	result := 0
	for _, t := range ShowTextureIndex {
		if t == texture {
			break
		}
		result++
	}
	return result
}

func DeleteShowTextureIndex(index int) {
	ShowTextureIndex = append(ShowTextureIndex[:index], ShowTextureIndex[index+1:]...)
}

func DeleteAllShowTextureIndex() {
	for i := 0; i < len(ShowTextureIndex); i++ {
		ShowTextureIndex = append(ShowTextureIndex[:0], ShowTextureIndex[1:]...)
	}
}
