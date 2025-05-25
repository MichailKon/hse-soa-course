FROM golang:1.24

WORKDIR /app

COPY go.mod ./
COPY ./user-service ./user-service
COPY ./common ./common

RUN go mod tidy
RUN go mod download

RUN go build -o user-service ./user-service

CMD ["./user-service/user-service"]
