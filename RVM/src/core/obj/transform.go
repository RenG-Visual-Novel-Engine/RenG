package obj

type Transform struct {
	Pos    Vector2
	Size   Vector2
	Rotate int

	Type SpecialTransform
}

/*--------- 특수 transform ---------*/

type SpecialTransform interface {
	specialTransformObj()
}

type Center struct {
}

func (c *Center) specialTransformObj() {}

type XCenter struct {
	Ypos int
}

func (xc *XCenter) specialTransformObj() {}

type YCenter struct {
	Xpos int
}

func (yc *YCenter) specialTransformObj() {}

type AxisCenter struct {
	Axis Vector2
}

func (ac *AxisCenter) specialTransformObj() {}
