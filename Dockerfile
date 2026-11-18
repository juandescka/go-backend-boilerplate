# syntax=docker/dockerfile:1

FROM golang:1.24-alpine3.21 AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/hris-core-service .

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S hris_core_service \
    && adduser -S -G hris_core_service -h /app -s /sbin/nologin hris_core_service

WORKDIR /app

COPY --from=builder /out/hris-core-service /app/hris-core-service
COPY --from=builder /src/migrations/mysql /app/migrations/mysql
RUN chown -R hris_core_service:hris_core_service /app

ENV GIN_MODE=release

USER hris_core_service

EXPOSE 8001

ENTRYPOINT ["/app/hris-core-service"]
CMD ["server"]
