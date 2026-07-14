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

FROM debian:bookworm-slim AS frontend
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY scripts/install-tailwind.sh ./scripts/install-tailwind.sh
COPY internal/webui/assets/input.css ./internal/webui/assets/input.css
COPY internal/webui/*.templ ./internal/webui/
RUN TAILWIND_BIN=/usr/local/bin/tailwindcss ./scripts/install-tailwind.sh \
    && mkdir -p /out \
    && tailwindcss -i internal/webui/assets/input.css -o /out/app.css --minify

FROM base AS build
COPY cmd ./cmd
COPY internal ./internal
COPY --from=frontend /out/app.css ./internal/webui/assets/dist/app.css
RUN CGO_ENABLED=0 go build -trimpath -o /out/voltr-api ./cmd/api

FROM alpine:3.22 AS final
RUN apk add --no-cache ca-certificates tzdata
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
