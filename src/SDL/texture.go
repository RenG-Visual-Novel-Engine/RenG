package sdl

/*
#cgo CFLAGS: -I./../ffmpeg/include
#cgo CFLAGS: -I.
#cgo LDFLAGS: -L./lib -lSDL2 -lSDL2main -lSDL2_image
#cgo LDFLAGS: -L./../ffmpeg/lib -lavcodec -lavformat -lavutil -lswscale

#include <include/SDL.h>
#include <include/SDL_image.h>

#include <libavcodec/avcodec.h>
#include <libavformat/avformat.h>
#include <libavutil/avutil.h>
#include <libswscale/swscale.h>
*/
import "C"
import "RenG/src/ffmpeg"

func (t *SDL_Texture) SetBlendMode() {
	C.SDL_SetTextureBlendMode((*C.SDL_Texture)(t.Texture), SDL_BLENDMODE_BLEND)
}

func (t *SDL_Texture) SetAlpha(alpha uint8) {
	C.SDL_SetTextureAlphaMod((*C.SDL_Texture)(t.Texture), C.uchar(alpha))
}

func (t *SDL_Texture) UpdateYUVTexture(video *ffmpeg.Video) {
	rect := CreateRect(t.Xpos, t.Ypos, t.Width, t.Height)
	data1 := C.uchar(*ffmpeg.FindFrameData(video.FrameYUV, 0))
	linesize1 := C.int(*ffmpeg.FindFrameLinesize(video.FrameYUV, 0))
	data2 := C.uchar(*ffmpeg.FindFrameData(video.FrameYUV, 1))
	linesize2 := C.int(*ffmpeg.FindFrameLinesize(video.FrameYUV, 1))
	data3 := C.uchar(*ffmpeg.FindFrameData(video.FrameYUV, 2))
	linesize3 := C.int(*ffmpeg.FindFrameLinesize(video.FrameYUV, 2))

	C.SDL_UpdateYUVTexture(
		t.Texture,
		&rect,
		&data1,
		linesize1,
		&data2,
		linesize2,
		&data3,
		linesize3,
	)
}
