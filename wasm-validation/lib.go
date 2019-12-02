package wasm_validation

import (
	"bytes"
	"github.com/go-interpreter/wagon/validate"
	"github.com/go-interpreter/wagon/wasm"
)

func ValidateWasm(code []byte) error {
	m, err := wasm.DecodeModule(bytes.NewBuffer(code))
	if err != nil {
		return err
	}

	if err := validate.VerifyModule(m); err != nil {
		return err
	}

	return nil
}
