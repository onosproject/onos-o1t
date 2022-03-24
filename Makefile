# SPDX-FileCopyrightText: 2022 2020-present Open Networking Foundation <info@opennetworking.org>
#
# SPDX-License-Identifier: Apache-2.0

export CGO_ENABLED=1
export GO111MODULE=on

.PHONY: build

ONOS_O1T_VERSION := latest
ONOS_BUILD_VERSION := v0.6.4
ONOS_PROTOC_VERSION := v0.6.4

build: # @HELP build the Go binaries and run all validations (default)
build:
	go build -o build/_output/onos-o1t ./cmd/onos-o1t

build-tools:=$(shell if [ ! -d "./build/build-tools" ]; then cd build && git clone https://github.com/onosproject/build-tools.git; fi)
include ./build/build-tools/make/onf-common.mk

test: # @HELP run the unit tests and source code validation
test: build deps linters license
	go test -race github.com/onosproject/onos-o1t/pkg/...
	go test -race github.com/onosproject/onos-o1t/cmd/...

jenkins-test:  # @HELP run the unit tests and source code validation producing a junit style report for Jenkins
jenkins-test: build deps license linters
	TEST_PACKAGES=github.com/onosproject/onos-o1t/... ./build/build-tools/build/jenkins/make-unit

protos: # @HELP compile the protobuf files (using protoc-go Docker)
	docker run -it -v `pwd`:/go/src/github.com/onosproject/onos-o1t \
		-w /go/src/github.com/onosproject/onos-o1t \
		--entrypoint build/bin/compile-protos.sh \
		onosproject/protoc-go:${ONOS_PROTOC_VERSION}

onos-o1t-docker: # @HELP build onos-o1t Docker image
onos-o1t-docker:
	@go mod vendor
	docker build . -f build/onos-o1t/Dockerfile \
		-t onosproject/onos-o1t:${ONOS_O1T_VERSION}
	@rm -rf vendor

images: # @HELP build all Docker images
images: build onos-o1t-docker

kind: # @HELP build Docker images and add them to the currently configured kind cluster
kind: images
	@if [ "`kind get clusters`" = '' ]; then echo "no kind cluster found" && exit 1; fi
	kind load docker-image onosproject/onos-o1t:${ONOS_O1T_VERSION}

all: build images

publish: # @HELP publish version on github and dockerhub
	./build/build-tools/publish-version ${VERSION} onosproject/onos-o1t

jenkins-publish: # @HELP Jenkins calls this to publish artifacts
	./build/bin/push-images
	./build/build-tools/release-merge-commit

clean:: # @HELP remove all the build artifacts
	rm -rf ./build/_output ./vendor ./cmd/onos-o1t/onos-o1t
	go clean -testcache github.com/onosproject/onos-o1t/...

