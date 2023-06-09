package image

/*
#cgo CFLAGS: -I./../../../../sdl/include
#cgo LDFLAGS: -L./../../../../sdl/lib -lSDL2 -lSDL2main -lSDL2_image

#include <SDL.h>
#include <SDL_image.h>

SDL_Rect* CreatesRect(int x, int y, int w, int h)
{
	SDL_Rect* Quad = (SDL_Rect*)malloc(sizeof(SDL_Rect));
	Quad->x = x;
	Quad->y = y;
	Quad->w = w;
	Quad->h = h;
	return Quad;
}

void FreesRect(SDL_Rect* r)
{
	free(r);
}
*/
import "C"
import (
	pixel "RenG/RVM/src/core/System/Game/Graphic/Image/Pixel"
	"RenG/RVM/src/core/globaltype"
	"unsafe"
)

func (i *Image) ChangeTextureAlpha(t *globaltype.SDL_Texture, alpha int) {
	C.SDL_SetTextureAlphaMod((*C.SDL_Texture)(t), C.uchar(alpha))
}

func (i *Image) CreateRGBASurface(RGBA pixel.RGBA) *C.SDL_Surface {
	i.lock.Lock()
	defer i.lock.Unlock()

	data, size := RGBA.GetPixels()
	width, height := RGBA.GetWH()

	return C.SDL_CreateRGBSurfaceWithFormatFrom(unsafe.Pointer(data), C.int(width), C.int(height), C.int(32), C.int(size*4), C.SDL_PIXELFORMAT_RGBA8888)
}

func (i *Image) CreateYUVTexture(data [8]*uint8, linesize [8]int32, Width, Height int64) *globaltype.SDL_Texture {
	i.lock.Lock()
	defer i.lock.Unlock()

	ret := C.SDL_CreateTexture((*C.SDL_Renderer)(i.renderer), C.SDL_PIXELFORMAT_IYUV, C.SDL_TEXTUREACCESS_STREAMING, C.int(Width), C.int(Height))

	render := C.CreatesRect(C.int(0), C.int(0), C.int(Width), C.int(Height))

	C.SDL_UpdateYUVTexture(
		ret,
		render,
		(*C.uchar)(data[0]),  //Yplane
		(C.int)(linesize[0]), //Ypitch
		(*C.uchar)(data[1]),  //Uplane
		(C.int)(linesize[1]), //Upitch
		(*C.uchar)(data[2]),  //Vplane
		(C.int)(linesize[2]), //Vpitch
	)

	C.FreesRect(render)

	return (*globaltype.SDL_Texture)(ret)
}
