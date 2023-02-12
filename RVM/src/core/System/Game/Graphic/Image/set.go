package image

/*
#cgo CFLAGS: -I./../../../../sdl/include
#cgo LDFLAGS: -L./../../../../sdl/lib -lSDL2 -lSDL2main -lSDL2_image

#include <SDL.h>
#include <SDL_image.h>
*/
import "C"
import "log"

func (i *Image) SetImageAlpha(name string, alpha int) {
	i.lock.Lock()
	defer i.lock.Unlock()

	if image, ok := i.images[name]; !ok {
		log.Fatalf("Image Name Error : got - %s", name)
	} else {
		i.ChangeTextureAlpha(image.texture, alpha)
	}
}
