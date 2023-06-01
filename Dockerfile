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
COPY ui/package.json /ui/package.json
COPY ui/package-lock.json /ui/package-lock.json
RUN --mount=type=cache,target=/usr/src/app/.npm \
    npm set cache /usr/src/app/.npm && \
    npm ci
COPY ui /ui
RUN npm run build

FROM alpine AS curl
RUN apk add --no-cache \
    curl

FROM curl AS slim
ENV SLIM_VERSION=1.40.2
RUN curl -L -o ds.tar.gz https://downloads.dockerslim.com/releases/${SLIM_VERSION}/dist_linux.tar.gz && \
    tar -xvf ds.tar.gz && \
    mv  dist_linux/slim /usr/local/bin/ && \
    mv  dist_linux/slim-sensor /usr/local/bin/

FROM alpine:3.18
LABEL org.opencontainers.image.title="Dockerfile Diff" \
    org.opencontainers.image.description="Diff local or remotes images so you can more easily see the differences in their Dockerfiles." \
    org.opencontainers.image.vendor="Felipe Cruz" \
    com.docker.desktop.extension.icon="https://raw.githubusercontent.com/felipecruz91/dockerfile-diff-extension/main/icon-blue.svg" \
    com.docker.desktop.extension.api.version="0.3.3" \
    com.docker.extension.screenshots='[{"alt":"screenshot", "url":"https://raw.githubusercontent.com/felipecruz91/dockerfile-diff-extension/main/docs/images/screenshot.png"}]' \
    com.docker.extension.detailed-description="When working with Docker, you may have local images that you have built and remote images that are available in a Docker registry such as Docker Hub. The Dockerfile is a set of instructions that defines how to build a Docker image. To compare the differences between the Dockerfiles of two images, you can use this extension to compare the Dockerfiles side by side, making it easier for you to identify what changes have been made." \
    com.docker.extension.publisher-url="https://github.com/felipecruz91" \
    com.docker.extension.additional-urls='[{"title":"GitHub repository","url":"https://github.com/felipecruz91/dockerfile-diff-extension"}, {"title":"Report an issue","url":"https://github.com/felipecruz91/dockerfile-diff-extension/issues"}]' \
    com.docker.extension.changelog="<ul><li>Upgrade Slim version to 1.40.2.</li><li>Fixed vulnerabilities from base image using Docker Scout.</li></ul>" \
    com.docker.extension.categories="utility-tools"

COPY --from=builder /backend/bin/service /
COPY docker-compose.yaml .
COPY metadata.json .
COPY icon-blue.svg .
COPY --from=client-builder /ui/build ui
COPY --from=slim /usr/local/bin/slim /usr/local/bin/slim
COPY --from=slim /usr/local/bin/slim-sensor /usr/local/bin/slim-sensor

CMD /service -socket /run/guest-services/backend.sock
