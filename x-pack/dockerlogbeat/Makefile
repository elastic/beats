DOCKER_HUB_ID?=ossifrage
NAME?=dockerlogbeat
DOCKER_PLUGIN_NAME?=${DOCKER_HUB_ID}/${NAME}
DOCKER_PLUGIN_VERSION?=0.0.1
DOCKER_PLUGIN=${DOCKER_PLUGIN_NAME}:${DOCKER_PLUGIN_VERSION}
CONTAINER_NAME=${NAME}_container
GOPATH?=~/go

all: build install

.PHONY: build
build: clean
	cd ../.. ; \
	docker build --target builder -t rootfsimage-build -f x-pack/dockerlogbeat/Dockerfile . || exit 1 ; \
	docker build --target final -t rootfsimage -f x-pack/dockerlogbeat/Dockerfile . 
	mkdir rootfs
	docker create --name ${CONTAINER_NAME} rootfsimage true
	docker export ${CONTAINER_NAME} | tar -x -C rootfs; \
	docker rm -vf ${CONTAINER_NAME}

.PHONY: install
install:
	docker plugin create ${DOCKER_PLUGIN} .
	docker plugin enable ${DOCKER_PLUGIN}

.PHONY: clean
clean:
	-docker rmi rootfsimage
	-docker rmi rootfsimage-build
	rm -rf rootfs
	-docker plugin disable ${DOCKER_PLUGIN}
	-docker plugin rm ${DOCKER_PLUGIN}
