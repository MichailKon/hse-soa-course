FROM golang:1.24-alpine as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -a -o user-service .

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/user-service .

EXPOSE 8081

CMD ["./user-service"]
