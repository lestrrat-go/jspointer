package jspointer

import (
	"errors"
	"reflect"
)

// Errors used in jspointer package
var (
	ErrInvalidPointer        = errors.New("invalid pointer")
	ErrNotFound              = errors.New("match to JSON pointer not found")
	ErrCanNotSet             = errors.New("field cannot be set to")
	ErrSliceIndexOutOfBounds = errors.New("slice index out of bounds")
)

// Consntants used in jspointer package. Mostly for internal usage only
const (
	EncodedTilde = "~0"
	EncodedSlash = "~1"
	Separator    = '/'
)

// JSPointer represents a JSON pointer
type JSPointer struct {
	tokens []string
}

// Result represents the result of evaluating a JSON pointer
type Result struct {
	Item interface{}
	Kind reflect.Kind
}
