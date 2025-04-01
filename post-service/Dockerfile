FROM golang:1.24-alpine as builder

WORKDIR /app

COPY go.* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -o post-service .

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/post-service .

EXPOSE 50051

CMD ["./post-service"]
