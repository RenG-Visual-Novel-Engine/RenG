#include "ffvideo.h"

VideoState* VideoInit(char* filename) {
    VideoState* vs = av_calloc(1, sizeof(VideoState));

    vs->cond = SDL_CreateCond();
    if (vs->cond == NULL)
        return NULL;

    vs->lock = SDL_CreateMutex();
    if (vs->lock == NULL) 
        return NULL;

    vs->filename =filename;

    return vs;
}

void FreeVideoState(VideoState* vs)
{
    while (1)
    {
        SurfaceQueueEntry* sqe = dequeue_surface(&vs->surface_queue);

        if (!sqe)
            break;
        
        if (sqe->pixels)
            SDL_free(sqe->pixels);

        av_free(vs);
    }

    if (vs->sws) {
		sws_freeContext(vs->sws);
	}

	if (vs->video_decode_frame) {
		av_frame_free(&vs->video_decode_frame);
	}

	/* Destroy audio stuff. */
	if (vs->swr) {
		swr_free(&vs->swr);
	}

	if (vs->audio_decode_frame) {
		av_frame_free(&vs->audio_decode_frame);
	}

	if (vs->audio_out_frame) {
		av_frame_free(&vs->audio_out_frame);
	}

	while (1) {
		AVFrame *f = dequeue_frame(&vs->audio_queue);

		if (!f) {
			break;
		}

		av_frame_free(&f);
	}

	/* Destroy/Close core stuff. */
	free_packet_queue(&vs->audio_packet_queue);
	free_packet_queue(&vs->video_packet_queue);

	if (vs->video_context) {
		avcodec_free_context(&vs->video_context);
	}
	if (vs->audio_context) {
		avcodec_free_context(&vs->audio_context);
	}

	if (vs->ctx) {

		if (vs->ctx->pb) {
			if (vs->ctx->pb->buffer) {
				av_freep(&vs->ctx->pb->buffer);
			}
			av_freep(&vs->ctx->pb);
		}

		avformat_close_input(&vs->ctx);
		avformat_free_context(vs->ctx);
	}

	/* Destroy alloc stuff. */
	if (vs->cond) {
		SDL_DestroyCond(vs->cond);
	}
	if (vs->lock) {
		SDL_DestroyMutex(vs->lock);
	}

	if (vs->rwops) {
		rwops_close(vs->rwops);
	}

	if (vs->filename) {
		av_free(vs->filename);
	}

	/* Add this MediaState to a queue to have its thread ended, and the MediaState
	 * deactivated.
	 */
	SDL_LockMutex(deallocate_mutex);
    vs->next = deallocate_queue;
    deallocate_queue = vs;
    SDL_UnlockMutex(deallocate_mutex);
}

// http://dranger.com/ffmpeg/

/*******************************************************************************
 * SDL_RWops <-> AVIOContext
 */

int rwops_read(void *opaque, uint8_t *buf, int buf_size) {
    SDL_RWops *rw = (SDL_RWops *) opaque;

    int rv = rw->read(rw, buf, 1, buf_size);
    return rv;

}

int rwops_write(void *opaque, uint8_t *buf, int buf_size) {
    printf("Writing to an SDL_rwops is a really bad idea.\n");
    return -1;
}

int64_t rwops_seek(void *opaque, int64_t offset, int whence) {
    SDL_RWops *rw = (SDL_RWops *) opaque;

    if (whence == AVSEEK_SIZE) {
    	return rw->size(rw);
    }

    // Ignore flags like AVSEEK_FORCE.
    whence &= (SEEK_SET | SEEK_CUR | SEEK_END);

    int64_t rv = rw->seek(rw, (int) offset, whence);
    return rv;
}

AVIOContext *rwops_open(SDL_RWops *rw) {

    unsigned char *buffer = av_malloc(RWOPS_BUFFER);
	if (buffer == NULL) {
		return NULL;
	}
    AVIOContext *rv = avio_alloc_context(
        buffer,
        RWOPS_BUFFER,
        0,
        rw,
        rwops_read,
        rwops_write,
        rwops_seek);
    if (rv == NULL) {
    	av_free(buffer);
    	return NULL;
    }

    return rv;
}

void rwops_close(SDL_RWops *rw) {
	rw->close(rw);
}

