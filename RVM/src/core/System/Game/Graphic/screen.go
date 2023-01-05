package graphic

import (
	animation "RenG/RVM/src/core/System/Game/Graphic/Animation"
	texture "RenG/RVM/src/core/System/Game/Graphic/Texture"
	"RenG/RVM/src/core/globaltype"
	"RenG/RVM/src/core/obj"
	"time"
)

func (g *Graphic) ActiveScreen(name string) {
	g.lock.Lock()
	defer g.lock.Unlock()

	screen := g.screens[name]
	bufferIndex := len(g.renderBuffer)

	g.renderBuffer = append(g.renderBuffer, []struct {
		texture   *globaltype.SDL_Texture
		transform obj.Transform
	}{})
	g.screenBps[name] = bufferIndex

	go g.screenEval(screen.Obj, bufferIndex)
}

func (g *Graphic) screenEval(
	so []obj.ScreenObject,
	bufferIndex int,
) {
	for _, object := range so {
		switch object := object.(type) {
		case *obj.Show:
			if object.T.Xsize == 0 {
				object.T.Xsize = g.images[object.Name].width
			}

			if object.T.Ysize == 0 {
				object.T.Ysize = g.images[object.Name].height
			}

			g.renderBuffer[bufferIndex] = append(
				g.renderBuffer[bufferIndex],
				struct {
					texture   *globaltype.SDL_Texture
					transform obj.Transform
				}{
					g.images[object.Name].texture,
					object.T,
				})

			if object.Anime != nil {
				switch object.Anime.Type {
				case animation.ANIME_ALPHA:
					//TODO
					g.lock.Lock()
					g.videos.Lock()
					texture.TextureAlphaChange(
						g.images[object.Name].texture,
						int(object.Anime.InitValue),
					)
					g.lock.Unlock()
					g.videos.Unlock()

					go func() {
						time.Sleep(time.Duration(float64(time.Second) * object.Anime.StartTime))

						start := time.Now()

						for {
							time.Sleep(time.Microsecond * 10)

							s := time.Since(start).Seconds()

							if s >= object.Anime.Duration {
								if object.Anime.Loop {
									start = time.Now()
									continue
								}
								break
							}

							g.lock.Lock()
							g.videos.Lock()
							texture.TextureAlphaChange(
								g.images[object.Name].texture,
								object.Anime.Curve(s/object.Anime.Duration),
							)
							g.lock.Unlock()
							g.videos.Unlock()
						}
					}()
				}
			}
		case *obj.PlayMusic:
			object.Audio.PlayMusic(object.Path, object.Loop)
		case *obj.PlayVideo:
			g.renderBuffer[bufferIndex] = append(
				g.renderBuffer[bufferIndex],
				struct {
					texture   *globaltype.SDL_Texture
					transform obj.Transform
				}{
					(*globaltype.SDL_Texture)(g.videos.GetTexture(object.Name)),
					object.T,
				},
			)
			g.videos.VideoStart(object.Name)
		case *obj.ActiveFunc:
			object.F()
		}
	}
}
