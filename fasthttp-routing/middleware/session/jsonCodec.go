package session

import (
	"github.com/bytedance/sonic"
	"helpers/unsafefn"
)

type JSONCodec struct{}
type Aux struct {
	V  map[string]interface{}
	CR string
}

func (J JSONCodec) Encode(csrfToken []byte, values map[string]interface{}) (encodedData []byte, err error) {
	// TODO implement me
	encodedData, err = sonic.Marshal(&Aux{
		V:  values,
		CR: unsafefn.BtoS(csrfToken),
	})
	return
}

func (J JSONCodec) Decode(bytes []byte,
	dstCsrfToken []byte) (csrfToken []byte, values map[string]interface{}, err error) {
	aux := Aux{}
	err = sonic.Unmarshal(bytes, &aux)
	if err != nil {
		return
	}
	values = aux.V
	csrfToken = append(dstCsrfToken, aux.CR...)
	return
}
