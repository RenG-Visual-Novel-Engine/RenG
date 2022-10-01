package util

func ErrorCheck(err error) {
	if err != nil {
		panic(err)
	}
}