static void deallocate(MediaState *ms) {

    while (1) {
		SurfaceQueueEntry *sqe = dequeue_surface(&ms->surface_queue);

		if (! sqe) {
			break;
		}

		if (sqe->pixels) {
#ifndef USE_POSIX_MEMALIGN
			SDL_free(sqe->pixels);
#else
			free(sqe->pixels);
#endif
		}
		av_free(sqe);
	}

	if (ms->sws) {
		sws_freeContext(ms->sws);
	}

	if (ms->video_decode_frame) {
		av_frame_free(&ms->video_decode_frame);
	}

	/* Destroy audio stuff. */
	if (ms->swr) {
		swr_free(&ms->swr);
	}

	if (ms->audio_decode_frame) {
		av_frame_free(&ms->audio_decode_frame);
	}

	if (ms->audio_out_frame) {
		av_frame_free(&ms->audio_out_frame);
	}

	while (1) {
		AVFrame *f = dequeue_frame(&ms->audio_queue);

		if (!f) {
			break;
		}

		av_frame_free(&f);
	}

	/* Destroy/Close core stuff. */
	free_packet_queue(&ms->audio_packet_queue);
	free_packet_queue(&ms->video_packet_queue);

	if (ms->video_context) {
		avcodec_free_context(&ms->video_context);
	}
	if (ms->audio_context) {
		avcodec_free_context(&ms->audio_context);
	}

	if (ms->ctx) {

		if (ms->ctx->pb) {
			if (ms->ctx->pb->buffer) {
				av_freep(&ms->ctx->pb->buffer);
			}
			av_freep(&ms->ctx->pb);
		}

		avformat_close_input(&ms->ctx);
		avformat_free_context(ms->ctx);
	}

	/* Destroy alloc stuff. */
	if (ms->cond) {
		SDL_DestroyCond(ms->cond);
	}
	if (ms->lock) {
		SDL_DestroyMutex(ms->lock);
	}

	if (ms->rwops) {
		rwops_close(ms->rwops);
	}

	if (ms->filename) {
		av_free(ms->filename);
	}

	/* Add this MediaState to a queue to have its thread ended, and the MediaState
	 * deactivated.
	 */
	SDL_LockMutex(deallocate_mutex);
    ms->next = deallocate_queue;
    deallocate_queue = ms;
    SDL_UnlockMutex(deallocate_mutex);

}

/* Perform the portion of deallocation that's been deferred to the main thread. */
static void deallocate_deferred() {

    SDL_LockMutex(deallocate_mutex);

    while (deallocate_queue) {
        MediaState *ms = deallocate_queue;
        deallocate_queue = ms->next;

        if (ms->thread) {
            SDL_WaitThread(ms->thread, NULL);
        }

        av_free(ms);
    }

    SDL_UnlockMutex(deallocate_mutex);
}

/* Frame queue ***************************************************************/

static void enqueue_frame(FrameQueue *fq, AVFrame *frame) {
	frame->opaque = NULL;

	if (fq->first) {
		fq->last->opaque = frame;
		fq->last = frame;
	} else {
		fq->first = fq->last = frame;
	}
}

static AVFrame *dequeue_frame(FrameQueue *fq) {
	if (!fq->first) {
		return NULL;
	}

	AVFrame *rv = fq->first;
	fq->first = (AVFrame *) rv->opaque;

	if (!fq->first) {
		fq->last = NULL;
	}

	return rv;
}


/* Packet queue **************************************************************/

static void enqueue_packet(PacketQueue *pq, AVPacket *pkt) {
	PacketQueueEntry *pqe = av_malloc(sizeof(PacketQueueEntry));
	if (pqe == NULL) {
		av_packet_free(&pkt);
		return;
	}

	pqe->pkt = pkt;
	pqe->next = NULL;

	if (!pq->first) {
		pq->first = pq->last = pqe;
	} else {
		pq->last->next = pqe;
		pq->last = pqe;
	}
}

static AVPacket *first_packet(PacketQueue *pq) {
	if (pq->first) {
		return pq->first->pkt;
	} else {
		return NULL;
	}
}

static void dequeue_packet(PacketQueue *pq) {
	if (! pq->first) {
		return;
	}

	PacketQueueEntry *pqe = pq->first;
	pq->first = pqe->next;

	if (!pq->first) {
		pq->last = NULL;
	}

	av_packet_free(&pqe->pkt);
	av_free(pqe);
}

static int count_packet_queue(PacketQueue *pq) {
    PacketQueueEntry *pqe = pq->first;

	int rv = 0;

	while (pqe) {
		rv += 1;
		pqe = pqe->next;
	}

	return rv;
}

static void free_packet_queue(PacketQueue *pq) {
	while(first_packet(pq)) {
		dequeue_packet(pq);
	}
}


/**
 * Reads a packet from one of the queues, filling the other queue if
 * necessary. Returns the packet, or NULL if end of file has been reached.
 */
