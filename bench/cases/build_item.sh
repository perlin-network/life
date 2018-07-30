#!/bin/bash

cd `dirname $1`
cargo build --release
