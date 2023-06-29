########################################################################
 # Copyright (c) Intel Corporation 2023
 # SPDX-License-Identifier: BSD-3-Clause
########################################################################

.PHONY: docker-log-analytics-setup download-filebeat-win prepare download-zip unzip-filebeat copy-filebeat quick-clean clean
include ../.env

PROJECT ?= aicsd
SETUP_SERVICE := log-analytics-setup
MSVERSION=$(shell cat ./VERSION 2>/dev/null || echo 0.0.0)
GIT_SHA=$(shell git rev-parse HEAD)

FILEBEAT_SERVICE := filebeat
FILEBEATWINDOWS := $(FILEBEAT_SERVICE)-win
DOWNLOADDIR := $(FILEBEATWINDOWS)/downloadFold
NECESSARY_ITEMS := LICENSE.txt NOTICE.txt README.md $(FILEBEAT_SERVICE).exe

docker-log-analytics-setup:
	docker build \
		--build-arg http_proxy \
		--build-arg https_proxy \
		--build-arg ELASTIC_VERSION=${ELASTIC_VERSION} \
		-f setup/Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t ${PROJECT}/${SETUP_SERVICE}:$(GIT_SHA) \
		-t ${PROJECT}/${SETUP_SERVICE}:${MSVERSION}-dev \
		.

download-filebeat-win: prepare quick-clean
	echo Filebeat downloaded...

prepare: download-zip $(DOWNLOADDIR) unzip-filebeat copy-filebeat

download-zip:
	curl 'https://artifacts.elastic.co/downloads/beats/filebeat/filebeat-8.3.2-windows-x86_64.zip' --output $(FILEBEAT_SERVICE).zip

$(DOWNLOADDIR):
	mkdir -p $(DOWNLOADDIR)

unzip-filebeat:
	unzip -q $(FILEBEAT_SERVICE).zip -d $(DOWNLOADDIR)

copy-filebeat:
	for item in $(NECESSARY_ITEMS); do \
		cp $(DOWNLOADDIR)/*/$$item $(FILEBEATWINDOWS)/$$item; \
	done;

quick-clean:
	rm -rf $(DOWNLOADDIR) $(FILEBEAT_SERVICE).zip
	
clean: quick-clean
	for item in $(NECESSARY_ITEMS); do \
		rm -f $(FILEBEATWINDOWS)/$$item; \
	done;