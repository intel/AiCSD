########################################################################
 # Copyright (c) Intel Corporation 2023
 # SPDX-License-Identifier: BSD-3-Clause
########################################################################

# build stage
FROM golang:1.19-alpine3.16 AS builder

RUN sed -e 's/dl-cdn[.]alpinelinux.org/dl-4.alpinelinux.org/g' -i~ /etc/apk/repositories && \
    apk add --update --no-cache make=4.3-r0 gcc=11.2.1_git20220219-r2 libc-dev=0.7.2-r3 libsodium-dev=1.0.18-r0 zeromq-dev=4.3.4-r0
WORKDIR /app

COPY go.mod ./
RUN go mod download all
COPY . .

ARG MAKE="make task-launcher"
RUN $MAKE

# final stage
FROM alpine:3.16
# TODO: Change Copyright to your company if open sourcing or remove label
LABEL license='SPDX-License-Identifier: BSD-3-Clause' \
  copyright='Copyright (c) 2023: Intel Corporation'
LABEL name='as-task-launcher' \ 
      version="${MSVERSION}"

# dumb-init is required as security-bootstrapper uses it in the entrypoint script
RUN apk add --update --no-cache ca-certificates=20230506-r0 zeromq=4.3.4-r0 dumb-init=1.2.5-r1

COPY --from=builder /app/as-task-launcher/LICENSE /LICENSE
COPY --from=builder /app/as-task-launcher/res/ /res/
COPY --from=builder /app/as-task-launcher/as-task-launcher /as-task-launcher

EXPOSE 59785

ENTRYPOINT ["/as-task-launcher"]
CMD ["-cp=consul.http://edgex-core-consul:8500", "--registry", "--confdir=/res", "-s"]