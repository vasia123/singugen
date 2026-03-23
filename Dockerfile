# Stage 1: Build both binaries
FROM golang:1.26.1-alpine AS builder

RUN apk add --no-cache ca-certificates tzdata

RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid 10321 \
    singugen

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

ARG VERSION=dev
ARG TARGETARCH

RUN CGO_ENABLED=0 GOARCH=${TARGETARCH} go build \
    -ldflags "-X main.version=${VERSION}" \
    -o /singugen ./cmd/singugen

RUN CGO_ENABLED=0 GOARCH=${TARGETARCH} go build \
    -o /singugen-agent ./cmd/agent

# Stage 2: Prepare filesystem with busybox
FROM busybox:stable AS base

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder /singugen /singugen
COPY --from=builder /singugen-agent /usr/local/bin/singugen-agent

RUN mkdir -p /data && chown 10321:10321 /data
RUN mkdir -p /tmp && chmod 1777 /tmp

# Stage 3: Minimal runtime
FROM scratch

COPY --from=base / /

USER singugen
WORKDIR /data

ENTRYPOINT ["/singugen"]