static AVPacket *read_packet(MediaState *ms, PacketQueue *pq) {

	AVPacket *pkt;
	AVPacket *rv;

	while (1) {

		rv = first_packet(pq);
		if (rv) {
			return rv;
		}

		pkt = av_packet_alloc();

		if (!pkt) {
			return NULL;
		}

		if (av_read_frame(ms->ctx, pkt)) {
			return NULL;
		}

		if (pkt->stream_index == ms->video_stream && ! ms->video_finished) {
			enqueue_packet(&ms->video_packet_queue, pkt);
		} else if (pkt->stream_index == ms->audio_stream && ! ms->audio_finished) {
			enqueue_packet(&ms->audio_packet_queue, pkt);
		} else {
			av_packet_free(&pkt);
		}
	}
}


/* Surface queue *************************************************************/

static void enqueue_surface(SurfaceQueueEntry **queue, SurfaceQueueEntry *sqe) {
	while (*queue) {
		queue = &(*queue)->next;
	}

	*queue = sqe;
}


static SurfaceQueueEntry *dequeue_surface(SurfaceQueueEntry **queue) {
	SurfaceQueueEntry *rv = *queue;

	if (rv) {
		*queue = rv->next;
	}

	return rv;
}


#if 0
static void check_surface_queue(MediaState *ms) {

	SurfaceQueueEntry **queue = &ms->surface_queue;

	int count = 0;

	while (*queue) {
		count += 1;
		queue = &(*queue)->next;
	}

	if (count != ms->surface_queue_size) {
		abort();
	}

}
#endif

/* Find decoder context ******************************************************/


static AVCodecContext *find_context(AVFormatContext *ctx, int index) {

    AVDictionary *opts = NULL;

	if (index == -1) {
		return NULL;
	}

	AVCodec *codec = NULL;
	AVCodecContext *codec_ctx = NULL;

	codec_ctx = avcodec_alloc_context3(NULL);

	if (codec_ctx == NULL) {
		return NULL;
	}

	if (avcodec_parameters_to_context(codec_ctx, ctx->streams[index]->codecpar) < 0) {
		goto fail;
	}

	codec_ctx->pkt_timebase = ctx->streams[index]->time_base;

    codec = avcodec_find_decoder(codec_ctx->codec_id);

    if (codec == NULL) {
        goto fail;
    }

    codec_ctx->codec_id = codec->id;

    av_dict_set(&opts, "threads", "auto", 0);
    av_dict_set(&opts, "refcounted_frames", "0", 0);

	if (avcodec_open2(codec_ctx, codec, &opts)) {
		goto fail;
	}

	return codec_ctx;

fail:

    av_dict_free(&opts);

	avcodec_free_context(&codec_ctx);
	return NULL;
}


/* Audio decoding *************************************************************/

static void decode_audio(MediaState *ms) {
	int ret;
	AVPacket *pkt;
	AVFrame *converted_frame;

	if (!ms->audio_context) {
		ms->audio_finished = 1;
		return;
	}

	if (ms->audio_decode_frame == NULL) {
		ms->audio_decode_frame = av_frame_alloc();
	}

	if (ms->audio_decode_frame == NULL) {
		ms->audio_finished = 1;
		return;
	}

	double timebase = av_q2d(ms->ctx->streams[ms->audio_stream]->time_base);

	if (ms->audio_queue_target_samples < audio_target_samples) {
	    ms->audio_queue_target_samples += audio_sample_increase;
	}

	while (ms->audio_queue_samples < ms->audio_queue_target_samples) {

		/** Read a packet, and send it to the decoder. */
		pkt = read_packet(ms, &ms->audio_packet_queue);
		ret = avcodec_send_packet(ms->audio_context, pkt);

		if (ret == 0) {
			dequeue_packet(&ms->audio_packet_queue);
		} else if (ret == AVERROR(EAGAIN) || ret == AVERROR_EOF) {
			// pass
		} else {
			ms->audio_finished = 1;
			return;
		}

		while (1) {

			ret = avcodec_receive_frame(ms->audio_context, ms->audio_decode_frame);

			// More input is needed.
			if (ret == AVERROR(EAGAIN)) {
				break;
			}

			if (ret < 0) {
				ms->audio_finished = 1;
				return;
			}

            converted_frame = av_frame_alloc();

			if (converted_frame == NULL) {
				ms->audio_finished = 1;
				return;
			}

            converted_frame->sample_rate = audio_sample_rate;
            converted_frame->channel_layout = AV_CH_LAYOUT_STEREO;
            converted_frame->format = AV_SAMPLE_FMT_S16;

			if (!ms->audio_decode_frame->channel_layout) {
				ms->audio_decode_frame->channel_layout = av_get_default_channel_layout(ms->audio_decode_frame->channels);

				if (audio_equal_mono && (ms->audio_decode_frame->channels == 1)) {
				    swr_alloc_set_opts(
                        ms->swr,
                        converted_frame->channel_layout,
                        converted_frame->format,
                        converted_frame->sample_rate,
                        ms->audio_decode_frame->channel_layout,
                        ms->audio_decode_frame->format,
                        ms->audio_decode_frame->sample_rate,
                        0,
                        NULL);

				    swr_set_matrix(ms->swr, stereo_matrix, 1);
				}
			}

			if(swr_convert_frame(ms->swr, converted_frame, ms->audio_decode_frame)) {
				av_frame_free(&converted_frame);
				continue;
			}

			double start = ms->audio_decode_frame->best_effort_timestamp * timebase;
			double end = start + 1.0 * converted_frame->nb_samples / audio_sample_rate;

			SDL_LockMutex(ms->lock);

			if (start >= ms->skip) {

				// Normal case, queue the frame.
				ms->audio_queue_samples += converted_frame->nb_samples;
				enqueue_frame(&ms->audio_queue, converted_frame);

			} else if (end < ms->skip) {
				// Totally before, drop the frame.
				av_frame_free(&converted_frame);

			} else {
				// The frame straddles skip, so we queue the (necessarily single)
				// frame and set the index into the frame.
				ms->audio_out_frame = converted_frame;
				ms->audio_out_index = BPS * (int) ((ms->skip - start) * audio_sample_rate);

			}

			SDL_UnlockMutex(ms->lock);
		}

	}

	return;

}


