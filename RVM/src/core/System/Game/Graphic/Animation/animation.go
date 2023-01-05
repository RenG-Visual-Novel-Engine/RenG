package animation

type Anime struct {
	Loop      bool
	Type      int
	InitValue float64
	StartTime float64
	Duration  float64
	Curve     func(t float64) int
}

const (
	ANIME_ALPHA = iota
)
