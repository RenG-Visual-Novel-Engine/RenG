package ffmpeg

/*
#cgo CFLAGS: -Wall -I./include
#cgo LDFLAGS: -L./lib -lavcodec -lavformat -lavutil -lswscale

#include <libavcodec/avcodec.h>
#include <libavformat/avformat.h>
#include <libavutil/avutil.h>
#include <libswscale/swscale.h>

typedef struct _SWSContext {
	struct SwsContext* sws_context;
} SWSContext;
*/
import "C"

type Video struct {
	FormatCtx *C.AVFormatContext
	Frame     *C.AVFrame
	FrameYUV  *C.AVFrame

	ObjectDict *C.AVDictionary

	Packet *C.AVPacket

	VideoStream int
	AudioStream int

	VideoCodec    *C.AVCodec
	AudioCodec    *C.AVCodec
	VideoCodecCtx *C.AVCodecContext
	AudioCodecCtx *C.AVCodecContext

	SWSCxt C.SWSContext

	Buffer *C.uchar

	X int
	Y int
	W int
	H int
}