/* Video decoding *************************************************************/

static enum AVPixelFormat get_pixel_format(SDL_Surface *surf) {
    uint32_t pixel;
    uint8_t *bytes = (uint8_t *) &pixel;

	pixel = SDL_MapRGBA(surf->format, 1, 2, 3, 4);

	enum AVPixelFormat fmt;

    if ((bytes[0] == 4 || bytes[0] == 0) && bytes[1] == 1) {
        fmt = AV_PIX_FMT_ARGB;
    } else if ((bytes[0] == 4  || bytes[0] == 0) && bytes[1] == 3) {
        fmt = AV_PIX_FMT_ABGR;
    } else if (bytes[0] == 1) {
        fmt = AV_PIX_FMT_RGBA;
    } else {
        fmt = AV_PIX_FMT_BGRA;
    }

    return fmt;
}


static SurfaceQueueEntry *decode_video_frame(MediaState *ms) {
	int ret;

	while (1) {

		AVPacket *pkt = read_packet(ms, &ms->video_packet_queue);
		ret = avcodec_send_packet(ms->video_context, pkt);


		if (ret == 0) {
			dequeue_packet(&ms->video_packet_queue);
		} else if (ret == AVERROR(EAGAIN) || ret == AVERROR_EOF) {
			// pass
		} else {
			ms->video_finished = 1;
			return NULL;
		}

		ret = avcodec_receive_frame(ms->video_context, ms->video_decode_frame);

		// More input is needed.
		if (ret == AVERROR(EAGAIN)) {
			continue;
		}

		if (ret < 0) {
			ms->video_finished = 1;
			return NULL;
		}

		break;
	}

	double pts = ms->video_decode_frame->best_effort_timestamp * av_q2d(ms->ctx->streams[ms->video_stream]->time_base);

	if (pts < ms->skip) {
		return NULL;
	}

	// If we're behind on decoding the frame, drop it.
	if (ms->video_pts_offset && (ms->video_pts_offset + pts < ms->video_read_time)) {

		// If we're 5s behind, give up on video for the time being, so we don't
		// blow out memory.
		if (ms->video_pts_offset + pts < ms->video_read_time - 5.0) {
			ms->video_finished = 1;
		}

		if (ms->frame_drops) {
		    return NULL;
		}
	}

	SDL_Surface *sample = rgba_surface;

	ms->sws = sws_getCachedContext(
		ms->sws,

		ms->video_decode_frame->width,
		ms->video_decode_frame->height,
		ms->video_decode_frame->format,

		ms->video_decode_frame->width,
		ms->video_decode_frame->height,
		get_pixel_format(rgba_surface),

		SWS_POINT,

		NULL,
		NULL,
		NULL
		);

	if (!ms->sws) {
		ms->video_finished = 1;
		return NULL;
	}

	SurfaceQueueEntry *rv = av_malloc(sizeof(SurfaceQueueEntry));
	if (rv == NULL) {
		ms->video_finished = 1;
		return NULL;
	}
	rv->w = ms->video_decode_frame->width + FRAME_PADDING * 2;
	rv->h = ms->video_decode_frame->height + FRAME_PADDING * 2;

	rv->pitch = rv->w * sample->format->BytesPerPixel;

	if (rv->pitch % ROW_ALIGNMENT) {
	    rv->pitch += ROW_ALIGNMENT - (rv->pitch % ROW_ALIGNMENT);
	}

#ifndef USE_POSIX_MEMALIGN
    rv->pixels = SDL_calloc(rv->pitch * rv->h, 1);
#else
    posix_memalign(&rv->pixels, ROW_ALIGNMENT, rv->pitch * rv->h);
    memset(rv->pixels, 0, rv->pitch * rv->h);
#endif

	rv->format = sample->format;
	rv->next = NULL;
	rv->pts = pts;

	uint8_t *surf_pixels = (uint8_t *) rv->pixels;
	uint8_t *surf_data[] = { &surf_pixels[FRAME_PADDING * rv->pitch + FRAME_PADDING * sample->format->BytesPerPixel] };
	int surf_linesize[] = { rv->pitch };

	sws_scale(
		ms->sws,

		(const uint8_t * const *) ms->video_decode_frame->data,
		ms->video_decode_frame->linesize,

		0,
		ms->video_decode_frame->height,

		surf_data,
		surf_linesize
		);

	return rv;
}


