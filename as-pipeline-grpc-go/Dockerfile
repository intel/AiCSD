########################################################################
 # Copyright (c) Intel Corporation 2024
 # SPDX-License-Identifier: BSD-3-Clause
########################################################################

FROM gocv/opencv:4.8.1
LABEL license='SPDX-License-Identifier: BSD-3-Clause' \
  copyright='Copyright (c) 2024: Intel Corporation'
LABEL name='as-pipeline-grpc-go' \
      version="${MSVERSION}"

RUN apt-get update && \
    apt-get -y install git make gcc libc-dev libsodium-dev ca-certificates dumb-init libzmq3-dev unzip build-essential autoconf libtool protobuf-compiler libprotobuf-dev net-tools --fix-missing && \
    rm -rf /var/lib/apt/lists/*

RUN mkdir /app
WORKDIR /app

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

# Compile API
RUN wget https://raw.githubusercontent.com/openvinotoolkit/model_server/main/src/kfserving_api/grpc_predict_v2.proto
RUN echo 'option go_package = "./grpc-client";' >> grpc_predict_v2.proto
RUN protoc --go_out="./" --go-grpc_out="./" ./grpc_predict_v2.proto

COPY . .
COPY go.mod ./
RUN go mod tidy

ARG MAKE="make pipeline-grpc-go"
RUN $MAKE

# COPY --from=builder /app/Attribution.txt /Attribution.txt
RUN mv /app/grpc_predict_v2.proto /grpc_predict_v2.proto
RUN mv /app/as-pipeline-grpc-go/LICENSE /LICENSE
RUN mv /app/as-pipeline-grpc-go/res/ /res/
RUN mv /app/as-pipeline-grpc-go/as-pipeline-grpc-go /as-pipeline-grpc-go

EXPOSE 59790
EXPOSE 59791

ENTRYPOINT ["/as-pipeline-grpc-go"]
CMD ["-cp=consul.http://edgex-core-consul:8500", "--registry", "--confdir=/res", "-s"]
