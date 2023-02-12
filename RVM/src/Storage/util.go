package storage

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
)

func (s *Storage) IsKeyValid(key string) (offset int, valid bool) {
	hash := sha256.New()
	hash.Write([]byte(key))
	hashKey := hash.Sum(nil)

	of := 0
	for {
		getKeyValue := make([]byte, 32)
		_, err := s.f.ReadAt(getKeyValue, int64(of))
		if errors.Is(err, io.EOF) {
			return -1, false
		}

		if string(hashKey) == string(getKeyValue) {
			break
		}
		size := make([]byte, 4)
		s.f.ReadAt(size, int64(of+32))
		of += int(binary.LittleEndian.Uint32(size)) + 36
	}
	return of, true
}
