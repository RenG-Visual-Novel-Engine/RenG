#include <libavcodec/avcodec.h>
#include <libavformat/avformat.h>
#include <libswscale/swscale.h>

#include <SDL.h>
#include <SDL_thread.h>
#include <windows.h> 

#include <stdio.h>

static SDL_mutex* lock;

typedef struct VideoState {
    AVFormatContext* ctx;
	AVCodecContext* codec_ctx;
	AVCodec* codec;
	struct SwsContext* sws_ctx;
	AVFrame* frame;

	int videoStream;

    unsigned long startTime;
	int delay;

	char* path;

// bool
	int nowPlaying;
	int stop;
	int loop;

	SDL_Texture* texture;
} VideoState;

int initCodec(VideoState* v) 
{
	AVFormatContext* ctx = avformat_alloc_context();
	if (!ctx)
		return 0;
	v->ctx = ctx;

    if (avformat_open_input(&ctx, v->path, NULL, NULL))
		return 0;

	if (avformat_find_stream_info(ctx, NULL))
		return 0;
	
	v->videoStream = -1;

	for (unsigned int i = 0; i < ctx->nb_streams; i++)
	{
		if (ctx->streams[i]->codecpar->codec_type == AVMEDIA_TYPE_VIDEO)
			v->videoStream = i;
	}

	if (v->videoStream == -1)
		return 0;

	AVCodecContext* codec_ctx;

	codec_ctx = avcodec_alloc_context3(NULL);
	if (!codec_ctx)
		return 0;

	if (avcodec_parameters_to_context(codec_ctx, ctx->streams[v->videoStream]->codecpar) < 0)
		return 0;

	codec_ctx->pkt_timebase = ctx->streams[v->videoStream]->time_base;

	AVCodec* codec = (AVCodec*)avcodec_find_decoder(codec_ctx->codec_id);
	if (!codec)
		return 0; 

	codec_ctx->codec_id = codec->id;

	if (avcodec_open2(codec_ctx, codec, NULL))
		return 0;

	v->codec_ctx = codec_ctx;
	v->codec = codec;

	v->sws_ctx = sws_getContext(
		v->codec_ctx->width,
		v->codec_ctx->height,
		v->codec_ctx->pix_fmt,
		v->codec_ctx->width,
		v->codec_ctx->height,
		AV_PIX_FMT_YUV420P,
		SWS_BILINEAR,
		NULL,
		NULL,
		NULL
	);

	return 1;
}

void destroyCodec(VideoState* v)
{
	avcodec_free_context(&v->codec_ctx);
	avformat_free_context(v->ctx);
	sws_freeContext(v->sws_ctx);
	
}

int DecodeFrame(VideoState* v, int index)
{
	if (index < v->codec_ctx->frame_number)
		return 1;

	AVFrame* frame = av_frame_alloc();
		
	for (;;)
	{
		int ret;

		AVPacket* pkt = av_packet_alloc();
		av_read_frame(v->ctx, pkt);

		if (!pkt->stream_index == v->videoStream)
		{
			av_packet_free(&pkt);
			continue;
		}

		ret = avcodec_send_packet(v->codec_ctx, pkt);

		if (ret == AVERROR_EOF)
		{
			av_packet_unref(pkt);

			return 0;
		}

		ret = avcodec_receive_frame(v->codec_ctx, frame);

		if (ret == AVERROR(EAGAIN))
			continue;

		if (index > v->codec_ctx->frame_number)
		{
			av_packet_unref(pkt);
			continue;
		}

		av_packet_unref(pkt);

		break;
	}
	// AVFrame* del = v->frame;
	v->frame = frame;

	return 1;
}

VideoState* VideoInit(char* path, SDL_Renderer* r) {
    VideoState* v = (VideoState*)malloc(sizeof(VideoState));

	v->path = path;

	if (!initCodec(v))
		return NULL;

	if (!lock)
		lock = SDL_CreateMutex();

	v->texture = SDL_CreateTexture(
		r,
		SDL_PIXELFORMAT_IYUV, 
		SDL_TEXTUREACCESS_STREAMING,
		v->codec_ctx->width,
		v->codec_ctx->height
	);

	v->nowPlaying = 0;

	DecodeFrame(v, 0);

	SDL_Rect render = { 0, 0, 1280, 720 };

	SDL_UpdateYUVTexture(
		v->texture,
		&render,
		v->frame->data[0],
		v->frame->linesize[0],
		v->frame->data[1],
		v->frame->linesize[1],
		v->frame->data[2],
		v->frame->linesize[2]
	);

	SDL_SetTextureBlendMode(v->texture, SDL_BLENDMODE_BLEND);

    return v;
}

void Lock() {
	SDL_LockMutex(lock);
}

void Unlock() {
	SDL_UnlockMutex(lock);
}

int video_thread(void* data) {
	VideoState* v = (VideoState*)data;
	SDL_Rect render = { 0, 0, 1280, 720 }; // TODO

	 // SDL_Delay(400);

	v->startTime = timeGetTime();

	for (;;) {
		Lock();

		if (v->stop) 
		{
			v->nowPlaying = 0;
			Unlock();
			break;
		}

		if (!DecodeFrame(v, (int)((timeGetTime() - v->startTime) / (1000.0 / 60.0))))
		{

			if (v->loop)
			{
				destroyCodec(v);
				initCodec(v);
				v->startTime = timeGetTime();
				Unlock();
				continue;
			}
			v->nowPlaying = 0;
			Unlock();
			break;
		}

		if (v->frame) {
			
			SDL_UpdateYUVTexture(
				v->texture,
				&render,
				v->frame->data[0],
				v->frame->linesize[0],
				v->frame->data[1],
				v->frame->linesize[1],
				v->frame->data[2],
				v->frame->linesize[2]
			);
		
			av_frame_free(&v->frame);

		}

		Unlock();

		SDL_Delay(v->delay);

	}

	return 0;
}

void Start(VideoState* v, int Loop) {
	v->delay = 10; //TODO
	v->nowPlaying = 1;
	v->loop = Loop;
	v->stop = 0;

	SDL_CreateThread(video_thread, "video_thread", v);
}