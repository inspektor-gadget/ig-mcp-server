# Dockerfile for Inspektor Gadget MCP Server

ARG BUILDER_IMAGE=golang:1.25.5-trixie@sha256:4f9d98ebaa759f776496d850e0439c48948d587b191fc3949b5f5e4667abef90
ARG BASE_IMAGE=gcr.io/distroless/static-debian11:latest@sha256:1dbe426d60caed5d19597532a2d74c8056cd7b1674042b88f7328690b5ead8ed

FROM --platform=${BUILDPLATFORM} ${BUILDER_IMAGE} AS builder

ARG TARGETARCH
ARG TARGETARCH
ARG VERSION=0.0.0
ENV VERSION=${VERSION}

# Copy the source code
COPY . /ig-mcp-server
WORKDIR /ig-mcp-server

# Build the ig-mcp-server binary
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
      -ldflags="-X main.version=${VERSION} -extldflags '-static'" \
      github.com/inspektor-gadget/ig-mcp-server/cmd/ig-mcp-server

# Final image
FROM ${BASE_IMAGE}

LABEL org.opencontainers.image.source=https://github.com/inspektor-gadget/ig-mcp-server
LABEL org.opencontainers.image.title="Inspektor Gadget MCP Server"
LABEL org.opencontainers.image.description="An AI interface for Inspektor Gadget to debug and monitor applications in Kubernetes clusters."
LABEL org.opencontainers.image.documentation="https://inspektor-gadget.io"
LABEL org.opencontainers.image.licenses=Apache-2.0

COPY --from=builder /ig-mcp-server/ig-mcp-server /ig-mcp-server

ENV KUBECONFIG=/kubeconfig

ENTRYPOINT ["/ig-mcp-server"]
