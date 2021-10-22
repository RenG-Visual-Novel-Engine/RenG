package ffmpeg

import "C"

func CInt(num int) C.int {
	return C.int(num)
}
