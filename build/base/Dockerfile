FROM onosproject/golang-build:v0.6.8

ENV GO111MODULE=on
ARG ONOS_MAKE_TARGET=build

COPY Makefile go.mod go.sum /go/src/github.com/onosproject/onos-o1t/
COPY cmd/ /go/src/github.com/onosproject/onos-o1t/cmd/
COPY pkg/ /go/src/github.com/onosproject/onos-o1t/pkg/
COPY vendor/ /go/src/github.com/onosproject/onos-o1t/vendor/

RUN cd /go/src/github.com/onosproject/onos-o1t && GOFLAGS=-mod=vendor make ${ONOS_MAKE_TARGET}
