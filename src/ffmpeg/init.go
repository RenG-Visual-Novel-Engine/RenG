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

void SwsCScale(struct SwsContext* sws_cxt, AVFrame* pFrame, AVFrame* pFrameYUV, int height)
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
	"unsafe"
)

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

func AVReadFrame(format *C.AVFormatContext, packet *C.AVPacket) int {
	return int(C.ReadFrame(format, packet))
}

func AVFreePacket(packet *C.AVPacket) {
	C.av_free_packet(packet)
}

func StreamIndex(packet *C.AVPacket) int {
	return int(packet.stream_index)
}

func AVCodecDecodeVideo(video *Video, frameFinished *C.int) {
	C.avcodec_decode_video2(video.VideoCodecCtx, video.Frame, frameFinished, video.Packet)
}

func SwsScale(video *Video) {
	C.SwsCScale(video.SWSCxt.sws_context, video.Frame, video.FrameYUV, C.int(video.H))
}

func FindFrameData(frameYUV *C.AVFrame, index int) *C.uint8_t {
	return frameYUV.data[index]
}

func FindFrameLinesize(frameYUV *C.AVFrame, index int) *C.int {
	return &frameYUV.linesize[index]
}
