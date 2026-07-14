FROM golang:alpine AS base

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

FROM base AS dev
RUN go install github.com/air-verse/air@latest
COPY .air.toml ./
COPY cmd ./cmd
COPY internal ./internal
ENTRYPOINT ["air"]

FROM base AS build
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -o /out/voltr-api ./cmd/api

FROM alpine:3.22 AS final
RUN apk add --no-cache ca-certificates
COPY --from=build /out/voltr-api /usr/local/bin/voltr-api
EXPOSE 8080
ENTRYPOINT ["voltr-api"]

FROM base AS cli-build
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -o /out/voltr-finance ./cmd/cli

FROM alpine:3.22 AS cli
RUN apk add --no-cache ca-certificates
COPY --from=cli-build /out/voltr-finance /usr/local/bin/voltr-finance
ENTRYPOINT ["voltr-finance"]
