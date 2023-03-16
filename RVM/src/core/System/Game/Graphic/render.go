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
	"strconv"
	"strings"
)

func (g *Graphic) Render() {
	g.userLock.Lock()
	g.sayLock.Lock()
	g.lock.Lock()
	g.Video.Lock()

	C.SDL_RenderClear((*C.SDL_Renderer)(g.renderer))
	C.SDL_SetRenderDrawColor(
		(*C.SDL_Renderer)(g.renderer),
		C.uchar(0x13), C.uchar(0x13), C.uchar(0x12), C.uchar(0xFF),
	)

	x, y := g.GetCurrentWindowSize()

	for i := 0; i < len(g.renderBuffer); i++ {
		for j := 0; j < len(g.renderBuffer[i]); j++ {
			r := C.CreateRect(
				C.int(float32(g.renderBuffer[i][j].transform.Pos.X)*float32(x)/float32(g.width)),
				C.int(float32(g.renderBuffer[i][j].transform.Pos.Y)*float32(y)/float32(g.height)),
				C.int(float32(g.renderBuffer[i][j].transform.Size.X)*float32(x)/float32(g.width)),
				C.int(float32(g.renderBuffer[i][j].transform.Size.Y)*float32(y)/float32(g.height)),
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
	g.lock.Unlock()
	g.sayLock.Unlock()
	g.userLock.Unlock()
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

func (g *Graphic) DeleteScreenTextureRenderBuffer(bps, index int) {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.renderBuffer[bps] = append(g.renderBuffer[bps][:index], g.renderBuffer[bps][index+1:]...)
}

func (g *Graphic) GetCurrentRenderBufferTextureNameANDTransformByBPS(bps int) []string {
	g.lock.Lock()
	defer g.lock.Unlock()

	var ret []string

	for _, t := range g.renderBuffer[bps] {
		name := "I#" + g.Image.GetImgaeTextureName(t.texture)
		if name == "I#" {
			name = "V#"
			v, l := g.Video.GetVideoNameANDLoopByTexture(t.texture)
			name += v + "?" + strconv.Itoa(l)
		}

		format := strings.Join(
			[]string{
				name,
				strconv.Itoa(t.transform.Pos.X),
				strconv.Itoa(t.transform.Pos.Y),
				strconv.Itoa(t.transform.Size.X),
				strconv.Itoa(t.transform.Size.Y),
				strconv.Itoa(t.transform.Rotate),
			},
			"?",
		)

		if name != "I#" && name != "V#" {
			ret = append(ret, format)
		}
	}

	return ret
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

func (g *Graphic) GetCurrentWindowSize() (x, y int) {
	var xsize, ysize C.int
	C.SDL_GetWindowSize((*C.SDL_Window)(g.window), &xsize, &ysize)
	return int(xsize), int(ysize)
}
