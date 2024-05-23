package utilconvert

func ToLower(s []byte) {
	for i := 0; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			s[i] += 'a' - 'A'
		}
	}
}
