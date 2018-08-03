package utils

import (
	"fmt"
)

// CatchPanic catches any panic and writes the error to out.
func CatchPanic(out *error) {
	if err := recover(); err != nil {
		*out = UnifyError(err)
	}
}

// UnifyError converts e to error.
func UnifyError(e interface{}) error {
	switch e.(type) {
	case error:
		return e.(error)
	default:
		return fmt.Errorf("%+v", e)
	}
}
