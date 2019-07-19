IMAGE_REGISTRY ?= quay.io/kubevirt
MUST_GATHER_IMAGE ?= must-gather
NODE_GATHER_IMAGE ?= node-gather
IMAGE_TAG ?= latest

build: manifests docker-build docker-push

manifests: 
	./build/release-manifests.sh ${IMAGE_REGISTRY} ${NODE_GATHER_IMAGE}

docker-build: 
	docker build -t ${IMAGE_REGISTRY}/${MUST_GATHER_IMAGE}:${IMAGE_TAG} .
	docker build -t ${IMAGE_REGISTRY}/${NODE_GATHER_IMAGE}:${IMAGE_TAG} ./node-gather/

docker-push:
	docker push ${IMAGE_REGISTRY}/${MUST_GATHER_IMAGE}:${IMAGE_TAG}
	docker push ${IMAGE_REGISTRY}/${NODE_GATHER_IMAGE}:${IMAGE_TAG}

.PHONY: build docker-build docker-push manifests
