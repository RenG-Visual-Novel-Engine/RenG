package main

import (
	"RenG/src/config"
	"RenG/src/core"
)

func MainLoop() {
	for !config.Quit {
		for config.Event.PollEvent() != 0 {
			switch config.Event.EventType() {
			case core.SDL_QUIT:
				config.Quit = true
			case core.SDL_WINDOWEVENT:
				switch config.Event.WindowEventType() {
				case core.SDL_WINDOWEVENT_SIZE_CHANGED:
					config.ChangeWidth, config.ChangeHeight = config.Event.ChangeWidthAndHeight()
				}
			case core.SDL_KEYDOWN:
				config.Event.HandleEvent(core.SDL_KEYDOWN, config.KeyDownEventChan)
			case core.SDL_MOUSEMOTION:
				config.Event.HandleEvent(core.SDL_MOUSEMOTION, config.MouseMotionEventChan)
			case core.SDL_MOUSEBUTTONDOWN:
				config.Event.HandleEvent(core.SDL_MOUSEBUTTONDOWN, config.MouseDownEventChan)
			case core.SDL_MOUSEBUTTONUP:
				config.Event.HandleEvent(core.SDL_MOUSEBUTTONUP, config.MouseUpEventChan)
			case core.SDL_MOUSEWHEEL:
				config.Event.HandleEvent(core.SDL_MOUSEWHEEL, config.MouseWheelEventChan)
			}
		}

		config.Renderer.RenderClear()
		config.Renderer.SetRenderDrawColor(0x00, 0x00, 0x00, 255)

		config.LayerMutex.Lock()
		for i := 0; i < len(config.LayerList.Layers); i++ {
			for j := 0; j < len(config.LayerList.Layers[i].Images); j++ {
				config.LayerList.Layers[i].Images[j].Render(config.Renderer, config.Width, config.Height, config.ChangeWidth, config.ChangeHeight)
			}
		}
		config.LayerMutex.Unlock()

		config.Renderer.RenderPresent()
	}
}
