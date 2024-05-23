package utilcrypt

import (
	"encoding/binary"
	"math/rand"
	"unsafe"
)

type RandomBytesGenerator func(bs []byte)

func DefaultRandomBytesGenerator(bs []byte) {
	size := len(bs)
	for i := 0; i < size/8; i++ {
		r1 := rand.Uint64()
		b1 := (*[8]byte)((unsafe.Pointer)(&r1))
		copy(bs[i*8:], b1[:])
	}
	if size%8 > 0 {
		r2 := rand.Uint64()
		b2 := (*[8]byte)((unsafe.Pointer)(&r2))
		copy(bs[:size-size%8], b2[:])
	}
}

type BytesWithHash interface {
	AppendHash(bs []byte)
	ValidateHash(bs []byte) bool
	EncodedLen(input int) (output int)
	DecodedLen(input int) (output int)
}
type SimpleHash struct {
}

func (s SimpleHash) AppendHash(bs []byte) {
	if len(bs) < 5 {
		panic("len(bs) at least 5.")
	}
	sum := (uint16(bs[0])*uint16(bs[1]) + 99) * uint16(bs[2])
	binary.LittleEndian.AppendUint16(bs[:len(bs)-2], sum)
}
func (s SimpleHash) ValidateHash(bs []byte) bool {
	if len(bs) < 3 {
		return false
	}
	sum := (uint16(bs[0])*uint16(bs[1]) + 99) * uint16(bs[2])
	sum2 := binary.LittleEndian.Uint16(bs[len(bs)-2:])
	return sum == sum2
}
func (s SimpleHash) EncodedLen(input int) (output int) {
	return input + 2
}
func (s SimpleHash) DecodedLen(input int) (output int) {
	return input - 2
}

//go:linkname NoescapeAppendHash helpers/utilcrypt.noescapeAppendHash
//go:noescape
func NoescapeAppendHash(appendHash BytesWithHash, bs []byte)
func noescapeAppendHash(appendHash BytesWithHash, bs []byte) {
	appendHash.AppendHash(bs)
}

//go:linkname NoescapeValidHash helpers/utilcrypt.noescapeValidHash
//go:noescape
func NoescapeValidHash(appendHash BytesWithHash, bs []byte) bool
func noescapeValidHash(appendHash BytesWithHash, bs []byte) bool {
	return appendHash.ValidateHash(bs)
}
