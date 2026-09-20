ARG GO_VERSION=1.27
ARG ALPINE_VERSION=3.22

FROM golang:${GO_VERSION}-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/service ./cmd/service

FROM golang:${GO_VERSION}-alpine AS tools

ARG GOOSE_VERSION=v3.27.2
ARG HEALTH_PROBE_VERSION=v0.4.39

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go install github.com/pressly/goose/v3/cmd/goose@${GOOSE_VERSION} \
    && CGO_ENABLED=0 go install github.com/grpc-ecosystem/grpc-health-probe@${HEALTH_PROBE_VERSION}

FROM alpine:${ALPINE_VERSION} AS migrate

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

COPY --from=tools /go/bin/goose /usr/local/bin/goose
COPY migrations /migrations

USER appuser

ENV GOOSE_DRIVER=postgres

ENTRYPOINT ["goose", "-dir", "/migrations"]
CMD ["up"]

FROM alpine:${ALPINE_VERSION} AS runtime

RUN apk add --no-cache ca-certificates \
    && addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=tools /go/bin/grpc-health-probe /usr/local/bin/grpc-health-probe
COPY --from=build --chown=appuser:appgroup /out/service /app/service
COPY --chown=appuser:appgroup configs/config.yml /app/configs/config.yml

USER appuser

EXPOSE 50051

ENTRYPOINT ["/app/service"]