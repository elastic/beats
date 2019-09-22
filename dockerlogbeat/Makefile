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
	-mkdir vendor/github.com/elastic/beats
	git archive --remote ${GOPATH}/src/github.com/elastic/beats/ HEAD | tar -x --exclude=x-pack -C vendor/github.com/elastic/beats
	docker build --target builder -t rootfsimage-build .
	docker build --target final -t rootfsimage .
	mkdir rootfs
	docker create --name ${CONTAINER_NAME} rootfsimage true
	docker export ${CONTAINER_NAME} | tar -x -C rootfs; \
	docker rm -vf ${CONTAINER_NAME}
	docker rmi rootfsimage
	docker rmi rootfsimage-build

.PHONY: install
install:
	docker plugin create ${DOCKER_PLUGIN} .
	docker plugin enable ${DOCKER_PLUGIN}

.PHONY: clean
clean:
	rm -rf rootfs
	-docker plugin disable ${DOCKER_PLUGIN}
	-docker plugin rm ${DOCKER_PLUGIN}
