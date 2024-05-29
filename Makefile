########################################################################
 # Copyright (c) Intel Corporation 2024
 # SPDX-License-Identifier: BSD-3-Clause
########################################################################

.PHONY: build tidy test clean docker integration-test logs
.SILENT: get-consul-acl-token
GO=CGO_ENABLED=1 go

# VERSION file is not needed for local development, In the CI/CD pipeline, a temporary VERSION file is written
# if you need a specific version, just override below
MSVERSION=$(shell cat ./VERSION 2>/dev/null || echo 0.0.0)

# This pulls the version of the SDK from the go.mod file. If the SDK is the only required module,
# it must first remove the word 'required' so the offset of $2 is the same if there are multiple required modules
SDKVERSION=$(shell cat ./go.mod | grep 'github.com/edgexfoundry/app-functions-sdk-go/v2 v' | sed 's/require//g' | awk '{print $$2}')

PROJECT=aicsd

MICROSERVICES=job-repository data-organizer file-receiver-gateway file-receiver-oem file-sender-oem file-sender-gateway task-launcher file-watcher pipeline-sim pipeline-val
MICROSERVICES_OEM=data-organizer file-receiver-oem file-sender-oem file-watcher
MICROSERVICES_OEM_DIRS=ms-data-organizer as-file-receiver-oem ms-file-sender-oem ms-file-watcher
DOCKERS=docker-job-repository docker-data-organizer docker-file-receiver-gateway docker-file-receiver-oem docker-file-sender-oem docker-file-sender-gateway docker-task-launcher docker-file-watcher docker-pipeline-sim docker-pipeline-val docker-web-ui docker-integration-test docker-grafana docker-geti-pipeline
REPOSITORY=ms-job-repository
ORGANIZER=ms-data-organizer
RECEIVER_GATEWAY=ms-file-receiver-gateway
RECEIVER_OEM=as-file-receiver-oem
SENDER_OEM=ms-file-sender-oem
SENDER_GATEWAY=as-file-sender-gateway
TASK_LAUNCHER=as-task-launcher
FILE_WATCHER=ms-file-watcher
PIPELINE_SIM=as-pipeline-sim
PIPELINE_GRPC_GO=as-pipeline-grpc-go
AiCSD_SIM=as-pipeline-val
WEB_UI=ms-web-ui
INTEGRATION_TEST=integration-tests
GETI_PIPELINE=geti-pipeline
GATEWAY_IP_ADDR=localhost
SLEEP_TIME=30

DIRS=$(wildcard */.)
SVCS=$(filter-out pkg/. tools/. integration-tests/. ms-web-ui/., $(DIRS))

# Linux needs no file extension, or prefix unless x-compiling for windows
EXT=

GOFLAGS=-ldflags "-X github.com/edgexfoundry/app-functions-sdk-go/v2/internal.SDKVersion=$(SDKVERSION) -X github.com/edgexfoundry/app-functions-sdk-go/v2/internal.ApplicationVersion=$(MSVERSION)" -buildvcs=false

GIT_SHA=$(shell git rev-parse HEAD)

define COMPOSE_DOWN
	docker compose -p edgex -f docker-compose-edgex.yml -f docker-compose-oem.yml -f docker-compose-gateway.yml -f docker-compose-sim.yml -f docker-compose-pipeline-val.yml -f docker-compose-geti.yml -f docker-compose-elyra.yml -f docker-compose-openvino.yml -f docker-compose-edgex-spiffe-spire.yml -f docker-compose-grpc-go.yml down $1
	docker compose -p monitor -f docker-compose-monitor.yml down $1
	docker compose -p log-analytics -f docker-compose-log-analytics.yml down $1
endef

VERIFY_CLIENTS=$(FILE_WATCHER) $(RECEIVER_OEM) $(TASK_LAUNCHER) $(ORGANIZER) $(SENDER_OEM) $(REPOSITORY)

build: $(MICROSERVICES)

build-win: $(MICROSERVICES_OEM)

tidy:
	go mod tidy

