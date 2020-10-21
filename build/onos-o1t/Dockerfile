ARG ONOS_O1T_BASE_VERSION=latest

FROM onosproject/onos-o1t-base:$ONOS_O1T_BASE_VERSION as base

FROM alpine:3.11
RUN apk add libc6-compat

USER nobody

COPY --from=base /go/src/github.com/onosproject/onos-o1t/build/_output/onos-o1t /usr/local/bin/onos-o1t

ENTRYPOINT ["onos-o1t"]
