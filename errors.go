package mapstructure

import (
	"fmt"
	"reflect"
)

// ErrCannotDecode is a generic error type that holds information about
// a decoding error together with the name of the field that caused the error.
type ErrCannotDecode struct {
	Name string
	Err  error
}

func (e *ErrCannotDecode) Error() string {
	return fmt.Sprintf("'%s': %s", e.Name, e.Err)
}

// ErrCannotParse extends ErrCannotDecode to include additional information
// about the expected type and the actual value that could not be parsed.
type ErrCannotParse struct {
	ErrCannotDecode
	Expected reflect.Value
	Value    interface{}
}

func (e *ErrCannotParse) Error() string {
	return fmt.Sprintf("'%s' cannot parse '%s' as '%s': %s",
		e.Name, e.Value, e.Expected.Type(), e.Err)
}

// ErrUnconvertibleType is an error type that indicates a value could not be
// converted to the expected type. It includes the name of the field, the
// expected type, and the actual value that was attempted to be converted.
type ErrUnconvertibleType struct {
	Name     string
	Expected reflect.Value
	Value    interface{}
}

func (e *ErrUnconvertibleType) Error() string {
	return fmt.Sprintf("'%s' expected type '%s', got unconvertible type '%s', value: '%v'",
		e.Name, e.Expected.Type(), reflect.TypeOf(e.Value), e.Value)
}
