GOTOOLS =	github.com/mitchellh/gox \
			github.com/Masterminds/glide \
			github.com/rigelrozanski/shelldown/cmd/shelldown
INCLUDE = -I=. -I=${GOPATH}/src -I=${GOPATH}/src/github.com/gogo/protobuf/protobuf

all: get_vendor_deps install test

build: gen_version
	go build ./cmd/...
	go build ./integration/...

# Build binaries for Linux platform.
linux: gen_version
	integration/build/build.sh

install: gen_version
	go install ./cmd/...
	go install ./integration/...

test: test_unit #test_integration test_cluster_deployment

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
	@rm -rf ./build

gen_doc:
	cd ./docs/commands/;go build -o generator.exe; ./generator.exe

BUILD_DATE := `date -u`
GIT_HASH := `git rev-parse HEAD`
VERSION_NUMER := `cat version/version_number.txt`
VERSIONFILE := version/version_generated.go

gen_version:
	@echo "package version" > $(VERSIONFILE)
	@echo "const (" >> $(VERSIONFILE)
	@echo "  Timestamp = \"$(BUILD_DATE)\"" >> $(VERSIONFILE)
	@echo "  Version = \"$(VERSION_NUMER)\"" >> $(VERSIONFILE)
	@echo "  GitHash = \"$(GIT_HASH)\"" >> $(VERSIONFILE)
	@echo ")" >> $(VERSIONFILE)

.PHONY: all build install test test_unit get_vendor_deps clean tools
