package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
)

// Start Point
//     역할
//      1.실행시 메인 게임 엔진 메인화면 코드가 있는 파일 주소를 인터프리터한테 넘김
//      2.파일이 제대로 작동할 준비가 되었는지 확인함
//
func main() {
	cmd := exec.Command("cmd")
	if runtime.GOOS == "windows" {
		root, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		cmd = exec.Command("core\\RenG", fmt.Sprintf("-r=%s\\RenGLauncher", root))
	}
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}
