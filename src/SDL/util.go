package sdl

/*
#cgo LDFLAGS: -L./lib -lSDL2 -lSDL2main -lSDL2_image -lSDL2_ttf -lSDL2_mixer

#include <include/SDL.h>
#include <include/SDL_image.h>
#include <include/SDL_ttf.h>
#include <include/SDL_mixer.h>

SDL_Color colorSet(int r, int g, int b)
{
	SDL_Color textColor = { 0, 0, 0 };
	return textColor;
}
*/
import "C"

func Color(r, g, b int) SDL_Color {
	return (SDL_Color)(C.colorSet(C.int(r), C.int(g), C.int(b)))
}
