package utils

import (
	"fmt"
)

func CatchPanic(out *error) {
	if err := recover(); err != nil {
		*out = UnifyError(err)
	}
}

func UnifyError(e interface{}) error {
	switch e.(type) {
	case error:
		return e.(error)
	default:
		return fmt.Errorf("%+v", e)
	}
}
