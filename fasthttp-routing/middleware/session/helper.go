package session

import "io"

func Helper(r io.Reader, b []byte) (int, error) {
	return r.Read(b)
}
