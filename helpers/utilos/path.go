package utilos

func PathSuffix(path string, slashCount int) (suffix string) {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			slashCount--
			if slashCount == 0 {
				return path[i+1:]
			}
		}
	}
	return path
}
