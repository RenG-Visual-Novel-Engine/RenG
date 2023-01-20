#include <SDL.h>

Uint32 eventType(SDL_Event event)
{
	return event.type;
}

Sint32 eventKey(SDL_Event event)
{
	return event.key.keysym.sym;
}