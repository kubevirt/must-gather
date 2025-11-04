IMAGE_REGISTRY ?= quay.io
IMAGE_TAG ?= latest
GINKGO_JUNIT_REPORT ?= report.xml

# MUST_GATHER_IMAGE needs to be passed explicitly to avoid accidentally pushing to kubevirt/must-gather
check-image-env:
ifndef MUST_GATHER_IMAGE
	$(error MUST_GATHER_IMAGE is not set.)
endif

IMAGE_NAME = $(IMAGE_REGISTRY)/$(MUST_GATHER_IMAGE):$(IMAGE_TAG)

build: check-image-env docker-multi-arch

# check
check:
	shellcheck -a -e SC2016 -e SC2317 --source-path=./collection-scripts collection-scripts/*

docker-build: check-image-env
	docker build -t ${IMAGE_NAME} -f Dockerfile.quay .

docker-push: check-image-env
	docker push ${IMAGE_NAME}

docker-build-%: check-image-env
	docker build --platform="linux/$*" -t ${IMAGE_NAME}-$* -f Dockerfile.quay .

docker-push-%: check-image-env docker-build-%
	docker push ${IMAGE_NAME}-$*

docker-multi-arch: docker-push-amd64 docker-push-arm64 docker-push-s390x
	docker manifest rm ${IMAGE_NAME} || true
	docker manifest create ${IMAGE_NAME} ${IMAGE_NAME}-amd64 ${IMAGE_NAME}-arm64 ${IMAGE_NAME}-s390x
	docker manifest push ${IMAGE_NAME}

test-build:
	(cd tests; go test -c -o must-gather.test .)

test: test-build
	tests/must-gather.test --ginkgo.label-filter=level:product --ginkgo.junit-report=${GINKGO_JUNIT_REPORT} --ginkgo.v

.PHONY: build docker-build docker-push docker-multi-arch test-build test
