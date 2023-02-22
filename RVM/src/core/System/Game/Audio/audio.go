package audio

/*
#cgo CFLAGS: -I./../../../sdl/include
#cgo LDFLAGS: -L./../../../sdl/lib -lSDL2 -lSDL2main -lSDL2_mixer

#include <SDL.h>
#include <SDL_mixer.h>
*/
import "C"
import (
	"fmt"
	"unsafe"
)

type Audio struct {
	Music    *Music
	Channels map[string]*Channel
	musics   map[string]*C.Mix_Music
	chunks   map[string]*C.Mix_Chunk
}

func Init() *Audio {
	// (frequency, format, channels. chuncksize, device, allowed_changes)
	if C.Mix_OpenAudioDevice(44100, C.MIX_DEFAULT_FORMAT, 2, 2048, nil,
		C.SDL_AUDIO_ALLOW_FREQUENCY_CHANGE|C.SDL_AUDIO_ALLOW_CHANNELS_CHANGE) < 0 {
		return nil
	}
	return &Audio{
		Music: NewMusic(),
		Channels: map[string]*Channel{
			"sound": NewChannel(),
			"voice": NewChannel(),
		},
		musics: make(map[string]*C.Mix_Music),
		chunks: make(map[string]*C.Mix_Chunk),
	}
}

func (a *Audio) Close() {

	//TODO
	for _, music := range a.musics {
		C.Mix_FreeMusic(music)
	}

	for _, chunk := range a.chunks {
		C.Mix_FreeChunk(chunk)
	}

	C.Mix_CloseAudio()
}

func (a *Audio) MakeChan(chanName string) {
	a.Channels[chanName] = NewChannel()
}

func (a *Audio) PlayMusic(path string, loop bool, ms int) error {
	m, ok := a.musics[path]
	if !ok {
		err := a.AddMusic(path)
		if err != nil {
			return err
		}
		m = a.musics[path]
	}

	if ms > 0 {
		a.Music.PlayWithFadeIn(m, loop, ms)
	} else if ms == 0 {
		a.Music.Play(m, loop)
	}

	return nil
}

func (a *Audio) StopMusic(ms int) {
	if ms > 0 {
		a.Music.StopWithFadeOut(ms)
	} else if ms == 0 {
		a.Music.Stop()
	}
}

func (a *Audio) PlayChannel(channelName string, path string) error {
	channel, ok := a.Channels[channelName]
	if !ok {
		channel = NewChannel()
		a.Channels[channelName] = channel
	}

	chunk, ok := a.chunks[path]
	if !ok {
		err := a.AddChunck(path)
		if err != nil {
			return err
		}
		chunk = a.chunks[path]
	}

	channel.Play(chunk)

	return nil
}

func (a *Audio) AddMusic(path string) error {
	cp := C.CString(path)
	defer C.free(unsafe.Pointer(cp))

	music := (*C.Mix_Music)(C.Mix_LoadMUS(cp))
	if music == nil {
		return fmt.Errorf("Music Load Error path : %s", path)
	}
	a.musics[path] = music

	return nil
}

func (a *Audio) AddChunck(path string) error {
	cp := C.CString(path)
	defer C.free(unsafe.Pointer(cp))

	rb := C.CString("rb")
	defer C.free(unsafe.Pointer(rb))

	chunk := (*C.Mix_Chunk)(C.Mix_LoadWAV_RW(C.SDL_RWFromFile(cp, rb), C.int(1)))
	if chunk == nil {
		return fmt.Errorf("Chunk Load Error path : %s", path)
	}
	a.chunks[path] = chunk

	return nil
}
