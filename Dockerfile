FROM golang:1.19-alpine AS builder
ENV CGO_ENABLED=0
WORKDIR /backend
COPY backend/go.* .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download
COPY backend/. .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o bin/service

FROM --platform=$BUILDPLATFORM node:18.12-alpine3.16 AS client-builder
WORKDIR /ui
# cache packages in layer
COPY ui/package.json /ui/package.json
COPY ui/package-lock.json /ui/package-lock.json
RUN --mount=type=cache,target=/usr/src/app/.npm \
    npm set cache /usr/src/app/.npm && \
    npm ci
# install
COPY ui /ui
RUN npm run build

FROM alpine AS curl
RUN apk add --no-cache \
    curl

FROM curl AS docker-slim
RUN curl -L -o ds.tar.gz https://downloads.dockerslim.com/releases/1.37.3/dist_linux.tar.gz && \
    tar -xvf ds.tar.gz && \
    mv  dist_linux/docker-slim /usr/local/bin/ && \
    mv  dist_linux/docker-slim-sensor /usr/local/bin/

FROM alpine
LABEL org.opencontainers.image.title="Dockerfile Diff" \
    org.opencontainers.image.description="Compare the Dockerfile of 2 images and find their differences." \
    org.opencontainers.image.vendor="Felipe" \
    com.docker.desktop.extension.api.version="0.3.3" \
    com.docker.extension.screenshots="" \
    com.docker.extension.detailed-description="" \
    com.docker.extension.publisher-url="" \
    com.docker.extension.additional-urls="" \
    com.docker.extension.changelog=""

COPY --from=builder /backend/bin/service /
COPY docker-compose.yaml .
COPY metadata.json .
COPY docker.svg .
COPY --from=client-builder /ui/build ui
COPY --from=docker-slim /usr/local/bin/docker-slim /usr/local/bin/docker-slim
COPY --from=docker-slim /usr/local/bin/docker-slim-sensor /usr/local/bin/docker-slim-sensor

CMD /service -socket /run/guest-services/backend.sock
