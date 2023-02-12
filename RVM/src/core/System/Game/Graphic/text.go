package graphic

/*
#cgo CFLAGS: -I./../../../sdl/include
#cgo LDFLAGS: -L./../../../sdl/lib -lSDL2 -lSDL2main -lSDL2_image -lSDL2_ttf

#include <SDL.h>
#include <SDL_image.h>
#include <SDL_ttf.h>

SDL_Color CreateColor(int r, int g, int b)
{
	SDL_Color textColor = { r, g, b };
	return textColor;
}
*/
import "C"
import (
	"RenG/RVM/src/core/globaltype"
	"RenG/RVM/src/core/obj"
	"math"
	"time"
	"unsafe"
)

func (g *Graphic) GetTextTexture(text, fontName string, color obj.Color) (*globaltype.SDL_Texture, int, int) {
	g.lock.Lock()
	defer g.lock.Unlock()

	cText := C.CString(text)
	defer C.free(unsafe.Pointer(cText))

	surface := C.TTF_RenderUTF8_Blended_Wrapped(
		(*C.TTF_Font)(g.fonts[fontName].Font),
		cText,
		C.CreateColor(C.int(color.R), C.int(color.G), C.int(color.B)),
		C.uint(g.fonts[fontName].LimitPixels),
	)
	defer C.SDL_FreeSurface(surface)

	texture := C.SDL_CreateTextureFromSurface((*C.SDL_Renderer)(g.renderer), surface)

	C.SDL_SetTextureBlendMode(texture, C.SDL_BLENDMODE_BLEND)
	C.SDL_SetTextureAlphaMod((texture), C.uchar(color.A))

	return (*globaltype.SDL_Texture)(texture), int(surface.w), int(surface.h)
}

func (g *Graphic) UpdateTypingFX() {
	g.lock.Lock()
	defer g.lock.Unlock()

	for _, screen := range g.typingFXs {
		for n, typingFX := range screen {
			s := time.Since(typingFX.StartTime).Seconds()

			if s >= typingFX.Duration {
				g.renderBuffer[typingFX.Bps][typingFX.Index].texture = typingFX.Data[len(typingFX.Data)-1].Texture
				g.renderBuffer[typingFX.Bps][typingFX.Index].transform = typingFX.Data[len(typingFX.Data)-1].Transform
				screen = append(screen[:n], screen[n+1:]...)
				continue
			}

			g.renderBuffer[typingFX.Bps][typingFX.Index].texture = typingFX.Data[int(math.Round(float64(len(typingFX.Data)-1)*(s/typingFX.Duration)))].Texture
			g.renderBuffer[typingFX.Bps][typingFX.Index].transform = typingFX.Data[int(math.Round(float64(len(typingFX.Data)-1)*(s/typingFX.Duration)))].Transform
		}
	}
}

func (g *Graphic) RegisterTypingFX(
	data []struct {
		Texture   *globaltype.SDL_Texture
		Transform obj.Transform
	},
	name string,
	duration float64,
	bps int,
	index int,
) {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.typingFXs[name] = append(g.typingFXs[name], struct {
		Data []struct {
			Texture   *globaltype.SDL_Texture
			Transform obj.Transform
		}
		Duration  float64
		StartTime time.Time
		Bps       int
		Index     int
	}{
		Data:      data,
		Duration:  duration,
		StartTime: time.Now(),
		Bps:       bps,
		Index:     index,
	})
}

func (g *Graphic) DeleteTypingFXByScreenName(screenName string) {
	g.lock.Lock()
	defer g.lock.Unlock()

	delete(g.typingFXs, screenName)
}

func (g *Graphic) RegisterTextMemPool(screenName string, texture *globaltype.SDL_Texture) {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.textMemPool[screenName] = append(g.textMemPool[screenName], texture)
}

func (g *Graphic) DestroyScreenTextTexture(screenName string) {
	g.lock.Lock()
	defer g.lock.Unlock()

	for _, t := range g.textMemPool[screenName] {
		C.SDL_DestroyTexture((*C.SDL_Texture)(t))
	}
}
