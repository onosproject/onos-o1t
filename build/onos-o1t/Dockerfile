# SPDX-FileCopyrightText: 2022 2020-present Open Networking Foundation <info@opennetworking.org>
#
# SPDX-License-Identifier: Apache-2.0

FROM onosproject/golang-build:v1.0 as build

ENV GO111MODULE=on
ARG ONOS_MAKE_TARGET=build

COPY Makefile go.mod go.sum /go/src/github.com/onosproject/onos-o1t/
COPY cmd/ /go/src/github.com/onosproject/onos-o1t/cmd/
COPY pkg/ /go/src/github.com/onosproject/onos-o1t/pkg/
COPY vendor/ /go/src/github.com/onosproject/onos-o1t/vendor/
COPY build/build-tools /go/src/github.com/onosproject/onos-o1t/build/build-tools/


RUN cd /go/src/github.com/onosproject/onos-o1t && GOFLAGS=-mod=vendor make ${ONOS_MAKE_TARGET}

FROM alpine:3.20
RUN apk add libc6-compat

USER nobody

COPY --from=build /go/src/github.com/onosproject/onos-o1t/build/_output/onos-o1t /usr/local/bin/onos-o1t

ENTRYPOINT ["onos-o1t"]
