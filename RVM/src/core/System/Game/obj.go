package game

import (
	animation "RenG/RVM/src/core/System/Game/Graphic/Animation"
	"RenG/RVM/src/core/obj"
)

func (g *Game) evalShow(s *obj.Show, bps int) {
	if s.T.Size.X == 0 {
		s.T.Size.X = g.Graphic.GetImageWidth(s.Name)
	}

	if s.T.Size.Y == 0 {
		s.T.Size.Y = g.Graphic.GetImageHeight(s.Name)
	}

	g.Graphic.AddScreenTextureRenderBuffer(
		bps,
		g.Graphic.GetImageTexture(s.Name),
		s.T,
	)

	if s.Anime != nil {
		switch s.Anime.Type {
		case animation.ANIME_ALPHA:
			g.Graphic.SetImageAlphaByName(s.Name, int(s.Anime.InitValue))
			g.Graphic.AddAnimation(s.Anime, bps, g.Graphic.GetCurrentTopScreenIndexByBps(bps))
		case animation.ANIME_ROTATE:
			g.Graphic.SetRotateByBps(bps, g.Graphic.GetCurrentTopScreenIndexByBps(bps), int(s.Anime.InitValue))
			g.Graphic.AddAnimation(s.Anime, bps, g.Graphic.GetCurrentTopScreenIndexByBps(bps))
		}
	}
}

func (g *Game) evalPlayVideo(pv *obj.PlayVideo, bps int) {
	g.Graphic.AddScreenTextureRenderBuffer(
		bps,
		g.Graphic.GetVideoTexture(pv.Name),
		pv.T,
	)
	g.Graphic.VideoStart(pv.Name)

	if pv.Anime != nil {
		switch pv.Anime.Type {
		case animation.ANIME_ALPHA:
			g.Graphic.SetVideoAlphaByName(pv.Name, int(pv.Anime.InitValue))
			g.Graphic.AddAnimation(pv.Anime, bps, g.Graphic.GetCurrentTopScreenIndexByBps(bps))
		case animation.ANIME_ROTATE:
			g.Graphic.SetRotateByBps(bps, g.Graphic.GetCurrentTopScreenIndexByBps(bps), int(pv.Anime.InitValue))
			g.Graphic.AddAnimation(pv.Anime, bps, g.Graphic.GetCurrentTopScreenIndexByBps(bps))
		}
	}
}

func (g *Game) evalText(t *obj.Text, bps int) {
	g.Graphic.AddScreenTextureRenderBuffer(
		bps,
		g.Graphic.GetTextTexture(t.Text, t.FontName, t.Color),
		t.T,
	)
}
