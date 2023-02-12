package graphic

import (
	"RenG/RVM/src/core/obj"
	"fmt"
	"time"
)

func (g *Graphic) UpdateAnimation() {
	g.lock.Lock()
	defer g.lock.Unlock()

	for name, screen := range g.animations {
		for n, anime := range screen {
			s := time.Since(anime.Anime.Time).Seconds()

			if s < anime.Anime.StartTime {
				continue
			}

			if s-anime.Anime.StartTime >= anime.Anime.Duration {
				if !anime.Anime.Loop {
					g.animations[name] = append(g.animations[name][:n], g.animations[name][n+1:]...)
					fmt.Println(screen, len(screen))
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
				g.Image.ChangeTextureAlpha(g.renderBuffer[anime.Bps][anime.Index].texture, anime.Anime.Curve((s-anime.Anime.StartTime)/anime.Anime.Duration))
				g.Video.Unlock()
			case obj.ANIME_ROTATE:
				g.Video.Lock()
				g.renderBuffer[anime.Bps][anime.Index].transform.Rotate = anime.Anime.Curve((s - anime.Anime.StartTime) / anime.Anime.Duration)
				g.Video.Unlock()
			case obj.ANIME_XPOS:
				g.Video.Lock()
				g.renderBuffer[anime.Bps][anime.Index].transform.Pos.X = anime.Anime.Curve((s - anime.Anime.StartTime) / anime.Anime.Duration)
				g.Video.Unlock()
			case obj.ANIME_YPOS:
				g.Video.Lock()
				g.renderBuffer[anime.Bps][anime.Index].transform.Pos.Y = anime.Anime.Curve((s - anime.Anime.StartTime) / anime.Anime.Duration)
				g.Video.Unlock()
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
	g.animations[screenName] = append(g.animations[screenName], struct {
		Anime *obj.Anime
		Bps   int
		Index int
	}{
		Anime: anime,
		Bps:   bps,
		Index: index,
	})
}

func (g *Graphic) DeleteAnimationByScreenName(screenName string) {
	g.lock.Lock()
	defer g.lock.Unlock()

	delete(g.animations, screenName)
}
