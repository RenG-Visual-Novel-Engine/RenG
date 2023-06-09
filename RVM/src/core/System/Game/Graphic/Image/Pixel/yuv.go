package pixel

/*
#cgo CFLAGS: -I./../../../../../sdl/include
#cgo CFLAGS: -I./../../../../../System/Game/Graphic/Image/c
#cgo LDFLAGS: -L./../../../../../sdl/lib -lSDL2 -lSDL2main -lSDL2_image






#include <convert.h>
*/
import "C"
import (
	"unsafe"
)

const (
	RENG_PIXELFORMAT_YUV420 = iota + RENG_PIXELFORMAT_RGBA8888 + 1
	RENG_PIXELFORMAT_YUV422
)

type YUV interface {
	SetYUV(Y, U, V *uint8)
	SetWH(Width, Height int64)

	GetYUV() (Y, U, V []uint8)
	GetWH() (Width, Height int64)

	ConvertRGBA(ConvertType int64) RGBA
}

type YUV420 struct {
	Width  int64
	Height int64

	Y []uint8
	U []uint8
	V []uint8
}

func (yuv *YUV420) SetYUV(Y, U, V *uint8) {
	yuv.Y = *(*[]uint8)(unsafe.Pointer(&Y))
	yuv.U = *(*[]uint8)(unsafe.Pointer(&U))
	yuv.V = *(*[]uint8)(unsafe.Pointer(&V))
}

func (yuv *YUV420) SetWH(Width, Height int64) {
	yuv.Width = Width
	yuv.Height = Height
}

func (yuv *YUV420) GetYUV() (Y, U, V []uint8) {
	return yuv.V, yuv.U, yuv.V
}

func (yuv *YUV420) GetWH() (Width, Height int64) {
	return yuv.Width, yuv.Height
}

func (yuv *YUV420) ConvertRGBA(ConvertType int64) RGBA {
	rgba := NewRGBA(RENG_PIXELFORMAT_RGBA8888)

	rgba.SetWH(yuv.Width, yuv.Height)

	pixels := C.YUV420ToRGBA8888(
		(*C.uchar)((*uint8)(unsafe.Pointer(&yuv.Y[0]))),
		(*C.uchar)((*uint8)(unsafe.Pointer(&yuv.U[0]))),
		(*C.uchar)((*uint8)(unsafe.Pointer(&yuv.V[0]))),
		C.int(yuv.Width),
		C.int(yuv.Height),
	)

	rgba.SetRGBA((*uint32)(pixels), yuv.Width)

	return rgba
}

func NewYUV(PixelFormatType int64) YUV {
	switch PixelFormatType {
	case RENG_PIXELFORMAT_YUV420:
		return &YUV420{}
	case RENG_PIXELFORMAT_YUV422:
		return nil
	default:
		return nil
	}
}
