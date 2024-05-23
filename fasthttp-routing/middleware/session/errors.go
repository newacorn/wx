package session

import (
	"reflect"
)

type DataError struct {
	s string
}
type StoreError struct {
	s string
}
type TokenError struct {
	s string
}
type EncodingError struct {
	s string
}

func NewEncodingError(err string) *EncodingError {
	return &EncodingError{s: err}
}
func NewTokenError(err string) *TokenError {
	return &TokenError{s: err}
}
func (t *EncodingError) Error() string {
	return reflect.TypeOf(t).String() + t.s
}
func (t *TokenError) Error() string {
	return reflect.TypeOf(t).String() + t.s
}
func NewDataError(err string) *DataError {
	return &DataError{s: err}
}
func (d *DataError) Error() string {
	return reflect.TypeOf(d).String() + d.s
}
func (s *StoreError) Error() string {
	return reflect.TypeOf(s).String() + s.s
}
func NewStoreError(err string) *StoreError {
	return &StoreError{s: err}
}
