########################################################################
 # Copyright (c) Intel Corporation 2023
 # SPDX-License-Identifier: BSD-3-Clause
########################################################################

.PHONY: build tidy test clean docker

GO=CGO_ENABLED=1 go

SERVICE_NAME=task-launcher

build:
	make -C ../ $(SERVICE_NAME)

tidy:
	go mod tidy

verify:
	if ["`git diff ./persist`" == ""]; then \
		echo "No changes in persistence file detected."; \
	else \
		echo "Changes detected in persistence file. \nUpdating mock."; \
		$(MAKE) client-update; \
	fi
	echo "Clients for $(SERVICE_NAME) verified."
	echo
	
client-update:
	mockery --dir=./persist --output=./persist/mocks --name=Persistence

docker:
	make -C ../ docker-$(SERVICE_NAME)

test:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) vet ./...
	gofmt -l $$(find . -type f -name '*.go'| grep -v "/vendor/")
	[ "`gofmt -l $$(find . -type f -name '*.go'| grep -v "/vendor/")`" = "" ]

clean:
	rm -f as-$(SERVICE_NAME)