job-repository:
	$(GO) build $(GOFLAGS) -o ./$(REPOSITORY)/$(REPOSITORY) ./$(REPOSITORY)

data-organizer:
	$(GO) build $(GOFLAGS) -o ./$(ORGANIZER)/$(ORGANIZER) ./$(ORGANIZER)

file-receiver-gateway:
	$(GO) build $(GOFLAGS) -o ./$(RECEIVER_GATEWAY)/$(RECEIVER_GATEWAY) ./$(RECEIVER_GATEWAY)

file-receiver-oem:
	$(GO) build $(GOFLAGS) -o ./$(RECEIVER_OEM)/$(RECEIVER_OEM) ./$(RECEIVER_OEM)

file-sender-oem:
	$(GO) build $(GOFLAGS) -o ./$(SENDER_OEM)/$(SENDER_OEM) ./$(SENDER_OEM)

file-sender-gateway:
	$(GO) build $(GOFLAGS) -o ./$(SENDER_GATEWAY)/$(SENDER_GATEWAY) ./$(SENDER_GATEWAY)

task-launcher:
	$(GO) build $(GOFLAGS) -o ./$(TASK_LAUNCHER)/$(TASK_LAUNCHER) ./$(TASK_LAUNCHER)

file-watcher:
	$(GO) build $(GOFLAGS) -o ./$(FILE_WATCHER)/$(FILE_WATCHER) ./$(FILE_WATCHER)

pipeline-sim:
	$(GO) build $(GOFLAGS) -o ./$(PIPELINE_SIM)/$(PIPELINE_SIM) ./$(PIPELINE_SIM)

pipeline-grpc-go:
	$(GO) build $(GOFLAGS) -o ./$(PIPELINE_GRPC_GO)/$(PIPELINE_GRPC_GO) ./$(PIPELINE_GRPC_GO)

pipeline-val:
	$(GO) build $(GOFLAGS) -o ./$(AiCSD_SIM)/$(AiCSD_SIM) ./$(AiCSD_SIM)

# NOTE: This is only used for local development. Jenkins CI does not use this make target
docker: ${DOCKERS}

# do not run with -j parameter as it will cause images to clean while others are being built
dev-docker: docker clean-builders

docker-build-gateway: docker-file-sender-gateway docker-task-launcher docker-file-receiver-gateway docker-web-ui docker-job-repository docker-pipeline-sim docker-geti-pipeline docker-ssh-proxy

docker-build-oem: docker-file-watcher docker-data-organizer docker-file-sender-oem docker-file-receiver-oem docker-ssh-oem

docker-job-repository:
	docker build \
	    --build-arg http_proxy \
	    --build-arg https_proxy \
		-f ${REPOSITORY}/Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t ${PROJECT}/${REPOSITORY}:$(GIT_SHA) \
		-t ${PROJECT}/${REPOSITORY}:${MSVERSION}-dev \
		.

docker-data-organizer:
	docker build \
	    --build-arg http_proxy \
	    --build-arg https_proxy \
		-f ${ORGANIZER}/Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t ${PROJECT}/${ORGANIZER}:$(GIT_SHA) \
		-t ${PROJECT}/${ORGANIZER}:${MSVERSION}-dev \
		.

docker-file-receiver-gateway:
	docker build \
	    --build-arg http_proxy \
	    --build-arg https_proxy \
		-f ${RECEIVER_GATEWAY}/Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t ${PROJECT}/${RECEIVER_GATEWAY}:$(GIT_SHA) \
		-t ${PROJECT}/${RECEIVER_GATEWAY}:${MSVERSION}-dev \
		.

docker-file-receiver-oem:
	docker build \
	    --build-arg http_proxy \
	    --build-arg https_proxy \
		-f ${RECEIVER_OEM}/Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t ${PROJECT}/${RECEIVER_OEM}:$(GIT_SHA) \
		-t ${PROJECT}/${RECEIVER_OEM}:${MSVERSION}-dev \
		.

