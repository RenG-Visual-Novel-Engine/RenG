package graphic

import (
	animation "RenG/RVM/src/core/System/Game/Graphic/Animation"
	"time"
)

func (g *Graphic) AddAnimation(
	anime *animation.Anime,
	bps int,
	index int,
) {
	g.lock.Lock()
	defer g.lock.Unlock()

	anime.Time = time.Now()
	g.animations = append(g.animations, struct {
		Anime *animation.Anime
		Bps   int
		Index int
	}{
		Anime: anime,
		Bps:   bps,
		Index: index,
	})
}