static void decode_video(MediaState *ms) {
	if (!ms->video_context) {
		ms->video_finished = 1;
		return;
	}

	if (!ms->video_decode_frame) {
		ms->video_decode_frame = av_frame_alloc();
	}

	if (!ms->video_decode_frame) {
		ms->video_finished = 1;
		return;
	}

	SDL_LockMutex(ms->lock);

	if (!ms->video_finished && (ms->surface_queue_size < FRAMES)) {

		SDL_UnlockMutex(ms->lock);

		SurfaceQueueEntry *sqe = decode_video_frame(ms);

		SDL_LockMutex(ms->lock);

		if (sqe) {
			enqueue_surface(&ms->surface_queue, sqe);
			ms->surface_queue_size += 1;
		}
	}

	if (!ms->video_finished && (ms->surface_queue_size < FRAMES)) {
		ms->needs_decode = 1;
	}

	SDL_UnlockMutex(ms->lock);
}


static int decode_sync_start(void *arg);
void media_read_sync(struct MediaState *ms);
void media_read_sync_finish(struct MediaState *ms);


/**
 * Returns 1 if there is a video frame ready on this channel, or 0 otherwise.
 */
int media_video_ready(struct MediaState *ms) {

	int consumed = 0;
	int rv = 0;

	if (ms->video_stream == -1) {
		return 1;
	}

	SDL_LockMutex(ms->lock);

	if (!ms->ready) {
		goto done;
	}

	if (ms->pause_time > 0) {
	    goto done;
	}

	double offset_time = current_time - ms->time_offset;

	/*
	 * If we have an obsolete frame, drop it.
	 */
	if (ms->video_pts_offset) {
		while (ms->surface_queue) {

			/* The PTS is greater that the last frame read, so we're good. */
			if (ms->surface_queue->pts + ms->video_pts_offset >= ms->video_read_time) {
				break;
			}

			/* Otherwise, drop it without display. */
			SurfaceQueueEntry *sqe = dequeue_surface(&ms->surface_queue);
			ms->surface_queue_size -= 1;

			if (sqe->pixels) {
#ifndef USE_POSIX_MEMALIGN
				SDL_free(sqe->pixels);
#else
				free(sqe->pixels);
#endif
			}
			av_free(sqe);

			consumed = 1;
		}
	}


	/*
	 * Otherwise, check to see if we have a frame with a PTS that has passed.
	 */

	if (ms->surface_queue) {
		if (ms->video_pts_offset) {
			if (ms->surface_queue->pts + ms->video_pts_offset <= offset_time + frame_early_delivery) {
				rv = 1;
			}
		} else {
			rv = 1;
		}
	}

done:

	/* Only signal if we've consumed something. */
	if (consumed) {
		ms->needs_decode = 1;
		SDL_CondBroadcast(ms->cond);
	}

	SDL_UnlockMutex(ms->lock);

	return rv;
}


SDL_Surface *media_read_video(MediaState *ms) {

	SDL_Surface *rv = NULL;
	SurfaceQueueEntry *sqe = NULL;

	if (ms->video_stream == -1) {
		return NULL;
	}

	double offset_time = current_time - ms->time_offset;

	SDL_LockMutex(ms->lock);

#ifndef __EMSCRIPTEN__
	while (!ms->ready) {
	    SDL_CondWait(ms->cond, ms->lock);
	}
#endif

	if (ms->pause_time > 0) {
	    goto done;
	}

	if (!ms->surface_queue_size) {
		goto done;
	}

	if (ms->video_pts_offset == 0.0) {
		ms->video_pts_offset = offset_time - ms->surface_queue->pts;
	}

	if (ms->surface_queue->pts + ms->video_pts_offset <= offset_time + frame_early_delivery) {
		sqe = dequeue_surface(&ms->surface_queue);
		ms->surface_queue_size -= 1;

	}

done:

    /* Only signal if we've consumed something. */
	if (sqe) {
		ms->needs_decode = 1;
		ms->video_read_time = offset_time;
		SDL_CondBroadcast(ms->cond);
	}

	SDL_UnlockMutex(ms->lock);

	if (sqe) {
		rv = SDL_CreateRGBSurfaceFrom(
			sqe->pixels,
			sqe->w,
			sqe->h,
			sqe->format->BitsPerPixel,
			sqe->pitch,
			sqe->format->Rmask,
			sqe->format->Gmask,
			sqe->format->Bmask,
			sqe->format->Amask
		);

		/* Force SDL to take over management of pixels. */
		rv->flags &= ~SDL_PREALLOC;
		av_free(sqe);
	}

	return rv;
}


