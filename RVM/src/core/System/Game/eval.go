package game

import (
	event "RenG/RVM/src/core/System/Game/Event"
	"RenG/RVM/src/core/globaltype"
	"RenG/RVM/src/core/obj"
	"unicode/utf8"
)

func (g *Game) evalShow(s *obj.Show, name string, bps int) {

	if s.T.Size.X != 0 && s.T.Size.Y != 0 {
		s.T = g.echoTransform(s.T, s.T.Size.X, s.T.Size.Y)
	} else {
		s.T = g.echoTransform(s.T, g.Graphic.Image.GetImageWidth(s.Name), g.Graphic.Image.GetImageHeight(s.Name))
	}

	g.Graphic.AddScreenTextureRenderBuffer(
		bps,
		g.Graphic.Image.GetImageTexture(s.Name),
		s.T,
	)

	if len(s.Anime) != 0 {
		for _, anime := range s.Anime {
			switch anime.Type {
			case obj.ANIME_ALPHA:
				g.Graphic.Image.SetImageAlpha(s.Name, int(anime.InitValue))
				g.Graphic.AddAnimation(name, anime, bps, g.Graphic.GetCurrentTopScreenIndexByBps(bps))
			case obj.ANIME_ROTATE:
				g.Graphic.SetRotateByBps(bps, g.Graphic.GetCurrentTopScreenIndexByBps(bps), int(anime.InitValue))
				g.Graphic.AddAnimation(name, anime, bps, g.Graphic.GetCurrentTopScreenIndexByBps(bps))
			}
		}
	}
}

func (g *Game) evalPlayVideo(pv *obj.PlayVideo, name string, bps int) {
	g.Graphic.AddScreenTextureRenderBuffer(
		bps,
		g.Graphic.GetVideoTexture(pv.Name),
		pv.T,
	)
	g.Graphic.VideoStart(name, pv.Name, pv.Loop)

	if len(pv.Anime) != 0 {
		for _, anime := range pv.Anime {
			switch anime.Type {
			case obj.ANIME_ALPHA:
				g.Graphic.SetVideoAlphaByName(pv.Name, int(anime.InitValue))
				g.Graphic.AddAnimation(name, anime, bps, g.Graphic.GetCurrentTopScreenIndexByBps(bps))
			case obj.ANIME_ROTATE:
				g.Graphic.SetRotateByBps(bps, g.Graphic.GetCurrentTopScreenIndexByBps(bps), int(anime.InitValue))
				g.Graphic.AddAnimation(name, anime, bps, g.Graphic.GetCurrentTopScreenIndexByBps(bps))
			}
		}
	}
}

func (g *Game) evalText(t *obj.Text, name string, bps int) {
	if t.Text != "" {
		return
	}

	if t.TypingFX {
		var data []struct {
			Texture   *globaltype.SDL_Texture
			Transform obj.Transform
		}
		for index, runeValue := range t.Text {
			texture, width, height := g.Graphic.GetTextTexture(t.Text[0:index]+string(runeValue), t.FontName, t.Color)
			g.Graphic.RegisterTextMemPool(name, texture)
			t.T = g.echoTransform(t.T, width, height)
		}
		g.Graphic.AddScreenTextureRenderBuffer(
			bps,
			data[0].Texture,
			data[0].Transform,
		)
		g.Graphic.RegisterTypingFX(data, name, float64(utf8.RuneCountInString(t.Text))/g.TextSpeed, bps, g.Graphic.GetCurrentTopScreenIndexByBps(bps))
		return
	}

	texture, width, height := g.Graphic.GetTextTexture(t.Text, t.FontName, t.Color)
	g.Graphic.RegisterTextMemPool(name, texture)
	t.T = g.echoTransform(t.T, width, height)

	g.Graphic.AddScreenTextureRenderBuffer(
		bps,
		texture,
		t.T,
	)
}

func (g *Game) evalTextPointer(t *obj.TextPointer, name string, bps int) {
	if *t.Text == "" {
		return
	}

	if t.TypingFX {
		var data []struct {
			Texture   *globaltype.SDL_Texture
			Transform obj.Transform
		}
		for index, runeValue := range *t.Text {
			var transform obj.Transform
			texture, width, height := g.Graphic.GetTextTexture((*t.Text)[0:index]+string(runeValue), t.FontName, t.Color)
			g.Graphic.RegisterTextMemPool(name, texture)
			transform = g.echoTransform(t.T, width, height)
			data = append(data, struct {
				Texture   *globaltype.SDL_Texture
				Transform obj.Transform
			}{
				Texture:   texture,
				Transform: transform,
			})
		}
		g.Graphic.AddScreenTextureRenderBuffer(
			bps,
			data[0].Texture,
			data[0].Transform,
		)
		g.Graphic.RegisterTypingFX(data, name, float64(utf8.RuneCountInString(*t.Text))/g.TextSpeed, bps, g.Graphic.GetCurrentTopScreenIndexByBps(bps))
		return
	}

	texture, width, height := g.Graphic.GetTextTexture(*t.Text, t.FontName, t.Color)
	g.Graphic.RegisterTextMemPool(name, texture)
	t.T = g.echoTransform(t.T, width, height)

	g.Graphic.AddScreenTextureRenderBuffer(
		bps,
		texture,
		t.T,
	)
}

