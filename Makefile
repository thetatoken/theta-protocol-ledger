GOTOOLS =	github.com/mitchellh/gox \
			github.com/Masterminds/glide \
			github.com/rigelrozanski/shelldown/cmd/shelldown
			
all: get_vendor_deps install test

build:
	go build ./cmd/...

install:
	go install ./cmd/...

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
