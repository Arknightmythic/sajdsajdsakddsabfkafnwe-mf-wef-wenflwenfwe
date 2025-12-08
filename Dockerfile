
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

FROM alpine:latest

RUN apk --no-cache add ca-certificates

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/main .

COPY .env.production .env

RUN mkdir -p /tmp/file_upload \
    && chown -R appuser:appgroup /tmp/file_upload \
    && chmod 755 /tmp/file_upload \
    && chown -R appuser:appgroup /app

EXPOSE 8000

VOLUME ["/tmp/file_upload"]

USER appuser

CMD ["./main"]