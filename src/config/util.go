package config

import (
	"RenG/src/core"
)

func DeleteScreen(name string) {
	screen := ScreenAllIndex[name]

	LayerMutex.Lock()
	for i := 0; i < screen.Count; i++ {
		LayerList.Layers[2].DeleteTexture(screen.First)
	}
	LayerMutex.Unlock()

	ScreenTextureIndex = append(ScreenTextureIndex[:screen.First], ScreenTextureIndex[screen.First+screen.Count:]...)
	ScreenIndex -= screen.Count

	delete(ScreenAllIndex, name)

	for key, screens := range ScreenAllIndex {
		if screens.First > screen.First {
			ScreenAllIndex[key] = Screen{
				First: screens.First - screen.Count,
				Count: screens.Count,
			}
		}
	}
}

func AddScreenTextureIndex(texture *core.SDL_Texture) {
	ScreenTextureIndex = append(ScreenTextureIndex, texture)
	ScreenIndex++
}

func ChangeScreenTextureIndex(texture *core.SDL_Texture, index int) {
	if len(ScreenTextureIndex) <= index {
		ScreenTextureIndex = append(ScreenTextureIndex, texture)
	} else {
		ScreenTextureIndex[index] = texture
	}
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

func DeleteAllLayerTexture() {
	cnt := len(LayerList.Layers)

	LayerMutex.Lock()
	for i := 0; i < cnt; i++ {
		LayerList.Layers[i].DeleteAllTexture()
	}
	LayerMutex.Unlock()
}

func StopAllChannel() {
	core.StopMusic(-1)
}