docker-file-sender-oem:
	docker build \
	    --build-arg http_proxy \
	    --build-arg https_proxy \
		-f ${SENDER_OEM}/Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t ${PROJECT}/${SENDER_OEM}:$(GIT_SHA) \
		-t ${PROJECT}/${SENDER_OEM}:${MSVERSION}-dev \
		.

docker-file-sender-gateway:
	docker build \
	    --build-arg http_proxy \
	    --build-arg https_proxy \
		-f ${SENDER_GATEWAY}/Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t ${PROJECT}/${SENDER_GATEWAY}:$(GIT_SHA) \
		-t ${PROJECT}/${SENDER_GATEWAY}:${MSVERSION}-dev \
		.

docker-task-launcher:
	docker build \
	    --build-arg http_proxy \
	    --build-arg https_proxy \
		-f ${TASK_LAUNCHER}/Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t ${PROJECT}/${TASK_LAUNCHER}:$(GIT_SHA) \
		-t ${PROJECT}/${TASK_LAUNCHER}:${MSVERSION}-dev \
		.

docker-file-watcher:
	docker build \
	    --build-arg http_proxy \
	    --build-arg https_proxy \
		-f ${FILE_WATCHER}/Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t ${PROJECT}/${FILE_WATCHER}:$(GIT_SHA) \
		-t ${PROJECT}/${FILE_WATCHER}:${MSVERSION}-dev \
		.

docker-pipeline-sim:
	docker build \
	    --build-arg http_proxy \
	    --build-arg https_proxy \
		-f ${PIPELINE_SIM}/Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t ${PROJECT}/${PIPELINE_SIM}:$(GIT_SHA) \
		-t ${PROJECT}/${PIPELINE_SIM}:${MSVERSION}-dev \
		.

docker-pipeline-grpc-go:
	docker build \
	    --build-arg http_proxy \
	    --build-arg https_proxy \
		-f ${PIPELINE_GRPC_GO}/Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t ${PROJECT}/${PIPELINE_GRPC_GO}:$(GIT_SHA) \
		-t ${PROJECT}/${PIPELINE_GRPC_GO}:${MSVERSION}-dev \
		.

docker-pipeline-val:
	docker build \
	    --build-arg http_proxy \
	    --build-arg https_proxy \
		-f ${AiCSD_SIM}/Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t ${PROJECT}/${AiCSD_SIM}:$(GIT_SHA) \
		-t ${PROJECT}/${AiCSD_SIM}:${MSVERSION}-dev \
		.

docker-web-ui:
	docker build \
	    --build-arg http_proxy \
	    --build-arg https_proxy \
		-f ${WEB_UI}/Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t ${PROJECT}/${WEB_UI}:$(GIT_SHA) \
		-t ${PROJECT}/${WEB_UI}:${MSVERSION}-dev \
		${WEB_UI}

docker-geti-pipeline:
	docker build \
		--build-arg HTTP_PROXY \
		--build-arg HTTPS_PROXY \
		--build-arg http_proxy \
		--build-arg https_proxy \
		-f ${GETI_PIPELINE}/Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t ${PROJECT}/${GETI_PIPELINE}:$(GIT_SHA) \
		-t ${PROJECT}/${GETI_PIPELINE}:${MSVERSION}-dev \
		${GETI_PIPELINE}

docker-grafana:
	$(MAKE) -C grafana docker-grafana

docker-ssh-proxy:
	docker compose -f docker-compose-edgex-spiffe-spire.yml -f docker-compose-edgex.yml build

docker-ssh-oem:
	docker compose -f docker-compose-oem-standalone.yml build

init-filebeat: 
	$(MAKE) -C log-analytics download-filebeat-win
	
clean-filebeat:
	$(MAKE) -C log-analytics clean

docker-integration-test:
	docker build \
	    --build-arg http_proxy \
	    --build-arg https_proxy \
		-f ${INTEGRATION_TEST}/Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t ${PROJECT}/${INTEGRATION_TEST}:$(GIT_SHA) \
		-t ${PROJECT}/${INTEGRATION_TEST}:${MSVERSION}-dev \
		.

install-tools:
	$(GO) install github.com/vakenbolt/go-test-report@v0.9.3
	$(GO) install github.com/vektra/mockery/v2@latest

