package storage

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
)

func (s *Storage) GetIntValue(key string) int {
	hash := sha256.New()
	hash.Write([]byte(key))
	hashKey := hash.Sum(nil)

	offset := 0
	for {
		getKeyValue := make([]byte, 32)
		_, err := s.f.ReadAt(getKeyValue, int64(offset))
		if errors.Is(err, io.EOF) {
			return -1
		}

		if string(hashKey) == string(getKeyValue) {
			break
		}
		size := make([]byte, 4)
		s.f.ReadAt(size, int64(offset+32))
		offset += int(binary.LittleEndian.Uint32(size)) + 36
	}

	size := make([]byte, 4)
	s.f.ReadAt(size, int64(offset+32))

	value := make([]byte, int(binary.LittleEndian.Uint32(size)))
	s.f.ReadAt(value, int64(offset+36))

	return int(binary.LittleEndian.Uint64(value))
}

func (s *Storage) GetStringValue(key string) string {
	hash := sha256.New()
	hash.Write([]byte(key))
	hashKey := hash.Sum(nil)

	offset := 0
	for {
		getKeyValue := make([]byte, 32)
		_, err := s.f.ReadAt(getKeyValue, int64(offset))
		if errors.Is(err, io.EOF) {
			return ""
		}

		if string(hashKey) == string(getKeyValue) {
			break
		}
		size := make([]byte, 4)
		s.f.ReadAt(size, int64(offset+32))
		offset += int(binary.LittleEndian.Uint32(size)) + 36
	}

	size := make([]byte, 4)
	s.f.ReadAt(size, int64(offset+32))

	value := make([]byte, int(binary.LittleEndian.Uint32(size)))
	s.f.ReadAt(value, int64(offset+36))

	return string(value)
}
