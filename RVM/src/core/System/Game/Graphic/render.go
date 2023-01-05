package graphic

/*
#cgo CFLAGS: -I./../../../sdl/include
#cgo LDFLAGS: -L./../../../sdl/lib -lSDL2 -lSDL2main -lSDL2_image -lSDL2_ttf

#include <SDL.h>
#include <SDL_image.h>
#include <SDL_ttf.h>

SDL_Rect CreateRect(int x, int y, int w, int h)
{
	SDL_Rect Quad = { x, y, w, h };
	return Quad;
}
*/
import "C"

func (g *Graphic) Render() {
	g.lock.Lock()
	defer g.lock.Unlock()

	C.SDL_RenderClear((*C.SDL_Renderer)(g.renderer))
	C.SDL_SetRenderDrawColor(
		(*C.SDL_Renderer)(g.renderer),
		C.uchar(0x13), C.uchar(0x13), C.uchar(0x12), C.uchar(0xFF),
	)
	for i := 0; i < len(g.renderBuffer); i++ {
		for j := 0; j < len(g.renderBuffer[i]); j++ {
			r := C.CreateRect(
				C.int(g.renderBuffer[i][j].transform.Xpos),
				C.int(g.renderBuffer[i][j].transform.Ypos),
				C.int(g.renderBuffer[i][j].transform.Xsize),
				C.int(g.renderBuffer[i][j].transform.Ysize),
			)
			g.videos.Lock()
			C.SDL_RenderCopy(
				(*C.SDL_Renderer)(g.renderer),
				(*C.SDL_Texture)(g.renderBuffer[i][j].texture),
				nil,
				&r,
			)
			g.videos.Unlock()
		}
	}

	C.SDL_RenderPresent((*C.SDL_Renderer)(g.renderer))
}
