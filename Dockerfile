FROM golang:1.24-alpine

WORKDIR /app

RUN go install github.com/air-verse/air@latest
COPY go.mod go.sum ./
RUN go mod download

COPY .air.toml sqlc.yaml ./

COPY database ./database
COPY cmd ./cmd
COPY internal ./internal
COPY .dev ./.dev

ENTRYPOINT [ "air" ]
CMD [ "." ]