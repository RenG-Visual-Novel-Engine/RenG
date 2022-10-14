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
	Channels map[string]*Channel
	Musics   map[string]*st.Mix_Music
	Chuncks  map[string]*st.Mix_Chunk
}

func Init() *Audio {
	// (frequency, format, channels. chuncksize, device, allowed_changes)
	if C.Mix_OpenAudioDevice(44100, st.MIX_DEFAULT_FORMAT, 2, 2048, nil,
		st.SDL_AUDIO_ALLOW_FREQUENCY_CHANGE|st.SDL_AUDIO_ALLOW_CHANNELS_CHANGE) < 0 {
		return nil
	}
	return &Audio{
		Channels: map[string]*Channel{
			"music": NewChannel(),
			"sound": NewChannel(),
			"voice": NewChannel(),
		},
		Musics:  make(map[string]*st.Mix_Music),
		Chuncks: make(map[string]*st.Mix_Chunk),
	}
}

func (a *Audio) Close() {
	C.Mix_CloseAudio()
}
