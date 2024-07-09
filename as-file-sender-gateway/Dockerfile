########################################################################
 # Copyright (c) Intel Corporation 2024
 # SPDX-License-Identifier: BSD-3-Clause
########################################################################

# build stage
FROM golang:1.22.5-alpine3.19 AS builder

RUN apk add --update --no-cache make=4.4.1-r2 gcc=13.2.1_git20231014-r0 libc-dev=0.7.2-r5 libsodium-dev=1.0.19-r0 zeromq-dev=4.3.5-r2
WORKDIR /app

COPY go.mod ./
RUN go mod download all
COPY . .

ARG MAKE="make file-sender-gateway"
RUN $MAKE

# final stage
FROM alpine:3.19
LABEL license='SPDX-License-Identifier: BSD-3-Clause' \
  copyright='Copyright (c) 2024: Intel Corporation'
LABEL name='as-file-sender-gateway' \ 
      version="${MSVERSION}"

# dumb-init is required as security-bootstrapper uses it in the entrypoint script
RUN apk add --update --no-cache ca-certificates=20240226-r0 zeromq=4.3.5-r2 dumb-init=1.2.5-r3

COPY --from=builder /app/as-file-sender-gateway/LICENSE /LICENSE
COPY --from=builder /app/as-file-sender-gateway/res/ /res/
COPY --from=builder /app/as-file-sender-gateway/as-file-sender-gateway /as-file-sender-gateway

EXPOSE 59786

ENTRYPOINT ["/as-file-sender-gateway"]
CMD ["-cp=consul.http://edgex-core-consul:8500", "--registry", "--confdir=/res", "-s"]