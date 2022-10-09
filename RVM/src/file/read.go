package file

import "os"

func ReadRGOCDir(path string) ([]string, error) {
	var files []string

	f, err := os.Open(path)
	if err != nil {
		return files, err
	}

	fileInfo, err := f.ReadDir(-1)
	if err != nil {
		return files, err
	}

	f.Close()

	for _, file := range fileInfo {
		if file.Name()[len(file.Name())-4:] == "rgoc" {
			files = append(files, file.Name())
		}
	}

	return files, nil
}

func ReadRGOCFile()
