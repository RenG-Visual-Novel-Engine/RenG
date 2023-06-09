package graphic

import (
	"RenG/RVM/src/core/obj"
	"time"
)

func (g *Graphic) UpdateAnimation() {
	g.lock.Lock()
	defer g.lock.Unlock()

	var DeleteStack []struct {
		name         string
		textureIndex int
		animeIndex   int
	}

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
						g.Video_Manager.Lock()
						g.Image_Manager.ChangeTextureAlpha(g.renderBuffer[anime.Bps][textureIndex].texture, anime.Anime.Curve(1))
						g.Video_Manager.Unlock()
					case obj.ANIME_ROTATE:
						g.Video_Manager.Lock()
						g.renderBuffer[anime.Bps][textureIndex].transform.Rotate = anime.Anime.Curve(1)
						g.Video_Manager.Unlock()
					case obj.ANIME_XPOS:
						g.Video_Manager.Lock()
						g.renderBuffer[anime.Bps][textureIndex].transform.Pos.X = anime.Anime.Curve(1)
						g.Video_Manager.Unlock()
					case obj.ANIME_YPOS:
						g.Video_Manager.Lock()
						g.renderBuffer[anime.Bps][textureIndex].transform.Pos.Y = anime.Anime.Curve(1)
						g.Video_Manager.Unlock()
					}
					if !anime.Anime.Loop {
						DeleteStack = append(DeleteStack, struct {
							name         string
							textureIndex int
							animeIndex   int
						}{
							name:         name,
							textureIndex: textureIndex,
							animeIndex:   n,
						})
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
					g.Video_Manager.Lock()
					g.Image_Manager.ChangeTextureAlpha(g.renderBuffer[anime.Bps][textureIndex].texture, anime.Anime.Curve((s-anime.Anime.StartTime)/anime.Anime.Duration))
					g.Video_Manager.Unlock()
				case obj.ANIME_ROTATE:
					g.Video_Manager.Lock()
					g.renderBuffer[anime.Bps][textureIndex].transform.Rotate = anime.Anime.Curve((s - anime.Anime.StartTime) / anime.Anime.Duration)
					g.Video_Manager.Unlock()
				case obj.ANIME_XPOS:
					g.Video_Manager.Lock()
					g.renderBuffer[anime.Bps][textureIndex].transform.Pos.X = anime.Anime.Curve((s - anime.Anime.StartTime) / anime.Anime.Duration)
					g.Video_Manager.Unlock()
				case obj.ANIME_YPOS:
					g.Video_Manager.Lock()
					g.renderBuffer[anime.Bps][textureIndex].transform.Pos.Y = anime.Anime.Curve((s - anime.Anime.StartTime) / anime.Anime.Duration)
					g.Video_Manager.Unlock()
				}
			}
		}
	}

	if len(DeleteStack) != 0 {
		for _, data := range DeleteStack {
			g.animations[data.name][data.textureIndex] = append(g.animations[data.name][data.textureIndex][:data.animeIndex], g.animations[data.name][data.textureIndex][data.animeIndex+1:]...)

			for target, temp := range DeleteStack {
				if data.name == temp.name && data.textureIndex == temp.textureIndex {
					if data.animeIndex < temp.animeIndex {
						DeleteStack[target].animeIndex--
					}
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

func (g *Graphic) UpdateAnimationScreenBPS(screenName string, bps int) {
	g.lock.Lock()
	defer g.lock.Unlock()

	for i, screen := range g.animations[screenName] {
		for n, _ := range screen {
			g.animations[screenName][i][n].Bps = bps
		}
	}
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
