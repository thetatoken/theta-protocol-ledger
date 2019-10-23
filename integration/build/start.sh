#!/bin/bash

echo "Building binaries..."

set -e
set -x

GOBIN=/usr/local/go/bin/go

$GOBIN build -o ./build/linux/theta ./cmd/theta
$GOBIN build -o ./build/linux/thetacli ./cmd/thetacli
$GOBIN build -o ./build/linux/dump_storeview ./integration/tools/dump_storeview
$GOBIN build -o ./build/linux/encrypt_sk ./integration/tools/encrypt_sk
$GOBIN build -o ./build/linux/generate_genesis ./integration/tools/generate_genesis
$GOBIN build -o ./build/linux/hex_obj_parser ./integration/tools/hex_obj_parser
$GOBIN build -o ./build/linux/inspect_data ./integration/tools/inspect_data
$GOBIN build -o ./build/linux/query_db ./integration/tools/query_db
$GOBIN build -o ./build/linux/sign_hex_msg ./integration/tools/sign_hex_msg

set +x 

echo "Done."



