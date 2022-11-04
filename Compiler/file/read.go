package file

func (f *File) Read() string {
	var context string

	for {
		b := make([]byte, 1)
		n, err := f.file.Read(b)
		if n == 0 {
			break
		}
		if err != nil {
			panic(err)
		}

		context += string(b)
	}

	return context
}
