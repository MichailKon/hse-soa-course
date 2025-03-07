FROM golang:1.24-alpine as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -o api-gateway .

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/api-gateway .

EXPOSE 8080

CMD ["./api-gateway"]
