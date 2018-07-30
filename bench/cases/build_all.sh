#!/bin/bash

cd `dirname $0`

rm -r build || true
mkdir build

find . -name "Cargo.toml" -exec ./build_item.sh "{}" ";"

find . -regex ".*/target/wasm32-unknown-unknown/release/[0-9a-zA-Z_]+.wasm" \
    -exec cp "{}" build/ ";"
