#include <libavcodec/avcodec.h>
#include <libavformat/avformat.h>
#include <libswresample/swresample.h>
#include <libavutil/time.h>
#include <libavutil/pixfmt.h>
#include <libswscale/swscale.h>

#include <SDL.h>
#include <SDL_thread.h>

#include <stdlib.h>

typedef struct PacketQueueEntry {
	AVPacket* pkt;
	struct PacketQueueEntry* next;
} PacketQueueEntry;

typedef struct PacketQueue {
	PacketQueueEntry* first;
	PacketQueueEntry* last;
} PacketQueue;

typedef struct FrameQueue {
	AVFrame* first;
	AVFrame* last;
} FrameQueue;

typedef struct SurfaceQueueEntry {
	struct SurfaceQueueEntry* next;

	SDL_Surface* surf;

	/* The pts, converted to seconds */
	double pts;

	/* The format. This is not refcounted, but it's kept alive by being
	 * the format of one of the sampel surfaces
	 */
	SDL_PixelFormat* format;

	/* As with SDL_Surface */
	int w, h, pitch;
	void* pixels;

} SurfaceQueueEntry;

typedef struct VideoState {
	/* The next entry in a list of MediaStates */
    struct VideoState* next;

    SDL_Thread* thread;
    
    /* The condition and lock */
    SDL_cond* cond;
	SDL_mutex* lock;

	SDL_RWops *rwops;

    char* filename;

    /*
	 * True if we this stream should have video.
	 */
	int want_video;

	/* This becomes true once the decode thread has finished initializing
	 * and the readers and writers can do their thing.
	 */
	int ready; // Lock.

	/* This is set to true when data has been read, in order to ask the
	 * decode thread to produce more data.
	 */
	int needs_decode; // Lock.

	/*
	 * This is set to true when data has been read, in order to ask the
	 * decode thread to shut down and deallocate all resources.
	 */
	int quit; // Lock

	/* The number of seconds to skip at the start. */
	double skip;

	/* These become true when the audio and video finish. */
	int audio_finished;
	int video_finished;

	/* Indexes of video and audio streams. */
	int video_stream;
	int audio_stream;

    /* The main context */
    AVFormatContext* ctx;

    /* Contexts for decoding audio and video streams */
	AVCodecContext* video_context;
	AVCodecContext* audio_context;

    /* Queues of packets going to the audio and video streams */
	PacketQueue video_packet_queue;
	PacketQueue audio_packet_queue;

    /* The total duration of the video. Only used for information purposes. */
	double total_duration;

	/* Audio Stuff ***********************************************************/

	/* The queue of converted audio frames. */
	FrameQueue audio_queue; // Lock

	/* The size of the audio queue, and the target size in seconds. */
	int audio_queue_samples;
	int audio_queue_target_samples;

	/* A frame used for decoding. */
	AVFrame *audio_decode_frame;

	/* The audio frame being read from, and the index into the audio frame. */
	AVFrame *audio_out_frame; // Lock
	int audio_out_index; // Lock

	SwrContext *swr;

	/* The duration of the audio stream, in samples.
	 * -1 means to play until we run out of data.
	 */
	int audio_duration;

	/* The number of samples that have been read so far. */
	int audio_read_samples; // Lock

	/* A frame that video is decoded into. */
	AVFrame *video_decode_frame;

    /* Video Stuff ***********************************************************/

	/* Software rescaling context. */
	struct SwsContext *sws;

	/* A queue of decoded video frames. */
	SurfaceQueueEntry *surface_queue; // Lock
	int surface_queue_size; // Lock

	/* The offset between a pts timestamp and realtime. */
	double video_pts_offset;

	/* The wall time the last video frame was read. */
	double video_read_time;

	/* Are frame drops allowed? */
	int frame_drops;

	/* The time the pause happened, or 0 if we're not paused. */
	double pause_time;

	/* The offset between now and the time of the current frame, at least for video. */
	double time_offset;
} VideoState;

#ifndef _WIN32
#define USE_POSIX_MEMALIGN
#endif

/* Should a mono channel be split into two equal stero channels (true) or
 * should the energy be split onto two stereo channels with 1/2 the energy
 * (false).
 */
static int audio_equal_mono = 1;

/* The weight of stereo channels when audio_equal_mono is true. */
static double stereo_matrix[] = { 1.0, 1.0 };

/* The output audio sample rate. */
static int audio_sample_rate = 44100;

static int audio_sample_increase = 44100 / 5;
static int audio_target_samples = 44100 * 2;

const int CHANNELS = 2;
const int BPC = 2; // Bytes per channel.
const int BPS = 4; // Bytes per sample.

const int FRAMES = 3;

// The alignment of each row of pixels.
const int ROW_ALIGNMENT = 16;

// The number of pixels on each side. This has to be greater that 0 (since
// Ren'G needs some padding), FRAME_PADDING * BPS has to be a multiple of
// 16 (alignment issues on ARM NEON), and has to match the crop in the
// read_video function of renpysound.pyx.
const int FRAME_PADDING = ROW_ALIGNMENT / 4;

const int SPEED = 1;

// How many seconds early can frames be delivered?
static const double frame_early_delivery = .005;

VideoState* deallocate_queue = NULL;
SDL_mutex* deallocate_mutex = NULL;

VideoState* VideoInit(char*);
void FreeVideoState(VideoState*);

#define RWOPS_BUFFER 65536

int rwops_read(void*, uint8_t*, int);
AVIOContext* rwops_open(SDL_RWops*);
void rwops_close(SDL_RWops*);

void deallocate_deferred();

/* Frame queue *****************************************/

void enqueue_frame(FrameQueue*, AVFrame*); 
AVFrame* dequeue_frame(FrameQueue*);

/* Packet queue ****************************************/

void enqueue_packet(PacketQueue*, AVPacket*);
AVPacket* first_packet(PacketQueue*);
void dequeue_packet(PacketQueue*);
int count_packet_queue(PacketQueue*);
void free_packet_queue(PacketQueue*);

/* Surface queue ***************************************/

void enqueue_surface(SurfaceQueueEntry**, SurfaceQueueEntry*);
SurfaceQueueEntry* dequeue_surface(SurfaceQueueEntry**);