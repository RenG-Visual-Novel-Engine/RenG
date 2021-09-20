package main

/*
#cgo LDFLAGS: -L./lib -lSDL2 -lSDL2main -lSDL2_image -lSDL2_ttf -lSDL2_mixer

#include <include/SDL.h>
#include <include/SDL_image.h>
#include <include/SDL_ttf.h>
#include <include/SDL_mixer.h>

#define SDL_BlitSurface SDL_UpperBlit

Mix_Chunk* MixLoadWAV(const char* file)
{
	return Mix_LoadWAV_RW(SDL_RWFromFile(file, "rb"), 1);
}

Uint32 eventType(SDL_Event event)
{
	return event.type;
}

SDL_Keycode key(SDL_Event event)
{
	return event.key.keysym.sym;
}
*/
import "C"
import (
	"fmt"
	"runtime"
)

const (
	width  = 640
	height = 480
)

const (
	IMG_INIT_PNG = C.IMG_INIT_PNG
)

var (
	window   *C.SDL_Window
	renderer *C.SDL_Renderer
	music    *C.Mix_Music
)

var (
	scratch *C.Mix_Chunk
	high    *C.Mix_Chunk
	medium  *C.Mix_Chunk
	low     *C.Mix_Chunk

	texture *C.SDL_Texture
)

func init() {
	runtime.LockOSThread()
}

func SDLInit() bool {

	if int(C.SDL_Init(C.SDL_INIT_VIDEO|C.SDL_INIT_AUDIO)) < 0 {
		fmt.Println("Init error")
		return false
	}

	C.SDL_SetHint(C.CString(C.SDL_HINT_RENDER_SCALE_QUALITY), C.CString("1"))

	window = C.SDL_CreateWindow(C.CString("SDL2 테스트"), C.SDL_WINDOWPOS_UNDEFINED, C.SDL_WINDOWPOS_UNDEFINED, width, height, C.SDL_WINDOW_SHOWN)

	if window == nil {
		fmt.Println("WindowError")
		return false
	}

	renderer = C.SDL_CreateRenderer(window, -1, C.SDL_RENDERER_ACCELERATED|C.SDL_RENDERER_PRESENTVSYNC)
	C.SDL_SetRenderDrawColor(renderer, 0xFF, 0xFF, 0xFF, 0xFF)

	if (C.IMG_Init(IMG_INIT_PNG) & IMG_INIT_PNG) == 0 {
		fmt.Println("SDLImage Error")
		return false
	}

	if C.Mix_OpenAudio(44100, C.MIX_DEFAULT_FORMAT, 2, 2048) < 0 {
		fmt.Println("SDLMixer Error")
		return false
	}

	return true
}

func LoadFromFile(path string) bool {
	var newTexture *C.SDL_Texture

	loadedSurface := C.IMG_Load(C.CString(path))
	if loadedSurface == nil {
		fmt.Println("loadedSurface Error")
		return false
	}
	newTexture = C.SDL_CreateTextureFromSurface(renderer, loadedSurface)
	C.SDL_FreeSurface(loadedSurface)

	texture = newTexture
	return true
}

func LoadMedia() bool {
	if !LoadFromFile("src\\SDL\\test\\prompt.png") {
		return false
	}

	music = C.Mix_LoadMUS(C.CString("src\\SDL\\test\\beat.wav"))
	if music == nil {
		return false
	}

	scratch = C.MixLoadWAV(C.CString("src\\SDL\\test\\scratch.wav"))
	if scratch == nil {
		return false
	}

	high = C.MixLoadWAV(C.CString("src\\SDL\\test\\high.wav"))
	if high == nil {
		return false
	}

	medium = C.MixLoadWAV(C.CString("src\\SDL\\test\\medium.wav"))
	if medium == nil {
		return false
	}

	low = C.MixLoadWAV(C.CString("src\\SDL\\test\\low.wav"))
	if low == nil {
		return false
	}

	return true
}

func Close() {
	C.SDL_DestroyTexture(texture)

	C.Mix_FreeChunk(scratch)
	C.Mix_FreeChunk(high)
	C.Mix_FreeChunk(medium)
	C.Mix_FreeChunk(low)

	C.Mix_FreeMusic(music)

	C.SDL_DestroyRenderer(renderer)
	C.SDL_DestroyWindow(window)

	C.Mix_Quit()
	C.IMG_Quit()
	C.SDL_Quit()
}

func main() {
	if !SDLInit() {
		fmt.Println("fail SDLInit")
	}

	if !LoadMedia() {
		fmt.Println("fail LoadMedia")
	}

	quit := false

	var e C.SDL_Event

	for !quit {
		for C.SDL_PollEvent(&e) != 0 {
			if C.eventType(e) == C.SDL_QUIT {
				quit = true
			} else if C.eventType(e) == C.SDL_KEYDOWN {
				switch C.key(e) {
				case C.SDLK_1:
					C.Mix_PlayChannelTimed(-1, high, 0, -1)
				case C.SDLK_2:
					C.Mix_PlayChannelTimed(-1, medium, 0, -1)
				case C.SDLK_3:
					C.Mix_PlayChannelTimed(-1, low, 0, -1)
				case C.SDLK_4:
					C.Mix_PlayChannelTimed(-1, scratch, 0, -1)
				case C.SDLK_9:
					if C.Mix_PlayingMusic() == 0 {
						C.Mix_PlayMusic(music, -1)
					} else {
						if C.Mix_PausedMusic() == 1 {
							C.Mix_ResumeMusic()
						} else {
							C.Mix_PauseMusic()
						}
					}
				case C.SDLK_0:
					C.Mix_HaltMusic()
				}
			}
		}
		C.SDL_SetRenderDrawColor(renderer, 0xFF, 0xFF, 0xFF, 0xFF)
		C.SDL_RenderClear(renderer)

		C.SDL_RenderCopy(renderer, texture, nil, nil)
		C.SDL_RenderPresent(renderer)
	}

	Close()
}
