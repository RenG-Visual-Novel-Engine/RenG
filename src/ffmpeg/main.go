package main

// #cgo CFLAGS: -I./include
// #cgo LDFLAGS: -L./lib -lavcodec -lavformat -lavutil
//
// #include <libavcodec/avcodec.h>
// #include <libavformat/avformat.h>
// #include <libavutil/avutil.h>
import "C"
import "fmt"

func main() {
	var format *C.AVFormatContext
	root := C.CString("D:\\video\\ball.mp4")
	if int(C.avformat_open_input(&format, root, nil, nil)) != 0 {
		fmt.Println("Error")
	} else {
		fmt.Println("Success")
	}
}