# The auto-verify-clients target is used to update/verify mocked client files.
auto-verify-clients:
	$(MAKE) install-tools -s
	@for svc in $(VERIFY_CLIENTS); do \
		make -C $$svc verify -s; \
	done
	@echo "Mockery Clients Verified."

# The client-update target manually updates the mocked files without checking client files.
client-update:
	$(MAKE) install-tools -s
	for svc in $(VERIFY_CLIENTS); do \
		make -C $$svc $@ -s; \
	done;

test:
	$(GO) test -coverprofile=coverage.out `go list ./... | grep -v integration-tests`
	$(GO) vet ./...
	gofmt -l $$(find . -type f -name '*.go'| grep -v "/vendor/")
	[ "`gofmt -l $$(find . -type f -name '*.go'| grep -v "/vendor/")`" = "" ]
	#./bin/test-attribution-txt.sh

# runs integration tests without the retry test cases
integration-test: files integration-test-pipeline-sim integration-test-pipeline-val

# runs integration tests for AiCSD using the pipeline simulator
integration-test-pipeline-sim: files
	$(GO) test -coverprofile=coverage.out -v ./${INTEGRATION_TEST}/pipeline-sim-tests/...

# runs integration tests for AiCSD using the pipeline simulator
integration-test-pipeline-val: files
	$(GO) test -coverprofile=coverage.out -v ./${INTEGRATION_TEST}/pipeline-val-tests/...

# runs all integration tests including the retry test cases using the pipeline simulator
integration-test-retry: files
	$(GO) test -tags=retry_tests -coverprofile=coverage.out -v ./${INTEGRATION_TEST}/retry-tests/...

unit-test-report: install-tools
	$(GO) test -coverprofile=coverage.out `go list ./... | grep -v integration-tests` -json \
    		| go-test-report -v -t "$(PROJECT) Unit Test Report" -o "unit-test-report.html"
	$(GO) tool cover -html=coverage.out -o "unit-test-cov-report.html"

# generates a test report for all of the integration test cases
integration-test-report: install-tools files
	$(GO) test -tags=retry_tests -coverprofile=coverage.out -v ./${INTEGRATION_TEST}/... -json \
    		| go-test-report -v -t "$(PROJECT) Integration Test Report" -o "integration-test-report.html"

test-report: unit-test-report integration-test-report

clean:
	for svc in $(SVCS); do \
		make -C $$svc $@; \
	done

files:
	if [ ! -d "${HOME}/data/gateway-files" ] || [ ! -d "${HOME}/data/oem-files" ]; then \
		mkdir -p ${HOME}/data/oem-files/input; \
		mkdir -p ${HOME}/data/oem-files/output; \
		chmod -R 777 ${HOME}/data/oem-files; \
		mkdir -p ${HOME}/data/gateway-files/input; \
		mkdir -p ${HOME}/data/gateway-files/output; \
		mkdir -p ${HOME}/data/gateway-files/archive; \
		mkdir -p ${HOME}/data/gateway-files/reject; \
		chmod -R 777 ${HOME}/data/gateway-files; \
	fi

run-ovms:
	docker compose -p edgex -f docker-compose-openvino.yml up -d

docker-elyra:
	$(MAKE) -C elyra $@

# The elyra tool is a pipeline visualization tool that can interact with the OVMS.
run-elyra: docker-elyra
	docker compose -p edgex -f docker-compose-elyra.yml up -d

run-portainer:
	docker compose -p portainer -f docker-compose-portainer.yml up -d

run-edgex: files
	GATEWAY_IP_ADDR=${GATEWAY_IP_ADDR} docker compose -p edgex \
		-f docker-compose-edgex.yml \
		up -d	

run-gateway: files
	GATEWAY_IP_ADDR=${GATEWAY_IP_ADDR} docker compose -p edgex \
		-f docker-compose-edgex.yml \
		-f docker-compose-edgex-spiffe-spire.yml \
		-f docker-compose-gateway.yml \
		up -d	

