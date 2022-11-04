package str

func SliceToken(input, tok string) string {
	var start int
	var ret string

	for i := 0; i < len(input); i++ {
		for input[i:i+len(tok)] != tok && i+len(tok) < len(input)-1 {
			i++
		}
		if i+len(tok) >= len(input)-1 {
			break
		}

		start = i

		for input[i] != '{' {
			i++
		}

		cnt := 1

		for {
			i++
			if input[i] == '{' {
				cnt++
			} else if input[i] == '}' {
				cnt--
			}
			if cnt == 0 {
				break
			}
		}

		ret += input[start:i+1] + "\n"
	}

	return ret
}
