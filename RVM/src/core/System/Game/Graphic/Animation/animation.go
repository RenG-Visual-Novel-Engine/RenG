package animation

import "time"

type Anime struct {
	Loop      bool
	Type      int
	InitValue float64
	StartTime float64
	Duration  float64
	Curve     func(t float64) int
	Time      time.Time
	End       func()
}

const (
	ANIME_ALPHA = iota
	ANIME_ROTATE
)
