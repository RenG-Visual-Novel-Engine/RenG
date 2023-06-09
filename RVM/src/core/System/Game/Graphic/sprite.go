package graphic

import (
	"math"
	"time"
)

func (g *Graphic) UpdateSprite() {
	g.lock.Lock()
	defer g.lock.Unlock()

	for key, screen := range g.sprites {
		for n, sprite := range screen {
			s := time.Since(sprite.StartTime).Seconds()

			if s >= sprite.Duration {
				g.renderBuffer[sprite.Bps][sprite.Index].texture = g.Sprite_Manager.GetSpriteImage(sprite.Name, g.Sprite_Manager.GetSpriteSize(sprite.Name)-1)
				if sprite.Loop {
					g.sprites[key][n].StartTime = time.Now()
					continue
				}
				g.sprites[key] = append(g.sprites[key][:n], g.sprites[key][n+1:]...)
				continue
			}

			g.renderBuffer[sprite.Bps][sprite.Index].texture = g.Sprite_Manager.GetSpriteImage(sprite.Name, int(math.Round(float64(g.Sprite_Manager.GetSpriteSize(sprite.Name)-1)*(s/sprite.Duration))))
		}
	}
}

func (g *Graphic) AddSprite(
	screenName,
	name string,
	bps,
	index int,
	duration float64,
	loop bool,
) {
	g.lock.Lock()
	defer g.lock.Unlock()

	_, ok := g.sprites[screenName]
	if !ok {
		g.sprites[screenName] = []struct {
			Name      string
			Bps       int
			Index     int
			Duration  float64
			Loop      bool
			StartTime time.Time
		}{}
	}
	g.sprites[screenName] = append(g.sprites[screenName], struct {
		Name      string
		Bps       int
		Index     int
		Duration  float64
		Loop      bool
		StartTime time.Time
	}{
		Name:      name,
		Bps:       bps,
		Index:     index,
		Duration:  duration,
		Loop:      loop,
		StartTime: time.Now(),
	})
}

func (g *Graphic) UpdateSpriteScreenBPS(screenName string, bps int) {
	g.lock.Lock()
	defer g.lock.Unlock()

	for n, _ := range g.sprites[screenName] {
		g.sprites[screenName][n].Bps = bps
	}
}

func (g *Graphic) DeleteSpriteByScreenName(screenName string) {
	g.lock.Lock()
	defer g.lock.Unlock()

	delete(g.sprites, screenName)
}
