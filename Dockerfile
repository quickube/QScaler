# Build the manager binary
FROM golang:1.23 AS builder

WORKDIR /workspace

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# Cache dependencies
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Copy the Go source files
COPY cmd/main.go cmd/main.go
COPY api/ api/
COPY internal/ internal/

# Build with cache for build artifacts
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -a -o manager cmd/main.go

# Use distroless as the minimal base image
FROM gcr.io/distroless/static:nonroot

WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
