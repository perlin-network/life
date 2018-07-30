#!/bin/bash

find "build" -name "*.wasm" -exec ./run_item.sh "{}" ";"
