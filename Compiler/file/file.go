package file

import (
	"RenG/Compiler/code"
	"RenG/Compiler/object"
	"RenG/Compiler/util"
	"io/ioutil"
	"os"
	"strconv"
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

func (f *File) WriteConstant(os []object.Object) {
	f.WriteFileByte('C')
	for _, o := range os {
		switch o.Type() {
		case object.INTEGER_OBJ:
			f.WriteFileBytes([]byte(strconv.Itoa(int(o.(*object.Integer).Value))))
		}
	}
	f.WriteFileByte('E')
}

func (f *File) WriteInstruction(is code.Instructions, os []object.Object) {
	f.WriteFileByte('B')
	for ip := 0; ip < len(is); ip++ {
		op := code.Opcode(is[ip])

		switch op {
		case code.OpConstant:
			f.WriteFileByte(byte(op))
			switch os[code.ReadUint32(is[ip+1:])].Type() {
			case object.INTEGER_OBJ:
				f.WriteFileByte(0x04)
				f.WriteFileBytes(is[ip+1 : ip+5])
			}
			ip += 4
		default:
			f.WriteFileByte(byte(op))
		}
	}
	f.WriteFileByte('E')
}

func (f *File) WriteFileByte(b byte) {
	_, err := f.file.Write([]byte{b})
	util.ErrorCheck(err)
}

func (f *File) WriteFileBytes(bs []byte) {
	_, err := f.file.Write(bs)
	util.ErrorCheck(err)
}

func (f *File) Read() []byte {
	content, _ := ioutil.ReadFile(f.path)
	return content
}
