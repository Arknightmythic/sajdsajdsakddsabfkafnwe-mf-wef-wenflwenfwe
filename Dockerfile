# Build stage
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/main .

# Copy production env file as .env
COPY .env.production .env

# Create upload directory with proper permissions
RUN mkdir -p /tmp/file_upload && chmod 777 /tmp/file_upload

EXPOSE 8000

VOLUME ["/tmp/file_upload"]

CMD ["./main"]