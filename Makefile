DOCKER_IMAGE=fastsync-build
UID=$(shell id -u)
PWD=$(shell pwd)
VERSION=$(shell git describe --always --tags)
BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
ifneq (${BRANCH}, "master")
	VERSION:=${VERSION}-${BRANCH}
endif

version:
	@echo ${VERSION}

build_image:
	docker build -t ${DOCKER_IMAGE} .

linux: build_image
	docker run -u ${UID} -e VERSION=${VERSION} -v ${PWD}:/go/src/github.com/exantech/monero-fastsync ${DOCKER_IMAGE}