static int decode_thread(void *arg) {
	MediaState *ms = (MediaState *) arg;

	int err;

	AVFormatContext *ctx = avformat_alloc_context();
	if (ctx == NULL) {
		goto finish;
	}
	ms->ctx = ctx;

	AVIOContext *io_context = rwops_open(ms->rwops);
	if (io_context == NULL) {
		goto finish;
	}
	ctx->pb = io_context;

	err = avformat_open_input(&ctx, ms->filename, NULL, NULL);
	if (err) {
		avformat_free_context(ctx);
		ms->ctx = NULL;
		goto finish;
	}

	err = avformat_find_stream_info(ctx, NULL);
	if (err) {
		goto finish;
	}


	ms->video_stream = -1;
	ms->audio_stream = -1;

	for (unsigned int i = 0; i < ctx->nb_streams; i++) {
		if (ctx->streams[i]->codecpar->codec_type == AVMEDIA_TYPE_VIDEO) {
			if (ms->want_video && ms->video_stream == -1) {
				ms->video_stream = i;
			}
		}

		if (ctx->streams[i]->codecpar->codec_type == AVMEDIA_TYPE_AUDIO) {
			if (ms->audio_stream == -1) {
				ms->audio_stream = i;
			}
		}
	}

	ms->video_context = find_context(ctx, ms->video_stream);
	ms->audio_context = find_context(ctx, ms->audio_stream);

	ms->swr = swr_alloc();
	if (ms->swr == NULL) {
		goto finish;
	}

	// Compute the number of samples we need to play back.
	if (ms->audio_duration < 0) {
		if (av_fmt_ctx_get_duration_estimation_method(ctx) != AVFMT_DURATION_FROM_BITRATE) {

			long long duration = ((long long) ctx->duration) * audio_sample_rate;
			ms->audio_duration = (unsigned int) (duration /  AV_TIME_BASE);

			ms->total_duration = 1.0 * ctx->duration / AV_TIME_BASE;

			// Check that the duration is reasonable (between 0s and 3600s). If not,
			// reject it.
			if (ms->audio_duration < 0 || ms->audio_duration > 3600 * audio_sample_rate) {
				ms->audio_duration = -1;
			}

			ms->audio_duration -= (unsigned int) (ms->skip * audio_sample_rate);


		} else {
			ms->audio_duration = -1;
		}
	}

	if (ms->skip != 0.0) {
		av_seek_frame(ctx, -1, (int64_t) (ms->skip * AV_TIME_BASE), AVSEEK_FLAG_BACKWARD);
	}

	while (!ms->quit) {

		if (! ms->audio_finished) {
			decode_audio(ms);
		}

		if (! ms->video_finished) {
			decode_video(ms);
		}

		SDL_LockMutex(ms->lock);

		if (!ms->ready) {
			ms->ready = 1;
			SDL_CondBroadcast(ms->cond);
		}

		if (!(ms->needs_decode || ms->quit)) {
			SDL_CondWait(ms->cond, ms->lock);
		}

		ms->needs_decode = 0;

		SDL_UnlockMutex(ms->lock);
	}


finish:
	/* Data used by the decoder should be freed here, while data shared with
	 * the readers should be freed in media_close.
	 */

	SDL_LockMutex(ms->lock);

	/* Ensures that every stream becomes ready. */
	if (!ms->ready) {
		ms->ready = 1;
		SDL_CondBroadcast(ms->cond);
	}

	while (!ms->quit) {
		SDL_CondWait(ms->cond, ms->lock);
	}

	SDL_UnlockMutex(ms->lock);

	deallocate(ms);

	return 0;
}


void media_read_sync_finish(struct MediaState *ms) {
	// copy/paste from end of decode_thread

	/* Data used by the decoder should be freed here, while data shared with
	 * the readers should be freed in media_close.
	 */

	SDL_LockMutex(ms->lock);

	/* Ensures that every stream becomes ready. */
	if (!ms->ready) {
		ms->ready = 1;
		SDL_CondBroadcast(ms->cond);
	}

	while (!ms->quit) {
		/* SDL_CondWait(ms->cond, ms->lock); */
	}

	SDL_UnlockMutex(ms->lock);

	deallocate(ms);
}


