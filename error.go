package mapstructure

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// joinedError implements the error interface and can represents multiple
// errors that occur in the course of a single decode.
type joinedError struct {
	Errors []string
}

func (e *joinedError) Error() string {
	points := make([]string, len(e.Errors))
	for i, err := range e.Errors {
		points[i] = fmt.Sprintf("* %s", err)
	}

	sort.Strings(points)
	return fmt.Sprintf(
		"%d error(s) decoding:\n\n%s",
		len(e.Errors), strings.Join(points, "\n"))
}

// Unwrap implements the Unwrap function added in Go 1.20.
func (e *joinedError) Unwrap() []error {
	if e == nil {
		return nil
	}

	result := make([]error, len(e.Errors))
	for i, e := range e.Errors {
		result[i] = errors.New(e)
	}

	return result
}

// TODO: replace with errors.Join when Go 1.20 is minimum version.
func appendErrors(errors []string, err error) []string {
	switch e := err.(type) {
	case *joinedError:
		return append(errors, e.Errors...)
	default:
		return append(errors, e.Error())
	}
}
