FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown
RUN go build -ldflags "-s -w \
    -X github.com/devbydaniel/tt/internal/version.Version=${VERSION} \
    -X github.com/devbydaniel/tt/internal/version.Commit=${COMMIT} \
    -X github.com/devbydaniel/tt/internal/version.Date=${DATE}" \
    -o /tt-sync ./cmd/tt-sync

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=build /tt-sync /usr/local/bin/tt-sync
EXPOSE 8080
VOLUME /data
ENV TT_DATA_DIR=/data
ENTRYPOINT ["tt-sync"]
