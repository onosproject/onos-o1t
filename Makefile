# SPDX-FileCopyrightText: 2022 2020-present Open Networking Foundation <info@opennetworking.org>
#
# SPDX-License-Identifier: Apache-2.0

export CGO_ENABLED=1
export GO111MODULE=on

.PHONY: build

ONOS_O1T_VERSION ?= latest
ONOS_BUILD_VERSION := v0.6.4
ONOS_PROTOC_VERSION := v0.6.4

GOLANG_CI_VERSION := v1.52.2

all: build docker-build

build: # @HELP build the Go binaries and run all validations (default)
	go build -o build/_output/onos-o1t ./cmd/onos-o1t

test: # @HELP run the unit tests and source code validation
test: build lint license
	go test -race github.com/onosproject/onos-o1t/pkg/...
	go test -race github.com/onosproject/onos-o1t/cmd/...

docker-build-onos-o1t: # @HELP build onos-o1t Docker image
	@go mod vendor
	docker build . -f build/onos-o1t/Dockerfile \
		-t onosproject/onos-o1t:${ONOS_O1T_VERSION}
	@rm -rf vendor

docker-build: # @HELP build all Docker images
docker-build: build docker-build-onos-o1t

docker-push-onos-o1t: # @HELP push onos-o1t Docker image
	docker push onosproject/onos-o1t:${ONOS_O1T_VERSION}

docker-push: # @HELP push docker images
docker-push: docker-push-onos-o1t

lint: # @HELP examines Go source code and reports coding problems
	golangci-lint --version | grep $(GOLANG_CI_VERSION) || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b `go env GOPATH`/bin $(GOLANG_CI_VERSION)
	golangci-lint run --timeout 15m

license: # @HELP run license checks
	rm -rf venv
	python3 -m venv venv
	. ./venv/bin/activate;\
	python3 -m pip install --upgrade pip;\
	python3 -m pip install reuse;\
	reuse lint

check-version: # @HELP check version is duplicated
	./build/bin/version_check.sh all

clean: # @HELP remove all the build artifacts
	rm -rf ./build/_output ./vendor ./cmd/onos-o1t/onos-o1t ./cmd/onos/onos venv
	go clean github.com/onosproject/onos-o1t/...

help:
	@grep -E '^.*: *# *@HELP' $(MAKEFILE_LIST) \
    | sort \
    | awk ' \
        BEGIN {FS = ": *# *@HELP"}; \
        {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}; \
    '