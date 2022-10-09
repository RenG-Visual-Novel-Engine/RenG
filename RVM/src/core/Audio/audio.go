package audio

/*
#cgo CFLAGS: -I./../sdl/include
#cgo LDFLAGS: -L./../sdl/lib -lSDL2 -lSDL2main -lSDL2_mixer

#include <SDL.h>
#include <SDL_mixer.h>
*/
import "C"
import "RenG/RVM/src/core/st"

type Audio struct {
}

func Init() *Audio {
	// (frequency, format, channels. chuncksize, device, allowed_changes)
	if C.Mix_OpenAudioDevice(44100, st.MIX_DEFAULT_FORMAT, 2, nil, st.SDL_AUDIO_ALLOW_FREQUENCY_CHANGE|st.SDL_AUDIO_ALLOW_CHANNELS_CHANGE) < 0 {
		return nil
	}
	return &Audio{}
}

func (a *Audio) Close() {
	C.Mix_CloseDevice()
}