static int decode_sync_start(void *arg) {
    // copy/paste from start of decode_thread
	MediaState *ms = (MediaState *) arg;

	int err;

	AVFormatContext *ctx = avformat_alloc_context();
	if (ctx == NULL) {
		media_read_sync_finish(ms);
	}
	ms->ctx = ctx;

	AVIOContext *io_context = rwops_open(ms->rwops);
	if (io_context == NULL) {
		media_read_sync_finish(ms);
	}
	ctx->pb = io_context;

	err = avformat_open_input(&ctx, ms->filename, NULL, NULL);
	if (err) {
		avformat_free_context(ctx);
		ms->ctx = NULL;
		media_read_sync_finish(ms);
	}

	err = avformat_find_stream_info(ctx, NULL);
	if (err) {
		media_read_sync_finish(ms);
	}


	ms->video_stream = -1;
	ms->audio_stream = -1;

	for (unsigned int i = 0; i < ctx->nb_streams; i++) {
		if (ctx->streams[i]->codecpar->codec_type == AVMEDIA_TYPE_VIDEO) {
			if (ms->want_video && ms->video_stream == -1) {
				ms->video_stream = i;
			}
		}

		if (ctx->streams[i]->codecpar->codec_type == AVMEDIA_TYPE_AUDIO) {
			if (ms->audio_stream == -1) {
				ms->audio_stream = i;
			}
		}
	}

	ms->video_context = find_context(ctx, ms->video_stream);
	ms->audio_context = find_context(ctx, ms->audio_stream);

	ms->swr = swr_alloc();
	if (ms->swr == NULL) {
		media_read_sync_finish(ms);
	}

	// Compute the number of samples we need to play back.
	if (ms->audio_duration < 0) {
		if (av_fmt_ctx_get_duration_estimation_method(ctx) != AVFMT_DURATION_FROM_BITRATE) {

			long long duration = ((long long) ctx->duration) * audio_sample_rate;
			ms->audio_duration = (unsigned int) (duration /  AV_TIME_BASE);

			ms->total_duration = 1.0 * ctx->duration / AV_TIME_BASE;

			// Check that the duration is reasonable (between 0s and 3600s). If not,
			// reject it.
			if (ms->audio_duration < 0 || ms->audio_duration > 3600 * audio_sample_rate) {
				ms->audio_duration = -1;
			}

			ms->audio_duration -= (unsigned int) (ms->skip * audio_sample_rate);


		} else {
			ms->audio_duration = -1;
		}
	}

	if (ms->skip != 0.0) {
		av_seek_frame(ctx, -1, (int64_t) (ms->skip * AV_TIME_BASE), AVSEEK_FLAG_BACKWARD);
	}

	// [snip!]

	return 0;
}


void media_read_sync(struct MediaState *ms) {
	// copy/paste from middle of decode_thread
	// printf("---* media_read_sync %p\n", ms);

	//while (!ms->quit) {
	if (!ms->quit) {
		// printf("     audio_finished: %d, video_finished: %d\n", ms->audio_finished, ms->video_finished);
		if (! ms->audio_finished) {
			decode_audio(ms);
		}

		if (! vs->video_finished) {
			decode_video(vs);
		}

		SDL_LockMutex(vs->lock);

		if (!vs->ready) {
			vs->ready = 1;
			SDL_CondBroadcast(vs->cond);
		}

		if (!(vs->needs_decode || vs->quit)) {
			/* SDL_CondWait(ms->cond, ms->lock); */
		}

		vs->needs_decode = 0;

		SDL_UnlockMutex(vs->lock);
	}
}