run-gateway-sim: files
	GATEWAY_IP_ADDR=${GATEWAY_IP_ADDR} docker compose -p edgex \
		-f docker-compose-edgex.yml \
		-f docker-compose-edgex-spiffe-spire.yml \
		-f docker-compose-gateway.yml \
		-f docker-compose-sim.yml \
		up -d

run-gateway-geti: files
	GATEWAY_IP_ADDR=${GATEWAY_IP_ADDR} docker compose -p edgex \
		-f docker-compose-edgex.yml \
		-f docker-compose-edgex-spiffe-spire.yml \
		-f docker-compose-gateway.yml \
		-f docker-compose-sim.yml \
		-f docker-compose-geti.yml \
		up -d

run: files
	GATEWAY_IP_ADDR=${GATEWAY_IP_ADDR} docker compose -p edgex \
		-f docker-compose-edgex.yml \
		-f docker-compose-oem.yml \
		-f docker-compose-gateway.yml \
		up -d

run-sim: files
	GATEWAY_IP_ADDR=${GATEWAY_IP_ADDR} docker compose -p edgex \
		-f docker-compose-edgex.yml \
		-f docker-compose-gateway.yml \
		-f docker-compose-oem.yml \
		-f docker-compose-sim.yml \
		up -d

run-geti: files
	GATEWAY_IP_ADDR=${GATEWAY_IP_ADDR} docker compose -p edgex \
		-f docker-compose-edgex.yml \
		-f docker-compose-oem.yml \
		-f docker-compose-gateway.yml \
		-f docker-compose-sim.yml \
		-f docker-compose-geti.yml \
		up -d

run-simulators: files
	GATEWAY_IP_ADDR=${GATEWAY_IP_ADDR} docker compose -p edgex \
		-f docker-compose-edgex.yml \
		-f docker-compose-sim.yml \
		-f docker-compose-pipeline-val.yml \
		up -d

run-pipeline-val: files
	GATEWAY_IP_ADDR=${GATEWAY_IP_ADDR} docker compose -p edgex \
		-f docker-compose-edgex.yml \
		-f docker-compose-pipeline-val.yml \
		up -d

run-pipeline-grpc-go: files
	GATEWAY_IP_ADDR=${GATEWAY_IP_ADDR} docker compose -p edgex \
		-f docker-compose-edgex.yml \
        -f docker-compose-pipeline-val.yml \
		-f docker-compose-grpc-go.yml \
		up -d

run-monitor: docker-grafana
	docker compose -p monitor \
		-f docker-compose-monitor.yml \
		up -d

run-log-analytics:
	docker compose -p log-analytics \
		-f docker-compose-log-analytics.yml \
		up -d

run-oem: files
	GATEWAY_IP_ADDR=${GATEWAY_IP_ADDR} docker compose -p edgex \
		-f docker-compose-oem-standalone.yml \
		up -d

down-portainer:
	docker compose -p portainer -f docker-compose-portainer.yml down

down:
	$(COMPOSE_DOWN)

down-oem:
	docker compose -p edgex -f docker-compose-oem-standalone.yml down $1

down-clean: clean-files
	$(call COMPOSE_DOWN,-v)

clean-files:
	-rm -rf ${HOME}/data/oem-files
	-rm -rf ${HOME}/data/gateway-files
	-rm -rf /tmp/data
	-sudo rm -rf /tmp/edgex

clean-keys:
	-sudo rm -rf edgex-res/ssh_keys
	-sudo rm -rf edgex-res/remote/sshd-remote/authorized_keys

clean-images:
	docker rmi --force $$(docker images | grep aicsd | awk '{print $$3}' | sort | uniq)

clean-integration-tests:
	-docker ps -a | grep -- integration-tests- | awk '{print $1}' | xargs docker rm -f
	-docker volume ls | grep -- integration-tests- | awk '{print $1}' | xargs docker volume rm -f
	-docker network ls | grep -- integration-tests- | awk '{print $1}' | xargs docker network rm -f

clean-builders:
	docker builder prune -f -a

clean-volumes:
	docker volume prune -f --filter all=true
	
