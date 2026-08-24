#################### BUILDER ###############################
FROM --platform=$BUILDPLATFORM golang:1.27-alpine3.24 AS builder

WORKDIR /build

ARG TARGETOS
ARG TARGETARCH
ARG BUILD_DATE
ARG BUILD_VERSION

ENV CGO_ENABLED=0
ENV GOVERSION=go1.26.5

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=go.sum,target=go.sum,readonly \
    --mount=type=bind,source=go.mod,target=go.mod,readonly \
    go mod download

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=bind,target=. \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath \
    -ldflags="-X github.com/scaleway/scaleway-secrets-store-csi/internal/version.BuildDate=${BUILD_DATE} \
              -X github.com/scaleway/scaleway-secrets-store-csi/internal/version.BuildVersion=${BUILD_VERSION} \
              -X github.com/scaleway/scaleway-secrets-store-csi/internal/version.GoVersion=${GOVERSION}" \
    -o /server ./cmd/server

#################### RUNTIME ###############################
FROM alpine:3.24

ARG BUILD_DATE
ARG BUILD_VERSION

LABEL org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.version="${BUILD_VERSION}"

RUN apk add --no-cache ca-certificates

COPY --from=builder /server /server

EXPOSE 8080

ENTRYPOINT ["/server"]