int media_read_audio(struct VideoState *vs, Uint8 *stream, int len) {
#ifdef __EMSCRIPTEN__
    media_read_sync(vs);
#endif

	SDL_LockMutex(vs->lock);

    if(!vs->ready) {
	    SDL_UnlockMutex(vs->lock);
	    memset(stream, 0, len);
	    return len;
	}

	int rv = 0;

	if (vs->audio_duration >= 0) {
		int remaining = (vs->audio_duration - vs->audio_read_samples) * BPS;
		if (len > remaining) {
			len = remaining;
		}

		if (!remaining) {
			vs->audio_finished = 1;
		}

	}

	while (len) {

		if (!vs->audio_out_frame) {
			vs->audio_out_frame = dequeue_frame(&vs->audio_queue);
			vs->audio_out_index = 0;
		}

		if (!vs->audio_out_frame) {
			break;
		}

		AVFrame *f = vs->audio_out_frame;

		int avail = f->nb_samples * BPS - vs->audio_out_index;
		int count;

		if (len > avail) {
			count = avail;
		} else {
			count = len;
		}

		memcpy(stream, &f->data[0][vs->audio_out_index], count);

		vs->audio_out_index += count;

		vs->audio_read_samples += count / BPS;
		vs->audio_queue_samples -= count / BPS;

		rv += count;
		len -= count;
		stream += count;

		if (vs->audio_out_index >= f->nb_samples * BPS) {
			av_frame_free(&vs->audio_out_frame);
			vs->audio_out_index = 0;
		}
	}

	/* Only signal if we've consumed something. */
	if (rv) {
		vs->needs_decode = 1;
		SDL_CondBroadcast(vs->cond);
	}

	SDL_UnlockMutex(vs->lock);

	if (vs->audio_duration >= 0) {
		if ((vs->audio_duration - vs->audio_read_samples) * BPS < len) {
			len = (vs->audio_duration - vs->audio_read_samples) * BPS;
		}

		memset(stream, 0, len);
		vs->audio_read_samples += len / BPS;
		rv += len;
	}

	return rv;
}

void media_wait_ready(struct VideoState *vs) {
#ifndef __EMSCRIPTEN__
    SDL_LockMutex(vs->lock);

    while (!vs->ready) {
        SDL_CondWait(vs->cond, vs->lock);
    }

    SDL_UnlockMutex(vs->lock);
#endif
}


double media_duration(VideoState *vs) {
	return vs->total_duration;
}

void media_start(VideoState *vs) {

#ifdef __EMSCRIPTEN__
    decode_sync_start(ms);
#else

    char buf[1024];

	snprintf(buf, 1024, "decode: %s", vs->filename);
	SDL_Thread *t = SDL_CreateThread(decode_thread, buf, (void *) vs);
	vs->thread = t;
#endif
}


VideoState *media_open(SDL_RWops *rwops, const char *filename) {

    deallocate_deferred();

    VideoState *vs = av_calloc(1, sizeof(VideoState));
	if (vs == NULL) {
		return NULL;
	}

	vs->filename = av_strdup(filename);
	if (vs->filename == NULL) {
		deallocate(vs);
		return NULL;
	}
	vs->rwops = rwops;

#ifndef __EMSCRIPTEN__
	vs->cond = SDL_CreateCond();
	if (vs->cond == NULL) {
		deallocate(vs);
		return NULL;
	}
	vs->lock = SDL_CreateMutex();
	if (vs->lock == NULL) {
		deallocate(vs);
		return NULL;
	}
#endif

	vs->audio_duration = -1;
	vs->frame_drops = 1;

	return vs;
}

/**
 * Sets the start and end of the stream. This must be called before
 * media_start.
 *
 * start
 *    The time in the stream at which the media starts playing.
 * end
 *    If not 0, the time at which the stream is forced to end if it has not
 *    already. If 0, the stream plays until its natural end.
 */
void media_start_end(VideoState *vs, double start, double end) {
	vs->skip = start;

	if (end >= 0) {
		if (end < start) {
			vs->audio_duration = 0;
		} else {
			vs->audio_duration = (int) ((end - start) * audio_sample_rate);
		}
	}
}

/**
 * Marks the channel as having video.
 */
void media_want_video(VideoState *vs, int video) {
	vs->want_video = 1;
	vs->frame_drops = (video != 2);
}

void media_pause(VideoState *vs, int pause) {
    if (pause && (vs->pause_time == 0)) {
        vs->pause_time = current_time;
    } else if ((!pause) && (vs->pause_time > 0)) {
        vs->time_offset += current_time - vs->pause_time;
        vs->pause_time = 0;
    }
}

void media_close(VideoState *vs) {

	if (!vs->thread) {
		deallocate(vs);
		return;
	}

	/* Tell the decoder to terminate. It will deallocate everything for us. */
	SDL_LockMutex(vs->lock);
	vs->quit = 1;

#ifdef __EMSCRIPTEN__
	media_read_sync_finish(ms);
#endif

	SDL_CondBroadcast(vs->cond);
	SDL_UnlockMutex(vs->lock);

}

void media_advance_time(void) {
	current_time = SPEED * av_gettime() * 1e-6;
}

void media_sample_surfaces(SDL_Surface *rgb, SDL_Surface *rgba) {
	rgb_surface = rgb;
	rgba_surface = rgba;
}

void media_init(int rate, int status, int equal_mono) {

    deallocate_mutex = SDL_CreateMutex();

	audio_sample_rate = rate / SPEED;
	audio_equal_mono = equal_mono;

    if (status) {
        av_log_set_level(AV_LOG_INFO);
    } else {
        av_log_set_level(AV_LOG_ERROR);
    }

}