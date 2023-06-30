########################################################################
 # Copyright (c) Intel Corporation 2023
 # SPDX-License-Identifier: BSD-3-Clause
########################################################################

.PHONY: docker-elyra

PROJECT ?= aicsd
SERVICE := elyra
MSVERSION=$(shell cat ./VERSION 2>/dev/null || echo 0.0.0)
GIT_SHA=$(shell git rev-parse HEAD)

docker-elyra:
	docker build \
		--build-arg http_proxy \
		--build-arg https_proxy \
		-f Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t ${PROJECT}/${SERVICE}:$(GIT_SHA) \
		-t ${PROJECT}/${SERVICE}:${MSVERSION}-dev \
		.