########################################################################
 # Copyright (c) Intel Corporation 2023
 # SPDX-License-Identifier: BSD-3-Clause
########################################################################

.PHONY: docker-build-mqtt-client run-mqtt-client build serve docker-build-bento docker-build run docker-down-bento docker-down-mqtt-client down

# build mqtt client docker image
docker-build-mqtt-client:
	docker build -t paho_mqtt_client .

# start mqtt client
run-mqtt-client:
	docker run -itd --network=edgex_edgex-network paho_mqtt_client:latest

# build bento service locally
build: 
	bentoml build

# deploy bento service locally
serve: run-mqtt-client
	bentoml serve

# build bento docker image
docker-build-bento:
ifndef BENTO_TAG
	$(error BENTO_TAG is not set, correct usage - make docker-build BENTO_TAG=<bento_image_name>:<bento_image_tag>)
else
	bentoml containerize ${BENTO_TAG}	
endif

docker-build: docker-build-mqtt-client docker-build-bento

# start bento & mqtt client docker services
run:
ifndef BENTO_TAG
	$(error BENTO_TAG is not set, correct usage - make run BENTO_TAG=<bento_image_name>:<bento_image_tag>)
else
	docker run -itd --network=edgex_edgex-network -p 3000:3000 -v ${HOME}/data/gateway-files:/tmp/files ${BENTO_TAG} serve
endif
	make run-mqtt-client
	

# stop & remove bento docker service
docker-down-bento:
ifndef BENTO_TAG
	$(error BENTO_TAG is not set, correct usage - make run BENTO_TAG=<bento_image_name>:<bento_image_tag>)
else
	docker rm $$(docker stop $$(docker ps | awk -v image=${BENTO_TAG} '$$2 == image {print $$1}'))
endif

# stop & remove mqtt client docker
docker-down-mqtt-client:
		docker rm $$(docker stop $$(docker ps | awk -v image="paho_mqtt_client:latest" '$$2 == image {print $$1}'))

down: docker-down-bento docker-down-mqtt-client 


