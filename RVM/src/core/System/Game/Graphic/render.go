package graphic

/*
#cgo CFLAGS: -I./../../../sdl/include
#cgo LDFLAGS: -L./../../../sdl/lib -lSDL2 -lSDL2main -lSDL2_image -lSDL2_ttf

#include <SDL.h>
#include <SDL_image.h>
#include <SDL_ttf.h>

SDL_Rect* CreateRect(int x, int y, int w, int h)
{
	SDL_Rect* Quad = (SDL_Rect*)malloc(sizeof(SDL_Rect));
	Quad->x = x;
	Quad->y = y;
	Quad->w = w;
	Quad->h = h;
	return Quad;
}

void FreeRect(SDL_Rect* r)
{
	free(r);
}
*/
import "C"
import (
	"RenG/RVM/src/core/globaltype"
	"RenG/RVM/src/core/obj"
)

func (g *Graphic) Render() {
	g.lock.Lock()
	defer g.lock.Unlock()
	C.SDL_RenderClear((*C.SDL_Renderer)(g.renderer))
	C.SDL_SetRenderDrawColor(
		(*C.SDL_Renderer)(g.renderer),
		C.uchar(0x13), C.uchar(0x13), C.uchar(0x12), C.uchar(0xFF),
	)
	g.Video.Lock()
	for i := 0; i < len(g.renderBuffer); i++ {
		for j := 0; j < len(g.renderBuffer[i]); j++ {
			r := C.CreateRect(
				C.int(g.renderBuffer[i][j].transform.Pos.X),
				C.int(g.renderBuffer[i][j].transform.Pos.Y),
				C.int(g.renderBuffer[i][j].transform.Size.X),
				C.int(g.renderBuffer[i][j].transform.Size.Y),
			)
			C.SDL_RenderCopyEx(
				(*C.SDL_Renderer)(g.renderer),
				(*C.SDL_Texture)(g.renderBuffer[i][j].texture),
				nil,
				r,
				C.double(g.renderBuffer[i][j].transform.Rotate),
				nil,
				C.SDL_FLIP_NONE,
			)
			C.FreeRect(r)
		}
	}
	C.SDL_RenderPresent((*C.SDL_Renderer)(g.renderer))
	g.Video.Unlock()
}

func (g *Graphic) AddScreenRenderBuffer() {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.renderBuffer = append(g.renderBuffer, []struct {
		texture   *globaltype.SDL_Texture
		transform obj.Transform
	}{})
}

func (g *Graphic) DeleteScreenRenderBuffer(bps int) {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.renderBuffer = append(g.renderBuffer[:bps], g.renderBuffer[bps+1:]...)
}

func (g *Graphic) AddScreenTextureRenderBuffer(
	bps int,
	texture *globaltype.SDL_Texture,
	transform obj.Transform,
) {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.renderBuffer[bps] = append(
		g.renderBuffer[bps],
		struct {
			texture   *globaltype.SDL_Texture
			transform obj.Transform
		}{
			texture,
			transform,
		})
}

func (g *Graphic) GetCurrentTopRenderBps() int {
	g.lock.Lock()
	defer g.lock.Unlock()

	return len(g.renderBuffer) - 1
}

func (g *Graphic) GetCurrentTopScreenIndexByBps(bps int) int {
	g.lock.Lock()
	defer g.lock.Unlock()

	return len(g.renderBuffer[bps]) - 1
}
