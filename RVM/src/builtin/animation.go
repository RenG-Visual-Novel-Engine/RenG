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
	Quart   func(Start, End int) func(t float64) int
	Quint   func(Start, End int) func(t float64) int
	Expo    func(Start, End int) func(t float64) int
	Circ    func(Start, End int) func(t float64) int
	Back    func(Start, End int) func(t float64) int
	Elastic func(Start, End int) func(t float64) int
	Bounce  func(Start, End int) func(t float64) int
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

func easeInQuart(Start, End int) func(t float64) int {
	return func(t float64) int {
		return int((t*t*t*t)*float64(End-Start) + float64(Start))
	}
}
func easeOutQuart(Start, End int) func(t float64) int {
	return func(t float64) int {
		return int((1-math.Pow(1-t, 4))*float64(End-Start) + float64(Start))
	}
}
func easeInOutQuart(Start, End int) func(t float64) int {
	return func(t float64) int {
		if t < 0.5 {
			return int((8*t*t*t*t)*float64(End-Start) + float64(Start))
		} else {
			return int((1-math.Pow(-2*t+2, 4)/2)*float64(End-Start) + float64(Start))
		}
	}
}

func easeInQuint(Start, End int) func(t float64) int {
	return func(t float64) int {
		return int((t*t*t*t*t)*float64(End-Start) + float64(Start))
	}
}

func easeOutQuint(Start, End int) func(t float64) int {
	return func(t float64) int {
		return int((1-math.Pow(1-t, 5))*float64(End-Start) + float64(Start))
	}
}
func easeInOutQuint(Start, End int) func(t float64) int {
	return func(t float64) int {
		if t < 0.5 {
			return int((16*t*t*t*t*t)*float64(End-Start) + float64(Start))

		} else {

			return int((1-math.Pow(-2*t+2, 5)/2)*float64(End-Start) + float64(Start))
		}
	}
}

func easeInExpo(Start, End int) func(t float64) int {
	return func(t float64) int {
		if t == 0 {
			return Start

		} else {

			return int(math.Pow(2, 10*t-10)*float64(End-Start) + float64(Start))
		}
	}
}

func easeOutExpo(Start, End int) func(t float64) int {
	return func(t float64) int {
		if t == 1 {
			return End
		} else {

			return int((1-math.Pow(2, -10*t))*float64(End-Start) + float64(Start))
		}
	}
}

func easeInOutExpo(Start, End int) func(t float64) int {
	return func(t float64) int {
		if t == 0 {
			return Start
		} else if t == 1 {
			return End
		} else if t < 0.5 {
			return int((math.Pow(2, 20*t-10)/2)*float64(End-Start) + float64(Start))
		} else {
			return int(((2-math.Pow(2, -20*t+10))/2)*float64(End-Start) + float64(Start))
		}
	}
}

func easeInCirc(Start, End int) func(t float64) int {
	return func(t float64) int {
		return int((1-math.Sqrt(1-math.Pow(t, 2)))*float64(End-Start) + float64(Start))
	}
}

func easeOutCirc(Start, End int) func(t float64) int {
	return func(t float64) int {
		return int((math.Sqrt(1-math.Pow(t-1, 2)))*float64(End-Start) + float64(Start))
	}
}

func easeInOutCirc(Start, End int) func(t float64) int {
	return func(t float64) int {
		if t < 0.5 {
			return int(((1-math.Sqrt(1-math.Pow(2*t, 2)))/2)*float64(End-Start) + float64(Start))
		} else {
			return int(((math.Sqrt(1-math.Pow(-2*t+2, 2))+1)/2)*float64(End-Start) + float64(Start))
		}
	}
}
func easeInBack(Start, End int) func(t float64) int {
	return func(t float64) int {
		const c1 float64 = 1.70158
		const c3 = c1 + 1

		return int((c3*t*t*t-c1*t*t)*float64(End-Start) + float64(Start))
	}
}

func easeOutBack(Start, End int) func(t float64) int {
	return func(t float64) int {
		const c1 float64 = 1.70158
		const c3 = c1 + 1

		return int((1+c3*math.Pow(t-1, 3)+c1*math.Pow(t-1, 2))*float64(End-Start) + float64(Start))
	}
}

func easeInOutBack(Start, End int) func(t float64) int {
	return func(t float64) int {
		const c1 float64 = 1.70158
		const c2 = c1 * 1.525

		if t < 0.5 {
			return int(((math.Pow(2*t, 2)*((c2+1)*2*t-c2))/2)*float64(End-Start) + float64(Start))
		} else {
			return int(((math.Pow(2*t-2, 2)*((c2+1)*(t*2-2)+c2)+2)/2)*float64(End-Start) + float64(Start))
		}
	}
}

func easeInElastic(Start, End int) func(t float64) int {
	return func(t float64) int {
		const c4 float64 = (2 * math.Pi) / 3

		if t == 0 {
			return Start
		} else if t == 1 {
			return End
		} else {
			return int((-math.Pow(2, 10*t-10)*math.Sin((t*10-10.75)*c4))*float64(End-Start) + float64(Start))
		}
	}
}

func easeOutElastic(Start, End int) func(t float64) int {
	return func(t float64) int {
		const c4 float64 = (2 * math.Pi) / 3

		if t == 0 {
			return Start
		} else if t == 1 {
			return End
		} else {
			return int((math.Pow(2, -10*t)*math.Sin((t*10-0.75)*c4)+1)*float64(End-Start) + float64(Start))
		}
	}
}

func easeInOutElastic(Start, End int) func(t float64) int {
	return func(t float64) int {
		const c5 float64 = (2 * math.Pi) / 4.5

		if t == 0 {
			return Start
		} else if t == 1 {
			return End
		} else if t < 0.5 {
			return int((-(math.Pow(2, 20*t-10)*math.Sin((20*t-11.125)*c5))/2)*float64(End-Start) + float64(Start))
		} else {
			return int(((math.Pow(2, -20*t+10)*math.Sin((20*t-11.125)*c5))/2+1)*float64(End-Start) + float64(Start))
		}
	}
}

func easeInBounce(Start, End int) func(t float64) int {
	return func(t float64) int {
		return int((1-easeOutBouncePrivate(1-t))*float64(End-Start) + float64(Start))
	}
}

func easeOutBounce(Start, End int) func(t float64) int {
	return func(t float64) int {
		return int(easeOutBouncePrivate(t)*float64(End-Start) + float64(Start))
	}
}

func easeOutBouncePrivate(t float64) float64 {
	if t < 4/11.0 {
		return (121 * t * t) / 16.0
	} else if t < 8/11.0 {
		return (363 / 40.0 * t * t) - (99 / 10.0 * t) + 17/5.0
	} else if t < 9/10.0 {
		return (4356 / 361.0 * t * t) - (35442 / 1805.0 * t) + 16061/1805.0
	} else {
		return (54 / 5.0 * t * t) - (513 / 25.0 * t) + 268/25.0
	}
}

func easeInOutBounce(Start, End int) func(t float64) int {
	return func(t float64) int {
		if t < 0.5 {
			return int(((1-easeOutBouncePrivate(1-2*t))/2)*float64(End-Start) + float64(Start))
		} else {
			return int(((1+easeOutBouncePrivate(2*t-1))/2)*float64(End-Start) + float64(Start))
		}
	}
}
