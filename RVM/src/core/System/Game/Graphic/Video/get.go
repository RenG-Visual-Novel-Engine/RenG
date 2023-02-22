package video

/*
#cgo CFLAGS: -I./../../../../sdl/include
#cgo LDFLAGS: -L./../../../../sdl/lib -lSDL2 -lSDL2main

#include <SDL.h>
*/
import "C"
import "RenG/RVM/src/core/globaltype"

func (v *Video) GetVideoNameByTexture(t *globaltype.SDL_Texture) string {
	v.Lock()
	defer v.Unlock()

	for name, video := range v.V {
		if video.texture == (*C.SDL_Texture)(t) {
			return name
		}
	}

	return ""
}

func (v *Video) GetVideoTexture(name string) *C.SDL_Texture {
	v.Lock()
	defer v.Unlock()

	if video, ok := v.V[name]; ok {
		return video.texture
	}
	return nil
}

func (v *Video) GetNowPlaying(name string) bool {
	v.Lock()
	defer v.Unlock()

	if _, ok := v.V[name]; ok {
		return v.V[name].nowPlaying == 1
	}
	return false
}
