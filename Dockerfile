FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o muhomu .

# ── Final image ──────────────────────────────────────────────────────────────
FROM alpine:latest

LABEL org.opencontainers.image.title="muhomu"
LABEL org.opencontainers.image.description="A customizable browser new tab dashboard"
LABEL org.opencontainers.image.source="https://github.com/gary-host-laptop/muhomu"
LABEL org.opencontainers.image.licenses="MIT"

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/muhomu .
COPY static/ ./static/
COPY templates/ ./templates/

RUN mkdir -p /data/images/profile /data/images/bg /data/images/favicons /data/widget-images /data/themes

EXPOSE 4444

ENV PORT=4444
ENV DATA_DIR=/data
ENV STATIC_DIR=/app/static

CMD ["./muhomu", "-port", "4444", "-data", "/data", "-static", "/app/static", "-config", "/data/config.yaml"]
