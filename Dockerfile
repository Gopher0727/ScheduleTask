ARG GO_VERSION=1.26.2
FROM golang:${GO_VERSION}-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/scheduletask ./cmd/scheduletask

FROM alpine:3.23
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S -g 10001 scheduletask \
    && adduser -S -u 10001 -G scheduletask scheduletask
WORKDIR /app
COPY --from=builder /out/scheduletask /usr/local/bin/scheduletask
COPY config.docker.toml /app/config.toml
USER 10001:10001
EXPOSE 8082 8083
ENTRYPOINT ["/usr/local/bin/scheduletask"]
CMD ["-config", "/app/config.toml"]
