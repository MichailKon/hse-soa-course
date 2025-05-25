FROM golang:1.24

WORKDIR /app

COPY go.mod ./
COPY ./api-gateway ./api-gateway
COPY ./common ./common

RUN go mod tidy
RUN go mod download

RUN go build -o api-gateway ./api-gateway

CMD ["./api-gateway/api-gateway"]
