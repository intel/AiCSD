########################################################################
 # Copyright (c) Intel Corporation 2023
 # SPDX-License-Identifier: BSD-3-Clause
########################################################################

.PHONY: build tidy test clean docker

GO=CGO_ENABLED=1 go

SERVICE_NAME=file-sender-oem

build:
	make -C ../ $(SERVICE_NAME)

tidy:
	go mod tidy

verify:
	if [ "`git diff ./clients/file_receiver`" = "" ]; then \
		echo "No changes in client file detected."; \
	else \
		echo "Changes detected in client file. \n Updating mock."; \
		$(MAKE) client-update; \
	fi
	echo "Clients for $(SERVICE_NAME) verified."
	echo

client-update:
	mockery --dir=./clients/file_receiver --output=./clients/file_receiver/mocks --all

docker:
	make -C ../ docker-$(SERVICE_NAME)

test:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) vet ./...
	gofmt -l $$(find . -type f -name '*.go'| grep -v "/vendor/")
	[ "`gofmt -l $$(find . -type f -name '*.go'| grep -v "/vendor/")`" = "" ]

clean:
	rm -f ms-$(SERVICE_NAME)