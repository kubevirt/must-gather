.PHONY: all format build run
all: format build

format:
	gofmt -w .

build:
	go build .

run:
	@[ "${VERSION}" ] || ( echo "VERSION is not set"; exit 1 )
	@[ "${GITHUB_TOKEN}" ] || ( echo "GITHUB_TOKEN is not set"; exit 1 )

	go run . \
		-version $(VERSION) \
		-github-token $(GITHUB_TOKEN) \
