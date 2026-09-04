# syntax=docker/dockerfile:1

ARG BASE=gcr.io/distroless/static-debian12:nonroot

# ── Release build ──
FROM golang:1.26-bookworm AS release-build
ARG TARGETARCH
ENV CGO_ENABLED=0 GOOS=linux
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN go build \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /out/sit-ics ./cmd/sit-ics

# ── Release image ──
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=release-build --chown=65534:65534 --chmod=555 /out/sit-ics /sit-ics
USER 65534:65534
EXPOSE 8080
CMD ["/sit-ics"]

# # ── Debug build ──
# FROM golang:1.26-bookworm AS debug-build
# ARG TARGETARCH
# ENV CGO_ENABLED=0 GOOS=linux
# WORKDIR /src
# COPY go.mod go.sum ./
# RUN go mod download
# COPY . .
# RUN go build -o /out/sit-ics ./cmd/sit-ics
#
# # ── Debug image ──
# FROM debug-build AS debug
# RUN apt-get update && apt-get install -y --no-install-recommends \
#     curl tini && \
#     rm -rf /var/lib/apt/lists/*
# USER 65534:65534
# COPY --from=debug-build --chown=65534:65534 --chmod=555 /out/sit-ics /sit-ics
# EXPOSE 8080
# ENTRYPOINT ["tini", "--"]
# CMD ["/sit-ics"]
