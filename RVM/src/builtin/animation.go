package builtin

import "math"

var (
	Anime = &AnimeCurve{
		Ease: Ease{
			In: EasingStyle{
				Sine:    easeInSine,
				Quad:    easeInQuad,
				Cubic:   easeInCubic,
				Quart:   easeInQuart,
				Quint:   easeInQuint,
				Expo:    easeInExpo,
				Circ:    easeInCirc,
				Back:    easeInBack,
				Elastic: easeInElastic,
				Bounce:  easeInBounce,
			},
			Out: EasingStyle{
				Sine:    easeOutSine,
				Quad:    easeOutQuad,
				Cubic:   easeOutCubic,
				Quart:   easeOutQuart,
				Quint:   easeOutQuint,
				Expo:    easeOutExpo,
				Circ:    easeOutCirc,
				Back:    easeOutBack,
				Elastic: easeOutElastic,
				Bounce:  easeOutBounce,
			},
			InOut: EasingStyle{
				Sine:    easeInOutSine,
				Quad:    easeInOutQuad,
				Cubic:   easeInOutCubic,
				Quart:   easeInOutQuart,
				Quint:   easeInOutQuint,
				Expo:    easeInOutExpo,
				Circ:    easeInOutCirc,
				Back:    easeInOutBack,
				Elastic: easeInOutElastic,
				Bounce:  easeInOutBounce,
			},
		},
	}
)

type AnimeCurve struct {
	Ease Ease
}

type Ease struct {
	In    EasingStyle
	Out   EasingStyle
	InOut EasingStyle
}

type EasingStyle struct {
	Sine    func(Start, End int) func(t float64) int
	Quad    func(Start, End int) func(t float64) int
	Cubic   func(Start, End int) func(t float64) int
	Quart   func(float64) float64
	Quint   func(float64) float64
	Expo    func(float64) float64
	Circ    func(float64) float64
	Back    func(float64) float64
	Elastic func(float64) float64
	Bounce  func(float64) float64
}

func easeInSine(Start, End int) func(t float64) int {
	return func(t float64) int {
		return int((1-math.Cos((t*math.Pi)/2))*float64(End-Start) + float64(Start))
	}
}

func easeOutSine(Start, End int) func(t float64) int {
	return func(t float64) int {
		return int(math.Sin((t*math.Pi)/2)*float64(End-Start) + float64(Start))
	}
}

func easeInOutSine(Start, End int) func(t float64) int {
	return func(t float64) int {
		return int(-(math.Cos(math.Pi*t)-1)/2*float64(End-Start) + float64(Start))
	}
}

func easeInQuad(Start, End int) func(t float64) int {
	return func(t float64) int {
		return int(t*t*float64(End-Start) + float64(Start))
	}
}

func easeOutQuad(Start, End int) func(t float64) int {
	return func(t float64) int {
		return int((1-(1-t)*(1-t))*float64(End-Start) + float64(Start))
	}
}

func easeInOutQuad(Start, End int) func(t float64) int {
	return func(t float64) int {
		if t < 0.5 {
			return int((2*t*t)*float64(End-Start) + float64(Start))
		} else {
			return int((1-math.Pow(-2*t+2, 2)/2)*float64(End-Start) + float64(Start))
		}
	}
}
func easeInCubic(Start, End int) func(t float64) int {
	return func(t float64) int {
		return int(t*t*t*float64(End-Start) + float64(Start))
	}
}

func easeOutCubic(Start, End int) func(t float64) int {
	return func(t float64) int {
		return int((1-math.Pow(1-t, 3))*float64(End-Start) + float64(Start))
	}
}

func easeInOutCubic(Start, End int) func(t float64) int {
	return func(t float64) int {
		if t < 0.5 {
			return int((4*t*t*t)*float64(End-Start) + float64(Start))
		} else {
			return int((1-math.Pow(-2*t+2, 3)/2)*float64(End-Start) + float64(Start))
		}
	}
}

func easeInQuart(t float64) float64 {
	return t * t * t * t
}

func easeOutQuart(t float64) float64 {
	return 1 - math.Pow(1-t, 4)
}

func easeInOutQuart(t float64) float64 {
	if t < 0.5 {
		return 8 * t * t * t * t
	} else {
		return 1 - math.Pow(-2*t+2, 4)/2
	}
}

