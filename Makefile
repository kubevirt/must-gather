IMAGE_REGISTRY ?= quay.io
IMAGE_TAG ?= latest
GINKGO_JUNIT_REPORT ?= report.xml

# MUST_GATHER_IMAGE needs to be passed explicitly to avoid accidentally pushing to kubevirt/must-gather
check-image-env:
	ifndef MUST_GATHER_IMAGE
	$(error MUST_GATHER_IMAGE is not set.)
	endif

build: check-image-env docker-build docker-push

# check
check:
	shellcheck -e SC2016 collection-scripts/*

docker-build: check-image-env
	docker build -t ${IMAGE_REGISTRY}/${MUST_GATHER_IMAGE}:${IMAGE_TAG} .

docker-push: check-image-env
	docker push ${IMAGE_REGISTRY}/${MUST_GATHER_IMAGE}:${IMAGE_TAG}

test-build:
	(cd tests; go test -c -o must-gather.test .)

test: test-build
	ACK_GINKGO_DEPRECATIONS=1.16.4 tests/must-gather.test --ginkgo.focus=level:product --junit-output=${GINKGO_JUNIT_REPORT} --ginkgo.v

.PHONY: build docker-build docker-push
