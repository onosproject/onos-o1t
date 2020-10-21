export CGO_ENABLED=1
export GO111MODULE=on

.PHONY: build

ONOS_O1T_VERSION := latest
ONOS_BUILD_VERSION := v0.6.4
ONOS_PROTOC_VERSION := v0.6.4

build: # @HELP build the Go binaries and run all validations (default)
build:
	go build -o build/_output/onos-o1t ./cmd/onos-o1t

test: # @HELP run the unit tests and source code validation
test: build deps linters
	go test -race github.com/onosproject/onos-o1t/pkg/...
	go test -race github.com/onosproject/onos-o1t/cmd/...

coverage: # @HELP generate unit test coverage data
coverage: build deps linters license_check
	./build/bin/coveralls-coverage

deps: # @HELP ensure that the required dependencies are in place
	go build -v ./...
	bash -c "diff -u <(echo -n) <(git diff go.mod)"
	bash -c "diff -u <(echo -n) <(git diff go.sum)"

linters: # @HELP examines Go source code and reports coding problems
	golangci-lint run --timeout 30m

license_check: # @HELP examine and ensure license headers exist
	@if [ ! -d "../build-tools" ]; then cd .. && git clone https://github.com/onosproject/build-tools.git; fi
	./../build-tools/licensing/boilerplate.py -v --rootdir=${CURDIR} --boilerplate LicenseRef-ONF-Member-1.0

gofmt: # @HELP run the Go format validation
	bash -c "diff -u <(echo -n) <(gofmt -d pkg/ cmd/ tests/)"

protos: # @HELP compile the protobuf files (using protoc-go Docker)
	docker run -it -v `pwd`:/go/src/github.com/onosproject/onos-o1t \
		-w /go/src/github.com/onosproject/onos-o1t \
		--entrypoint build/bin/compile-protos.sh \
		onosproject/protoc-go:${ONOS_PROTOC_VERSION}

onos-o1t-base-docker: # @HELP build onos-o1t base Docker image
	@go mod vendor
	docker build . -f build/base/Dockerfile \
		--build-arg ONOS_BUILD_VERSION=${ONOS_BUILD_VERSION} \
		--build-arg ONOS_MAKE_TARGET=build \
		-t onosproject/onos-o1t-base:${ONOS_O1T_VERSION}
	@rm -rf vendor

onos-o1t-docker: # @HELP build onos-o1t Docker image
onos-o1t-docker: onos-o1t-base-docker
	docker build . -f build/onos-o1t/Dockerfile \
		--build-arg ONOS_O1T_BASE_VERSION=${ONOS_O1T_VERSION} \
		-t onosproject/onos-o1t:${ONOS_O1T_VERSION}

images: # @HELP build all Docker images
images: build onos-o1t-docker

kind: # @HELP build Docker images and add them to the currently configured kind cluster
kind: images
	@if [ "`kind get clusters`" = '' ]; then echo "no kind cluster found" && exit 1; fi
	kind load docker-image onosproject/onos-o1t:${ONOS_O1T_VERSION}

all: build images

publish: # @HELP publish version on github and dockerhub
	./../build-tools/publish-version ${VERSION} onosproject/onos-o1tt

bumponosdeps: # @HELP update "onosproject" go dependencies and push patch to git. Add a version to dependency to make it different to $VERSION
	./../build-tools/bump-onos-deps ${VERSION}

clean: # @HELP remove all the build artifacts
	rm -rf ./build/_output ./vendor ./cmd/onos-o1t/onos-o1t
	go clean -testcache github.com/onosproject/onos-o1t/...

help:
	@grep -E '^.*: *# *@HELP' $(MAKEFILE_LIST) \
    | sort \
    | awk ' \
        BEGIN {FS = ": *# *@HELP"}; \
        {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}; \
    '
