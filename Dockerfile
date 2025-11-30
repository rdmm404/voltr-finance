FROM golang:alpine AS base

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download


FROM base AS dev

RUN go install github.com/air-verse/air@latest
COPY .dev ./.dev
COPY .air.toml ./

COPY cmd ./cmd
COPY internal ./internal

ENTRYPOINT [ "air" ]
CMD [ "." ]


FROM base AS build

COPY cmd ./cmd
COPY internal ./internal
RUN go build -o main cmd/main.go


FROM alpine:3.22 AS final

COPY --from=build /app/main .
ENTRYPOINT [ "./main" ]