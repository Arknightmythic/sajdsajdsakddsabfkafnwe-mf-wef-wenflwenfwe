
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod tidy
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -o /app/main .

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app/

COPY --from=builder /app/main .

RUN mkdir -p /app/uploads/documents

EXPOSE 8080

CMD ["./main"]