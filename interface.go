package jspointer

import (
	"errors"
	"reflect"
)

var ErrInvalidPointer = errors.New("invalid pointer")
var ErrNotFound = errors.New("match to JSON pointer not found")
var ErrCanNotSet = errors.New("field cannot be set to")
var ErrSliceIndexOutOfBounds = errors.New("slice index out of bounds")

const (
	EncodedTilde = "~0"
	EncodedSlash = "~1"
	Separator    = '/'
)

type JSPointer struct {
	tokens []string
}

type Result struct {
	Item interface{}
	Kind reflect.Kind
}
