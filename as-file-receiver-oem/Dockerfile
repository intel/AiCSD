########################################################################
 # Copyright (c) Intel Corporation 2023
 # SPDX-License-Identifier: BSD-3-Clause
########################################################################

#build stage
FROM golang:1.21-alpine3.18 AS builder

RUN apk add --update --no-cache make
WORKDIR /app

COPY go.mod ./
RUN go mod download all
COPY . .

ARG MAKE="make file-receiver-oem"
RUN $MAKE

#final stage
FROM alpine:3.18
LABEL license='SPDX-License-Identifier: BSD-3-Clause' \
  copyright='Copyright (c) 2023: Intel Corporation'
LABEL name='as-file-receiver-oem' \ 
      version="${MSVERSION}"

COPY --from=builder /app/as-file-receiver-oem/LICENSE /LICENSE
COPY --from=builder /app/as-file-receiver-oem/res/ /res/
COPY --from=builder /app/as-file-receiver-oem/as-file-receiver-oem /as-file-receiver-oem

# dumb-init is required as security-bootstrapper uses it in the entrypoint script
RUN apk add --update --no-cache ca-certificates dumb-init

EXPOSE 59787

ENTRYPOINT ["/as-file-receiver-oem"]
CMD ["-cp=consul.http://edgex-core-consul:8500", "--registry", "-s"]