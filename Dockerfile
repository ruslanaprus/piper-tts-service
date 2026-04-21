FROM golang:1.22 AS builder
WORKDIR /app
COPY . .
RUN go build -o piper-service main.go

FROM debian:bookworm-slim
WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates curl tar libsndfile1 \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/piper-service .

RUN curl -fsSL https://github.com/rhasspy/piper/releases/download/2023.11.14-2/piper_linux_aarch64.tar.gz \
    | tar xz -C /usr/local/bin --strip-components=1

RUN useradd -m piperuser && \
    mkdir -p /app/voices && \
    chown -R piperuser:piperuser /app/voices

USER piperuser

EXPOSE 5000

HEALTHCHECK --interval=30s --timeout=10s --retries=3 \
  CMD curl -fs http://localhost:5000/health || exit 1

CMD ["./piper-service"]
