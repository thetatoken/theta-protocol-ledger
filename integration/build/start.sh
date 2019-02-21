#!/bin/bash

echo "Building binaries..."

set -e
set -x

go build -o ./build/linux/theta ./cmd/theta
go build -o ./build/linux/thetacli ./cmd/thetacli
go build -o ./build/linux/dump_storeview ./integration/tools/dump_storeview
go build -o ./build/linux/encrypt_sk ./integration/tools/encrypt_sk
go build -o ./build/linux/generate_genesis ./integration/tools/generate_genesis
go build -o ./build/linux/hex_obj_parser ./integration/tools/hex_obj_parser
go build -o ./build/linux/inspect_data ./integration/tools/inspect_data
go build -o ./build/linux/query_db ./integration/tools/query_db
go build -o ./build/linux/sign_hex_msg ./integration/tools/sign_hex_msg

set +x 

echo "Done."



