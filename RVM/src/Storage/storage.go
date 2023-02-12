package storage

import "os"

type Storage struct {
	path string
	f    *os.File
}

/*
저장소를 불러옵니다.
만약 존재하지 않다면 새로 생성하여 불러옵니다.
*/
func LoadStorage(path string) *Storage {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(0666))
	if err != nil {
		panic(err)
	}

	return &Storage{
		path: path,
		f:    file,
	}
}

/*

 */
func (s *Storage) CloseStorage() {
	s.f.Close()
}
