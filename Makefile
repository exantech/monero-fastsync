DOCKER_IMAGE=fastsync-build
UID=$(shell id -u)
PWD=$(shell pwd)

build_image:
	docker build -t ${DOCKER_IMAGE} .

linux: build_image
	docker run -u ${UID} -v ${PWD}:/go/src/github.com/exantech/monero-fastsync ${DOCKER_IMAGE}
