GOTOOLS =	github.com/mitchellh/gox \
			github.com/Masterminds/glide \
			github.com/rigelrozanski/shelldown/cmd/shelldown
INCLUDE = -I=. -I=${GOPATH}/src -I=${GOPATH}/src/github.com/gogo/protobuf/protobuf

all: get_vendor_deps install test

build:
	go build ./cmd/...
	go build ./integration/...

install:
	go install ./cmd/...
	go install ./integration/...

test: test_unit test_integration test_cluster_deployment

test_unit:
	go test `glide novendor` -tags=unit

test_integration:
	go test `glide novendor` -tags=integration

test_experimental:
	go test -race `glide novendor` -tags=experimental

test_cluster_deployment:
	go test -race `glide novendor` -tags=cluster_deployment

get_vendor_deps: tools
	glide install

tools:
	@go get $(GOTOOLS)

clean:
	@rm -rf ./vendor

gen-doc:
	go get github.com/cpuguy83/go-md2man/md2man
	cd ./docs/commands/;go build -o generator.exe; ./generator.exe

.PHONY: all build install test test_unit get_vendor_deps clean tools
