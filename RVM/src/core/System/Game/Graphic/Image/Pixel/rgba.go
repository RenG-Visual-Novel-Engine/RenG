package pixel

const (
	RENG_PIXELFORMAT_RGBA8888 = iota
)

type RGBA interface {
	SetRGBA(RGBA *uint32, size int64)
	SetWH(Width, Height int64)

	GetPixels() (data *uint32, size int64)
	GetWH() (Width, Height int64)
}

type RGBA8888 struct {
	Width  int64
	Height int64

	Pixels *uint32
	Pitch  int64
}

func (rgba *RGBA8888) SetRGBA(RGBA *uint32, size int64) {
	rgba.Pixels = RGBA
	rgba.Pitch = size
}

func (rgba *RGBA8888) SetWH(Width, Height int64) {
	rgba.Width = Width
	rgba.Height = Height
}

func (rgba *RGBA8888) GetPixels() (data *uint32, size int64) {
	return rgba.Pixels, rgba.Pitch
}

func (rgba *RGBA8888) GetWH() (Width, Height int64) {
	return rgba.Width, rgba.Height
}

func NewRGBA(PixelFormatType int64) RGBA {
	switch PixelFormatType {
	case RENG_PIXELFORMAT_RGBA8888:
		return &RGBA8888{}
	default:
		return nil
	}
}
