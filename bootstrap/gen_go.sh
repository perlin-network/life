#!/bin/sh

cargo build --release --target wasm32-unknown-unknown
wasm-gc target/wasm32-unknown-unknown/release/wasm_validation.wasm
WASM_B64=$(base64 < target/wasm32-unknown-unknown/release/bootstrap.wasm)
VM_B64=$(../life -emit-bc-unsafe -bc-exports get_code_buf,check_and_parse target/wasm32-unknown-unknown/release/bootstrap.wasm | base64)
cat > wasm.go << EOF
package bootstrap

import "encoding/base64"

var WASMBytecode = _getCodeFromB64("${WASM_B64}")
var VMBytecode = _getCodeFromB64("${VM_B64}")

func _getCodeFromB64(input string) []byte {
    code, err := base64.StdEncoding.DecodeString(input)
    if err != nil {
        panic(err)
    }
    return code
}
EOF
