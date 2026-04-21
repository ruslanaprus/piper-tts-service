FROM golang:1.22 AS builder
WORKDIR /app
COPY . .
RUN go build -o piper-service main.go

FROM debian:stable-slim
WORKDIR /app
COPY --from=builder /app/piper-service .
COPY voices/ /app/voices/
COPY piper /usr/local/bin/piper

EXPOSE 5000
CMD ["./piper-service"]