func easeInQuint(t float64) float64 {
	return t * t * t * t * t
}

func easeOutQuint(t float64) float64 {
	return 1 - math.Pow(1-t, 5)
}

func easeInOutQuint(t float64) float64 {
	if t < 0.5 {
		return 16 * t * t * t * t * t

	} else {

		return 1 - math.Pow(-2*t+2, 5)/2
	}
}

func easeInExpo(t float64) float64 {
	if t == 0 {
		return 0

	} else {

		return math.Pow(2, 10*t-10)
	}
}

func easeOutExpo(t float64) float64 {
	if t == 1 {
		return 1
	} else {

		return 1 - math.Pow(2, -10*t)
	}
}

func easeInOutExpo(t float64) float64 {
	if t == 0 {
		return 0
	} else if t == 1 {
		return 1
	} else if t < 0.5 {
		return math.Pow(2, 20*t-10) / 2
	} else {
		return (2 - math.Pow(2, -20*t+10)) / 2
	}
}

func easeInCirc(t float64) float64 {
	return 1 - math.Sqrt(1-math.Pow(t, 2))
}

func easeOutCirc(t float64) float64 {
	return math.Sqrt(1 - math.Pow(t-1, 2))
}

func easeInOutCirc(t float64) float64 {
	if t < 0.5 {
		return (1 - math.Sqrt(1-math.Pow(2*t, 2))) / 2
	} else {
		return (math.Sqrt(1-math.Pow(-2*t+2, 2)) + 1) / 2
	}
}

func easeInBack(t float64) float64 {
	const c1 float64 = 1.70158
	const c3 = c1 + 1

	return c3*t*t*t - c1*t*t
}

func easeOutBack(t float64) float64 {
	const c1 float64 = 1.70158
	const c3 = c1 + 1

	return 1 + c3*math.Pow(t-1, 3) + c1*math.Pow(t-1, 2)
}

func easeInOutBack(t float64) float64 {
	const c1 float64 = 1.70158
	const c2 = c1 * 1.525

	if t < 0.5 {
		return (math.Pow(2*t, 2) * ((c2+1)*2*t - c2)) / 2
	} else {
		return (math.Pow(2*t-2, 2)*((c2+1)*(t*2-2)+c2) + 2) / 2
	}
}

func easeInElastic(t float64) float64 {
	const c4 float64 = (2 * math.Pi) / 3

	if t == 0 {
		return 0
	} else if t == 1 {
		return 1
	} else {
		return -math.Pow(2, 10*t-10) * math.Sin((t*10-10.75)*c4)
	}
}

func easeOutElastic(t float64) float64 {
	const c4 float64 = (2 * math.Pi) / 3

	if t == 0 {
		return 0
	} else if t == 1 {
		return 1
	} else {
		return math.Pow(2, -10*t)*math.Sin((t*10-0.75)*c4) + 1
	}
}
func easeInOutElastic(t float64) float64 {
	const c5 float64 = (2 * math.Pi) / 4.5

	if t == 0 {
		return 0
	} else if t == 1 {
		return 1
	} else if t < 0.5 {
		return -(math.Pow(2, 20*t-10) * math.Sin((20*t-11.125)*c5)) / 2
	} else {
		return (math.Pow(2, -20*t+10)*math.Sin((20*t-11.125)*c5))/2 + 1
	}
}

func easeInBounce(t float64) float64 {
	return 1 - easeOutBounce(1-t)
}

func easeOutBounce(t float64) float64 {
	const n1 float64 = 7.5625
	const d1 float64 = 2.75

	if t < 1/d1 {
		return n1 * t * t
	} else if t < 2/d1 {
		return (n1 * t) - (1.5 / d1 * t) + 0.75
	} else if t < 2.5/d1 {
		return (n1 * t) - (2.25 / d1 * t) + 0.9375
	} else {
		return (n1 * t) - (2.625 / d1 * t) + 0.984375
	}
}

func easeInOutBounce(t float64) float64 {
	if t < 0.5 {
		return (1 - easeOutBounce(1-2*t)) / 2
	} else {
		return (1 + easeOutBounce(2*t-1)) / 2
	}
}
