########################################################################
 # Copyright (c) Intel Corporation 2023
 # SPDX-License-Identifier: BSD-3-Clause
########################################################################

.PHONY: build tidy test clean docker

GO=CGO_ENABLED=1 go

SERVICE_NAME=pipeline-sim

build:
	make -C ../ $(SERVICE_NAME)

tidy:
	make -C ../ tidy

docker:
	make -C ../ docker-$(SERVICE_NAME)

test:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) vet ./...
	gofmt -l $$(find . -type f -name '*.go'| grep -v "/vendor/")
	[ "`gofmt -l $$(find . -type f -name '*.go'| grep -v "/vendor/")`" = "" ]

clean:
	rm -f as-$(SERVICE_NAME)