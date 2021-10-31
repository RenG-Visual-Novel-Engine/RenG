package core

/*
#cgo CFLAGS: -I./ffmpeg/include
#cgo LDFLAGS: -L./ffmpeg/lib -lavcodec -lavformat -lavutil -lswscale
#cgo CFLAGS: -I./sdl/include
#cgo LDFLAGS: -L./sdl/lib -lSDL2 -lSDL2main -lSDL2_image

#include <libavcodec/avcodec.h>
#include <libavformat/avformat.h>
#include <libavutil/avutil.h>
#include <libswscale/swscale.h>

#include <SDL.h>
#include <SDL_image.h>

typedef struct _SWSContext {
	struct SwsContext* sws_context;
} SWSContext;

int FindVideoStream(AVFormatContext* pFormatCtx)
{
	for (int i = 0; i < pFormatCtx->nb_streams; i++)
	{
		if (pFormatCtx->streams[i]->codec->codec_type == AVMEDIA_TYPE_VIDEO)
		{
			return i;
		}
	}
	return -1;
}

int FindAudioStream(AVFormatContext* pFormatCtx)
{
	for (int i = 0; i < pFormatCtx->nb_streams; i++)
	{
		if (pFormatCtx->streams[i]->codec->codec_type == AVMEDIA_TYPE_AUDIO)
		{
			return i;
		}
	}
	return -1;
}

AVCodecContext* FindCodecContext(AVFormatContext* pFormatCtx, int stream)
{
	return pFormatCtx->streams[stream]->codec;
}

void SWSContextFill(SWSContext* ctx, AVCodecContext* codec, int width, int height)
{
	ctx->sws_context = sws_getContext(
		width,
		height,
		codec->pix_fmt,
		width,
		height,
		AV_PIX_FMT_YUV420P,
		SWS_BILINEAR,
		NULL,
		NULL,
		NULL);
}

uint8_t* bufferMalloc(int numBytes)
{
	return (uint8_t*)av_malloc(numBytes * sizeof(uint8_t));
}

void AVPictureFill(AVFrame* pFrameYUV, uint8_t* buffer, int width, int height)
{
	avpicture_fill((AVPicture*)pFrameYUV, buffer, AV_PIX_FMT_YUV420P, width, height);
}

void SwsScale(struct SwsContext* sws_cxt, AVFrame* pFrame, AVFrame* pFrameYUV, int height)
{
	sws_scale(sws_cxt, (uint8_t const* const*)pFrame->data, pFrame->linesize, 0, height, pFrameYUV->data, pFrameYUV->linesize);
}

int ReadFrame(AVFormatContext* pFormatCtx, AVPacket* packet)
{
	return av_read_frame(pFormatCtx, packet);
}
*/
import "C"
import (
	"sync"
	"unsafe"
)

type Video struct {
	FormatCtx *C.AVFormatContext
	Frame     *C.AVFrame
	FrameYUV  *C.AVFrame

	ObjectDict *C.AVDictionary

	Packet C.AVPacket

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

func OpenVideo(path string, width, height int) *Video {
	var video Video

	video.W = width
	video.H = height

	root := C.CString(path)
	defer C.free(unsafe.Pointer(root))

	if int(C.avformat_open_input(&video.FormatCtx, root, nil, nil)) != 0 {
		return nil
	}

	if int(C.avformat_find_stream_info(video.FormatCtx, nil)) != 0 {
		return nil
	}

	video.VideoStream = int(C.FindVideoStream(video.FormatCtx))
	video.AudioStream = int(C.FindAudioStream(video.FormatCtx))

	video.VideoCodecCtx = C.FindCodecContext(video.FormatCtx, C.int(video.VideoStream))
	video.AudioCodecCtx = C.FindCodecContext(video.FormatCtx, C.int(video.AudioStream))

	video.VideoCodec = C.avcodec_find_decoder(video.VideoCodecCtx.codec_id)

	if int(C.avcodec_open2(video.VideoCodecCtx, video.VideoCodec, &video.ObjectDict)) != 0 {
		return nil
	}

	video.Frame = C.av_frame_alloc()
	video.FrameYUV = C.av_frame_alloc()

	C.SWSContextFill(&video.SWSCxt, video.VideoCodecCtx, C.int(video.W), C.int(video.H))

	video.Buffer = C.bufferMalloc(C.avpicture_get_size(C.AV_PIX_FMT_YUV420P, C.int(video.W), C.int(video.H)))

	C.AVPictureFill(video.FrameYUV, video.Buffer, C.int(video.W), C.int(video.H))

	return &video
}

func PlayVideo(video *Video, texture *SDL_Texture, layerMutex *sync.RWMutex, layerList LayerList, renderer *SDL_Renderer) {
	frameFinished := C.int(0)
	rect := CreateRect(0, 0, 1280, 720)

	for int(C.av_read_frame(video.FormatCtx, &video.Packet)) >= 0 {

		if int(video.Packet.stream_index) == video.VideoStream {
			C.avcodec_decode_video2(video.VideoCodecCtx, video.Frame, &frameFinished, &video.Packet)

			if int(frameFinished) == 1 {
				C.SwsScale(video.SWSCxt.sws_context, video.Frame, video.FrameYUV, C.int(video.H))

				C.SDL_UpdateYUVTexture(
					texture.Texture,
					&rect,
					video.FrameYUV.data[0],
					video.FrameYUV.linesize[0],
					video.FrameYUV.data[1],
					video.FrameYUV.linesize[1],
					video.FrameYUV.data[2],
					video.FrameYUV.linesize[2],
				)
				renderer.RenderClear()
				C.SDL_RenderCopy((*C.SDL_Renderer)(renderer), texture.Texture, nil, nil)
				renderer.RenderPresent()
			}
		}

		C.av_free_packet(&video.Packet)
	}
}

func FindFrameData(frameYUV *C.AVFrame, index int) *C.uint8_t {
	return frameYUV.data[index]
}

func FindFrameLinesize(frameYUV *C.AVFrame, index int) *C.int {
	return &frameYUV.linesize[index]
}
