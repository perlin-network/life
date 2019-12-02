package utils

import (
	"fmt"
	"runtime/debug"
)

// CatchPanic catches any panic and writes the error to out.
func CatchPanic(out *error) {
	if err := recover(); err != nil {
		*out = fmt.Errorf("Error: %s\n---GO TRACEBACK---\n%s", UnifyError(err), string(debug.Stack()))
	}
}

// UnifyError converts e to error.
func UnifyError(e interface{}) error {
	switch et := e.(type) {
	case error:
		return et
	default:
		return fmt.Errorf("%+v", e)
	}
}