func (g *Game) evalButton(b *obj.Button, name string, bps int) {
	if b.T.Size.X != 0 && b.T.Size.Y != 0 {
		b.T = g.echoTransform(b.T, b.T.Size.X, b.T.Size.Y)
	} else {
		b.T = g.echoTransform(b.T, g.Graphic.Image.GetImageWidth(b.MainImageName), g.Graphic.Image.GetImageHeight(b.MainImageName))
	}

	g.Graphic.AddScreenTextureRenderBuffer(
		bps,
		g.Graphic.Image.GetImageTexture(b.MainImageName),
		b.T,
	)

	index := g.Graphic.GetCurrentTopScreenIndexByBps(bps)

	if len(b.Anime) != 0 {
		for _, anime := range b.Anime {
			switch anime.Type {
			case obj.ANIME_ALPHA:
				g.Graphic.Image.SetImageAlpha(b.MainImageName, int(anime.InitValue))
				g.Graphic.AddAnimation(name, anime, bps, g.Graphic.GetCurrentTopScreenIndexByBps(bps))
			case obj.ANIME_ROTATE:
				g.Graphic.SetRotateByBps(bps, g.Graphic.GetCurrentTopScreenIndexByBps(bps), int(anime.InitValue))
				g.Graphic.AddAnimation(name, anime, bps, g.Graphic.GetCurrentTopScreenIndexByBps(bps))
			}
		}
	}

	if b.HoverImageName != "" {
		if b.Action != nil {
			g.Event.AddButtonEvent(name, event.ButtonEvent{
				Action: func(e *event.EVENT_MouseButton) {
					var xpos, ypos int = g.Graphic.GetCurrentTexturePosition(bps, index)
					var xsize, ysize int = g.Graphic.GetCurrentTextureSize(bps, index)
					if e.X > xpos &&
						e.Y > ypos &&
						e.X < xpos+xsize &&
						e.Y < ypos+ysize &&
						e.Button == event.SDL_BUTTON_LEFT {
						b.Action()
					}
				},
				Hover: func(e *event.EVENT_MouseMotion) {
					var xpos, ypos int = g.Graphic.GetCurrentTexturePosition(bps, index)
					var xsize, ysize int = g.Graphic.GetCurrentTextureSize(bps, index)
					if e.X > xpos &&
						e.Y > ypos &&
						e.X < xpos+xsize &&
						e.Y < ypos+ysize {
						g.Graphic.ChangeTextureByBps(g.screenBps[name], index, b.HoverImageName)
					} else {
						g.Graphic.ChangeTextureByBps(g.screenBps[name], index, b.MainImageName)
					}
				},
			})
		}
	} else {
		if b.Action != nil {
			g.Event.AddButtonEvent(name, event.ButtonEvent{
				Action: func(e *event.EVENT_MouseButton) {
					var xpos, ypos int = g.Graphic.GetCurrentTexturePosition(bps, index)
					var xsize, ysize int = g.Graphic.GetCurrentTextureSize(bps, index)
					if e.X > xpos &&
						e.Y > ypos &&
						e.X < xpos+xsize &&
						e.Y < ypos+ysize {
						b.Action()
					}
				},
				Hover: func(e *event.EVENT_MouseMotion) {},
			})
		}
	}
}

func (g *Game) echoTransform(t obj.Transform, width, height int) obj.Transform {
	if t.Type != nil {
		switch b := (t.Type).(type) {
		case *obj.Center:
			t.Pos = obj.Vector2{
				X: (g.width - width) / 2,
				Y: (g.height - height) / 2,
			}
		case *obj.XCenter:
			t.Pos = obj.Vector2{
				X: (g.width - width) / 2,
				Y: b.Ypos,
			}
		case *obj.YCenter:
			t.Pos = obj.Vector2{
				X: b.Xpos,
				Y: (g.height - height) / 2,
			}
		case *obj.AxisCenter:
			t.Pos = obj.Vector2{
				X: b.Axis.X - width/2,
				Y: b.Axis.Y - height/2,
			}
		}
	}
	t.Size.X = width
	t.Size.Y = height

	return t
}
