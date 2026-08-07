TARGETOS ?= linux
TARGETARCH ?= amd64
BUILD_VERSION ?= latest
DOCKER_IMAGE ?= scaleway/scaleway-secrets-store-csi

.PHONY: build
build:
	@go build -o scaleway-secrets-store-csi -ldflags "-X github.com/scaleway/scaleway-secrets-store-csi/internal/version.BuildVersion=$(BUILD_VERSION)" ./cmd/server

.PHONY: test
test:
	@go test ./...

.PHONY: docker-build
docker-build:
	docker build . \
		--build-arg BUILD_DATE=$(shell date -u --iso-8601=seconds) \
		--build-arg BUILD_VERSION=$(BUILD_VERSION) \
		--build-arg TARGETOS=$(TARGETOS) \
		--build-arg TARGETARCH=$(TARGETARCH) \
		-t $(DOCKER_IMAGE):$(BUILD_VERSION)

.PHONY: docker-push
docker-push:
	docker push $(DOCKER_IMAGE):$(BUILD_VERSION)
