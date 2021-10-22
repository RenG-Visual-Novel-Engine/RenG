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

SDL_Rect createRect(int x, int y, int w, int h)
{
	SDL_Rect Quad = { x, y, w, h };
	return Quad;
}
*/
import "C"

func CreateColor(r, g, b int) SDL_Color {
	return (SDL_Color)(C.colorSet(C.int(r), C.int(g), C.int(b)))
}

func CreateRect(x, y, w, h int) C.SDL_Rect {
	return C.createRect(C.int(x), C.int(y), C.int(w), C.int(h))
}
