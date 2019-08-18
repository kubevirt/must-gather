IMAGE_REGISTRY ?= quay.io
MUST_GATHER_IMAGE ?= kubevirt/must-gather
NODE_GATHER_IMAGE ?= kubevirt/node-gather
IMAGE_TAG ?= latest

build: manifests docker-build docker-push

manifests: 
	./build/release-manifests.sh ${NODE_GATHER_IMAGE} ${IMAGE_REGISTRY}

docker-build: 
	docker build -t ${IMAGE_REGISTRY}/${MUST_GATHER_IMAGE}:${IMAGE_TAG} .
	docker build -t ${IMAGE_REGISTRY}/${NODE_GATHER_IMAGE}:${IMAGE_TAG} ./node-gather/

docker-push:
	docker push ${IMAGE_REGISTRY}/${MUST_GATHER_IMAGE}:${IMAGE_TAG}
	docker push ${IMAGE_REGISTRY}/${NODE_GATHER_IMAGE}:${IMAGE_TAG}

.PHONY: build docker-build docker-push manifests
