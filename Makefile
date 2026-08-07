TARGETOS ?= linux
TARGETARCH ?= amd64
BUILD_DATE ?= $(shell date -u --iso-8601=seconds)
BUILD_VERSION ?= $(shell git rev-parse HEAD)

DOCKER_IMAGE ?= scaleway/scaleway-secrets-store-csi
DOCKER_TAG ?= $(shell git rev-parse HEAD)
PLATFORMS = linux/amd64,linux/arm64

.PHONY: build
build:
	go build -o scaleway-secrets-store-csi -ldflags "-X github.com/scaleway/scaleway-secrets-store-csi/internal/version.BuildVersion=$(BUILD_VERSION)" ./cmd/server

.PHONY: test
test:
	go test ./...

.PHONY: release
release:
	docker buildx build \
		--cache-from type=gha \
		--cache-to type=gha,mode=max \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--build-arg BUILD_VERSION=$(BUILD_VERSION) \
		--platform=$(PLATFORMS) \
		--push \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) \
		.

.PHONY: docker-build
docker-build:
	docker build . \
		--build-arg TARGETOS=$(TARGETOS) \
		--build-arg TARGETARCH=$(TARGETARCH) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--build-arg BUILD_VERSION=$(BUILD_VERSION) \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG)

.PHONY: docker-push
docker-push:
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
