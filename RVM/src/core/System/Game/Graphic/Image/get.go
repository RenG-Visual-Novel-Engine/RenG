package image

import "RenG/RVM/src/core/globaltype"

func (i *Image) GetImageTexture(name string) *globaltype.SDL_Texture {
	i.lock.Lock()
	defer i.lock.Unlock()

	if image, ok := i.images[name]; !ok {
		return nil
	} else {
		return image.texture
	}
}

func (i *Image) GetImageWidth(name string) int {
	i.lock.Lock()
	defer i.lock.Unlock()

	if image, ok := i.images[name]; !ok {
		return 0
	} else {
		return image.width
	}
}

func (i *Image) GetImageHeight(name string) int {
	i.lock.Lock()
	defer i.lock.Unlock()

	if image, ok := i.images[name]; !ok {
		return 0
	} else {
		return image.height
	}
}
