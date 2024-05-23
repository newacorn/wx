package session

import (
	"bytes"
	"encoding/gob"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/valyala/bytebufferpool"
)

// Codec is the interface for encoding/decoding session data to and from a byte
// slice for use by the session store.
type Codec interface {
	Encode(csrfToken []byte, values map[string]interface{}) ([]byte, error)
	Decode(encodedData []byte, dstCsrfToken []byte) (csrfToken []byte, values map[string]interface{}, err error)
}

// GobCodec is used for encoding/decoding session data to and from a byte
// slice using the encoding/gob package.
type GobCodec struct{}

// Encode converts a session deadline and values into a byte slice.
func (GobCodec) Encode(deadline time.Time, values map[string]interface{}) ([]byte, *bytebufferpool.ByteBuffer, error) {
	aux := &struct {
		Deadline time.Time
		Values   map[string]interface{}
	}{
		Deadline: deadline,
		Values:   values,
	}
	b := bytebufferpool.Get()

	// var b bytes.Buffer
	if err := gob.NewEncoder(b).Encode(&aux); err != nil {
		err = NewDataError("codecEncode:" + err.Error())
		log.Error().Str("Error", err.Error()).Send()
		return nil, b, err
	}

	return b.Bytes(), b, nil
}

// Decode converts a byte slice into a session deadline and values.
func (GobCodec) Decode(b []byte) (time.Time, map[string]interface{}, error) {
	aux := &struct {
		Deadline time.Time
		Values   map[string]interface{}
	}{}

	r := bytes.NewReader(b)
	if err := gob.NewDecoder(r).Decode(&aux); err != nil {
		err = NewDataError("decode session data: " + err.Error())
		log.Error().Str("Err", err.Error()).Send()
		return time.Time{}, nil, err
	}

	return aux.Deadline, aux.Values, nil
}
