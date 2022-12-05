#include <libavcodec/avcodec.h>
#include <libavformat/avformat.h>
#include <libswscale/swscale.h>

#include <SDL.h>
#include <SDL_thread.h>

typedef struct VideoState {
    AVFormatContext* ctx;
	AVCodecContext* codec_ctx;
	AVCodec* codec;
	struct SwsContext* sws_ctx;
	AVFrame* frame;

	int videoStream;

    int startTime;
	int delay;

    SDL_mutex* lock;

	SDL_Texture* texture;
} VideoState;

VideoState* VideoInit(char* path) {
    VideoState* v = (VideoState*)malloc(sizeof(VideoState));

    AVFormatContext* ctx = avformat_alloc_context();
	if (!ctx)
		return NULL;
	v->ctx = ctx;

    if (avformat_open_input(&ctx, path, NULL, NULL))
		return NULL;

	if (avformat_find_stream_info(ctx, NULL))
		return NULL;
	
	v->videoStream = 0;

	for (unsigned int i = 0; i < ctx->nb_streams; i++)
	{
		if (ctx->streams[i]->codecpar->codec_type == AVMEDIA_TYPE_VIDEO)
			v->videoStream = i;
	}

	if (v->videoStream == -1)
		return NULL;

	AVCodecContext* codec_ctx;
	AVCodec* codec;

	codec_ctx = avcodec_alloc_context3(NULL);
	if (!codec_ctx)
		return NULL;

	if (avcodec_parameters_to_context(codec_ctx, ctx->streams[v->videoStream]->codecpar) < 0)
		return NULL;

	codec_ctx->pkt_timebase = ctx->streams[v->videoStream]->time_base;

	codec = avcodec_find_decoder(codec_ctx->codec_id);
	if (!codec)
		return NULL;

	codec_ctx->codec_id = codec->id;

	if (avcodec_open2(codec_ctx, codec, NULL))
		return NULL;

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

	v->lock = SDL_CreateMutex();

    return v;
}

void DecodeFrame(VideoState* v, int index)
{
	if (index < v->codec_ctx->frame_number)
		return;

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
			break;
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
}

void Lock(VideoState* v) {
	SDL_LockMutex(v->lock);
}

void Unlock(VideoState* v) {
	SDL_UnlockMutex(v->lock);
}

int video_thread(void* data) {
	VideoState* v = (VideoState*)data;
	SDL_Rect render = { 0, 0, 1920, 1080 }; // TODO

	for (;;) {
		Lock(v);

		DecodeFrame(v, (clock() - v->startTime) / (1000 / 30));
	
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

		Unlock(v);

		SDL_Delay(v->delay);

	}

	return 0;
}

void Start(VideoState* v, SDL_Renderer* r) {
	v->texture = SDL_CreateTexture(
		r,
		SDL_PIXELFORMAT_YV12, 
		SDL_TEXTUREACCESS_STREAMING,
		v->codec_ctx->width,
		v->codec_ctx->height
	);

	v->delay = 1000 / 30; //TODO
	v->startTime = clock();

	SDL_CreateThread(video_thread, "video_thread", v);
}