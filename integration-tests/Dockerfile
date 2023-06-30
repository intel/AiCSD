########################################################################
 # Copyright (c) Intel Corporation 2023
 # SPDX-License-Identifier: BSD-3-Clause
########################################################################

FROM docker/compose:alpine-1.29.2

RUN apk add --no-cache ca-certificates wget && \
    [ ! -e /etc/nsswitch.conf ] && echo 'hosts: files dns' > /etc/nsswitch.conf

COPY --from=golang:1.19.2-alpine3.16 /usr/local/go/ /usr/local/go/

ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH
RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 777 "$GOPATH"

RUN apk add --update --no-cache make gcc git libc-dev zeromq-dev pkgconfig
