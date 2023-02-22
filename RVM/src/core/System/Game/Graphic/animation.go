package graphic

import (
	"RenG/RVM/src/core/obj"
	"time"
)

func (g *Graphic) UpdateAnimation() {
	g.lock.Lock()
	defer g.lock.Unlock()

	for name, screen := range g.animations {
		for textureIndex, animes := range screen {
			for n, anime := range animes {
				s := time.Since(anime.Anime.Time).Seconds()

				if s < anime.Anime.StartTime {
					continue
				}

				if s-anime.Anime.StartTime >= anime.Anime.Duration {
					switch anime.Anime.Type {
					case obj.ANIME_ALPHA:
						g.Video.Lock()
						g.Image.ChangeTextureAlpha(g.renderBuffer[anime.Bps][textureIndex].texture, anime.Anime.Curve(1))
						g.Video.Unlock()
					case obj.ANIME_ROTATE:
						g.Video.Lock()
						g.renderBuffer[anime.Bps][textureIndex].transform.Rotate = anime.Anime.Curve(1)
						g.Video.Unlock()
					case obj.ANIME_XPOS:
						g.Video.Lock()
						g.renderBuffer[anime.Bps][textureIndex].transform.Pos.X = anime.Anime.Curve(1)
					case obj.ANIME_YPOS:
						g.Video.Lock()
						g.renderBuffer[anime.Bps][textureIndex].transform.Pos.Y = anime.Anime.Curve(1)
					}
					if !anime.Anime.Loop {
						g.animations[name][textureIndex] = append(g.animations[name][textureIndex][:n], g.animations[name][textureIndex][n+1:]...)
						if anime.Anime.End != nil {
							g.lock.Unlock()
							anime.Anime.End()
							g.lock.Lock()
						}
						continue
					}
					anime.Anime.StartTime = 0
					anime.Anime.Time = time.Now()
					continue
				}

				switch anime.Anime.Type {
				case obj.ANIME_ALPHA:
					g.Video.Lock()
					g.Image.ChangeTextureAlpha(g.renderBuffer[anime.Bps][textureIndex].texture, anime.Anime.Curve((s-anime.Anime.StartTime)/anime.Anime.Duration))
					g.Video.Unlock()
				case obj.ANIME_ROTATE:
					g.Video.Lock()
					g.renderBuffer[anime.Bps][textureIndex].transform.Rotate = anime.Anime.Curve((s - anime.Anime.StartTime) / anime.Anime.Duration)
					g.Video.Unlock()
				case obj.ANIME_XPOS:
					g.Video.Lock()
					g.renderBuffer[anime.Bps][textureIndex].transform.Pos.X = anime.Anime.Curve((s - anime.Anime.StartTime) / anime.Anime.Duration)
					g.Video.Unlock()
				case obj.ANIME_YPOS:
					g.Video.Lock()
					g.renderBuffer[anime.Bps][textureIndex].transform.Pos.Y = anime.Anime.Curve((s - anime.Anime.StartTime) / anime.Anime.Duration)
					g.Video.Unlock()
				}
			}
		}
	}
}

func (g *Graphic) AddAnimation(
	screenName string,
	anime *obj.Anime,
	bps int,
	index int,
) {
	g.lock.Lock()
	defer g.lock.Unlock()

	anime.Time = time.Now()
	_, ok := g.animations[screenName]
	if !ok {
		g.animations[screenName] = make(map[int][]struct {
			Anime *obj.Anime
			Bps   int
		})
	}
	g.animations[screenName][index] = append(g.animations[screenName][index], struct {
		Anime *obj.Anime
		Bps   int
	}{
		Anime: anime,
		Bps:   bps,
	})
}

func (g *Graphic) DeleteAnimationByTextureIndex(screenName string, textureIndex int) {
	g.lock.Lock()
	defer g.lock.Unlock()

	delete(g.animations[screenName], textureIndex)
}

func (g *Graphic) DeleteAnimationByScreenName(screenName string) {
	g.lock.Lock()
	defer g.lock.Unlock()

	delete(g.animations, screenName)
}
