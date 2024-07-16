########################################################################
 # Copyright (c) Intel Corporation 2023
 # SPDX-License-Identifier: BSD-3-Clause
########################################################################

.PHONY: build tidy test clean docker

GO=CGO_ENABLED=1 go

SERVICE_NAME=file-receiver-gateway

build:
	make -C ../ $(SERVICE_NAME)

tidy:
	go mod tidy

docker:
	make -C ../ docker-$(SERVICE_NAME)

test:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) vet ./...
	gofmt -l $$(find . -type f -name '*.go'| grep -v "/vendor/")
	[ "`gofmt -l $$(find . -type f -name '*.go'| grep -v "/vendor/")`" = "" ]

fuzztest:
	$(GO) test -fuzz=FuzzTransmitJob -fuzztime 30s ./controller
	
clean:
	rm -f ms-$(SERVICE_NAME)