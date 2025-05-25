FROM golang:1.24

WORKDIR /app

COPY go.mod ./
COPY ./statistics-service ./statistics-service
COPY ./common ./common

RUN go mod tidy
RUN go mod download

RUN go build -o statistics-service ./statistics-service

CMD ["./statistics-service/statistics-service"]
