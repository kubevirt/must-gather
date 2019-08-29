IMAGE_REGISTRY ?= quay.io
MUST_GATHER_IMAGE ?= kubevirt/must-gather
IMAGE_TAG ?= latest

build: docker-build docker-push

docker-build:
	docker build -t ${IMAGE_REGISTRY}/${MUST_GATHER_IMAGE}:${IMAGE_TAG} .

docker-push:
	docker push ${IMAGE_REGISTRY}/${MUST_GATHER_IMAGE}:${IMAGE_TAG}

.PHONY: build docker-build docker-push
