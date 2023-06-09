package image

/*
#cgo CFLAGS: -I./../../../../sdl/include
#cgo CFLAGS: -I./../../../../System/Game/Graphic/Image/c
#cgo LDFLAGS: -L./../../../../sdl/lib -lSDL2 -lSDL2main -lSDL2_image

#include <SDL.h>
*/
import "C"
import (
	pixel "RenG/RVM/src/core/System/Game/Graphic/Image/Pixel"
	"RenG/RVM/src/core/globaltype"
)

func (i *Image) ConvertFrameDataToYUV(data [8]*uint8, linesize [8]int32, Width, Height int64) pixel.YUV {
	yuv := pixel.NewYUV(pixel.RENG_PIXELFORMAT_YUV420)

	yuv.SetYUV(
		data[0],
		data[1],
		data[2],
	)

	yuv.SetWH(
		Width,
		Height,
	)

	return yuv
}

func (i *Image) ConvertSurfaceToTexture(sur *C.SDL_Surface) *globaltype.SDL_Texture {
	return (*globaltype.SDL_Texture)(C.SDL_CreateTextureFromSurface((*C.SDL_Renderer)(i.renderer), sur))
}
