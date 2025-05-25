FROM golang:1.24

WORKDIR /app

COPY go.mod ./
COPY ./post-service ./post-service
COPY ./common ./common

RUN go mod tidy
RUN go mod download

RUN go build -o post-service ./post-service

CMD ["./post-service/post-service"]