docs: clean-docs
	mkdocs build
	mkdocs serve -a localhost:8008

# These will be useful for local development without the need to install mkdocs on your host
docs-builder-image:
	docker build \
		-f Dockerfile.docs \
		-t $(PROJECT)/mkdocs \
		.

build-docs: docs-builder-image
	docker run --rm \
		-v $(PWD):/docs \
		-w /docs \
		$(PROJECT)/mkdocs \
		build

serve-docs: docs-builder-image
	docker run --rm \
		-it \
		-p 8008:8000 \
		-v $(PWD):/docs \
		-w /docs \
		$(PROJECT)/mkdocs

clean-docs:
	rm -rf docs/

logs: 
	test -d logs || mkdir logs	
	$(eval KP_SERVICES=$(shell sh -c "docker ps --format '{{.Names}}' | grep edgex"))		

	for container in $(KP_SERVICES);do \
		docker logs $${container} > logs/$${container}.txt 2>&1; \
	done	
		
	docker logs web-ui > logs/web-ui.txt 2>&1 &

	sleep 5 # this wait will ensure the files have content before zipping them up, otherwise, files will be empty inside the zip file
	zip logs.zip logs/*

copy-files: files
	find ${COPY_DIR} -maxdepth 1 -type f -name "*.*" | while read name; do \
  		echo "Copying " $${name} ; \
  		cp "$${name}" ${HOME}/data/oem-files/input ; \
  		sleep ${SLEEP_TIME} ; \
    done

get-consul-acl-token:
	docker exec -it edgex-edgex-core-consul-1 /bin/sh -c \
		'cat /tmp/edgex/secrets/consul-acl-token/mgmt_token.json | jq -r '.SecretID' '

generate-ssh-tunnel-keys:
	test -f id_rsa || ssh-keygen -N '' -C oem-ssh-proxy -t rsa -b 4096 -f id_rsa
	test -d ./edgex-res/ssh_keys || mkdir ./edgex-res/ssh_keys
	cp -f id_rsa* ./edgex-res/ssh_keys
	cp -f id_rsa.pub ./edgex-res/remote/sshd-remote/authorized_keys
	rm id_rsa*

add-ssh-server-entry:
	docker exec -ti edgex-security-spire-config spire-server entry create \
    		-socketPath /tmp/edgex/secrets/spiffe/private/api.sock \
    		-parentID spiffe://edgexfoundry.org/spire/agent/x509pop/cn/remote-agent \
    		-dns "file-watcher" \
    		-spiffeID  spiffe://edgexfoundry.org/service/app-file-watcher \
    		-selector "docker:label:com.docker.compose.service:file-watcher"
	docker exec -ti edgex-security-spire-config spire-server entry create \
		-socketPath /tmp/edgex/secrets/spiffe/private/api.sock \
		-parentID spiffe://edgexfoundry.org/spire/agent/x509pop/cn/remote-agent \
		-dns "data-organizer" \
		-spiffeID  spiffe://edgexfoundry.org/service/app-data-organizer \
		-selector "docker:label:com.docker.compose.service:data-organizer"
	docker exec -ti edgex-security-spire-config spire-server entry create \
    		-socketPath /tmp/edgex/secrets/spiffe/private/api.sock \
    		-parentID spiffe://edgexfoundry.org/spire/agent/x509pop/cn/remote-agent \
    		-dns "file-sender-oem" \
    		-spiffeID  spiffe://edgexfoundry.org/service/app-file-sender-oem \
    		-selector "docker:label:com.docker.compose.service:file-sender-oem"
	docker exec -ti edgex-security-spire-config spire-server entry create \
    		-socketPath /tmp/edgex/secrets/spiffe/private/api.sock \
    		-parentID spiffe://edgexfoundry.org/spire/agent/x509pop/cn/remote-agent \
    		-dns "file-receiver-oem" \
    		-spiffeID  spiffe://edgexfoundry.org/service/app-file-receiver-oem \
    		-selector "docker:label:com.docker.compose.service:file-receiver-oem"