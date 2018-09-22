GOTOOLS =	github.com/mitchellh/gox \
			github.com/Masterminds/glide \
			github.com/rigelrozanski/shelldown/cmd/shelldown
INCLUDE = -I=. -I=${GOPATH}/src -I=${GOPATH}/src/github.com/gogo/protobuf/protobuf

all: get_vendor_deps install test

build:
	go build ./cmd/...

install:
	go install ./cmd/...

protoc:
	#go get github.com/gogo/protobuf
	#go get github.com/gogo/protobuf/proto
	#go get github.com/gogo/protobuf/gogoproto
	#go get github.com/gogo/protobuf/protoc-gen-gogo
	#npm install -g protobufjs
	protoc $(INCLUDE) --gogo_out=plugins=:. ledger/types/serialization/*.proto
	pbjs -t static-module ledger/types/serialization/types.proto -o ledger/types/serialization/types.pb.js

test: test_unit test_integration

test_unit:
	go test `glide novendor` -tags=unit

test_integration:
	go test `glide novendor` -tags=integration

test_experimental:
	go test -race `glide novendor` -tags=experimental

get_vendor_deps: tools
	glide install

tools:
	@go get $(GOTOOLS)

clean:
	@rm -rf ./vendor

.PHONY: all build install test test_unit get_vendor_deps clean tools
