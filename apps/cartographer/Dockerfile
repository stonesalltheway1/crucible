# syntax=docker/dockerfile:1.7
#
# Multi-stage build for crucible-cartographer.
#   1. builder:  Go 1.23-alpine, vendors and compiles the static binary.
#   2. runtime:  distroless/static, ~2 MB image, non-root.
#
# Build:
#   docker build -f apps/cartographer/Dockerfile -t cartographer:dev .
# Run:
#   docker run --rm -p 9420:9420 cartographer:dev
#
# This Dockerfile expects the build context to be the repo root so it
# can copy the cartographer module + its (currently zero) intra-repo
# replace directives in one COPY layer.

ARG GO_VERSION=1.23
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS builder

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=2026.06.0

WORKDIR /src

# Cache module deps separately so a code-only change skips re-download.
COPY apps/cartographer/go.mod ./apps/cartographer/go.mod
RUN cd apps/cartographer && go mod download

# Copy the cartographer source last so source edits don't bust the dep
# layer.
COPY apps/cartographer/ ./apps/cartographer/

WORKDIR /src/apps/cartographer
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath \
        -ldflags "-s -w -X main.version=${VERSION}" \
        -o /out/crucible-cartographer \
        ./cmd/crucible-cartographer

# ---- Runtime: distroless static (no shell, no package manager) ----
FROM gcr.io/distroless/static-debian12:nonroot

LABEL org.opencontainers.image.title="crucible-cartographer"
LABEL org.opencontainers.image.description="Crucible Cartographer — day-1 customer experience"
LABEL org.opencontainers.image.source="https://github.com/stonesalltheway1/crucible"
LABEL org.opencontainers.image.licenses="Apache-2.0"

# Copy the binary; distroless runs as 'nonroot' (uid 65532) by default.
COPY --from=builder /out/crucible-cartographer /usr/local/bin/crucible-cartographer

EXPOSE 9420
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/crucible-cartographer"]
