package audio

/*
#cgo CFLAGS: -I./../../../sdl/include
#cgo LDFLAGS: -L./../../../sdl/lib -lSDL2 -lSDL2main -lSDL2_mixer

#include <SDL.h>
#include <SDL_mixer.h>
*/
import "C"

type Channel struct {
	chanIndex int
	volume    int
}

var (
	assignedChannel = 0
)

func NewChannel() *Channel {
	ch := &Channel{
		chanIndex: assignedChannel,
		volume:    64, // 0 ~ 128
	}

	assignedChannel++

	return ch
}

//func (c *Channel) Play(chunk *t.Mix_Chunk) error {

//}
