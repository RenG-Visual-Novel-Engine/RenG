package storage

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"os"
)

func (s *Storage) SetIntValue(key string, value int) {
	if offset, ok := s.IsKeyValid(key); !ok {
		hash := sha256.New()
		hash.Write([]byte(key))
		s.f.Write(hash.Sum(nil))

		size := make([]byte, 4)
		binary.LittleEndian.PutUint32(size, 8)
		s.f.Write(size)

		v := make([]byte, 8)
		binary.LittleEndian.PutUint64(v, uint64(value))
		s.f.Write(v)
	} else {
		v := make([]byte, 8)
		binary.LittleEndian.PutUint64(v, uint64(value))
		s.f.WriteAt(v, int64(offset+36))
	}
}

func (s *Storage) SetStringValue(key, value string) {
	if offset, ok := s.IsKeyValid(key); !ok {
		hash := sha256.New()
		hash.Write([]byte(key))

		s.f.Seek(0, io.SeekEnd)
		s.f.Write(hash.Sum(nil))

		size := make([]byte, 4)
		binary.LittleEndian.PutUint32(size, uint32(len(value)))
		s.f.Write(size)

		_, err := s.f.WriteString(value)
		if err != nil {
			panic(err)
		}
	} else {
		f, err := os.Create("temp")
		if err != nil {
			panic(err)
		}

		hash := make([]byte, 32)
		s.f.ReadAt(hash, int64(offset))
		f.Write(hash)

		size := make([]byte, 4)
		binary.LittleEndian.PutUint32(size, uint32(len(value)))
		f.Write(size)

		f.WriteString(value)

		oldSize := make([]byte, 4)
		s.f.ReadAt(oldSize, int64(offset+32))

		temp1 := make([]byte, offset)
		s.f.ReadAt(temp1, 0)

		f.Write(temp1)

		s.f.Seek(int64(offset+36+int(binary.LittleEndian.Uint32(oldSize))), io.SeekStart)

		temp2 := []byte{}
		for {
			b := make([]byte, 1)
			_, err := s.f.Read(b)
			if errors.Is(err, io.EOF) {
				break
			}
			temp2 = append(temp2, b[0])
		}
		f.Write(temp2)

		f.Close()
		s.f.Close()

		os.Remove(s.path)
		err = os.Rename("temp", s.path)
		if err != nil {
			panic(err)
		}

		s.f, err = os.OpenFile(s.path, os.O_RDWR|os.O_CREATE, os.FileMode(0644))
		if err != nil {
			panic(err)
		}
	}
}
