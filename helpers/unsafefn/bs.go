package unsafefn

import (
	"hash"
	"io"
	"unsafe"
)

func BtoS(bs []byte) string {
	return unsafe.String(unsafe.SliceData(bs), len(bs))
}

func StoB(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

//go:linkname NoescapeRead 	helpers/unsafefn.noescapeRead
//go:noescape
func NoescapeRead(r io.Reader, b []byte) (int, error)

func noescapeRead(r io.Reader, b []byte) (int, error) {
	return r.Read(b)
}

//go:linkname NoescapeSum  helpers/unsafefn.noescapeSum
//go:noescape
func NoescapeSum(h hash.Hash, b []byte) []byte
func noescapeSum(h hash.Hash, b []byte) []byte {
	return h.Sum(b)
}
