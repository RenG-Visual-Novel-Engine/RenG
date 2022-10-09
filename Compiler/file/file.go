package file

import (
	"RenG/Compiler/util"
	"os"
)

type File struct {
	path string
	file *os.File
}

func CreateFile(path string) *File {
	_, err := os.Stat(path)

	if os.IsNotExist(err) {
		file, err := os.Create(path)
		util.ErrorCheck(err)

		file.Close()
	}

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, os.FileMode(0644))
	util.ErrorCheck(err)

	return &File{
		path: path,
		file: file,
	}
}

func (f *File) CloseFile() {
	f.file.Close()
}
