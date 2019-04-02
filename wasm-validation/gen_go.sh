#!/bin/sh

cargo build --release --target wasm32-unknown-unknown
wasm-gc target/wasm32-unknown-unknown/release/wasm_validation.wasm
B64=$(base64 < target/wasm32-unknown-unknown/release/wasm_validation.wasm)
cat > validator.go << EOF
package wasm_validation

import "encoding/base64"

var ValidatorCode = _getValidatorCode()

func _getValidatorCode() []byte {
    code, err := base64.StdEncoding.DecodeString("${B64}")
    if err != nil {
        panic(err)
    }
    return code
}
EOF
