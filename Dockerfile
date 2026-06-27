FROM golang:1.22-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o mutabu .

# ── Final image ──────────────────────────────────────────────────────────────
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/mutabu .
COPY static/ ./static/

RUN mkdir -p /data/images

EXPOSE 4444

ENV PORT=4444
ENV DATA_DIR=/data
ENV STATIC_DIR=/app/static

CMD ["./mutabu", "-port", "4444", "-data", "/data", "-static", "/app/static"]
