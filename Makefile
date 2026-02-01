.PHONY: build run dev test lint clean build-sync run-sync dev-sync docker-build docker-up docker-down

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -s -w \
	-X github.com/devbydaniel/tt/internal/version.Version=$(VERSION) \
	-X github.com/devbydaniel/tt/internal/version.Commit=$(COMMIT) \
	-X github.com/devbydaniel/tt/internal/version.Date=$(DATE)

build:
	go build -ldflags '$(LDFLAGS)' -o tt ./cmd/tt

run: build
	./tt

dev:
	TT_DATA_DIR=./dev-data go run -ldflags '$(LDFLAGS)' ./cmd/tt

test:
	go test ./...

lint:
	golangci-lint run ./...

clean:
	rm -f tt tt-sync

build-sync:
	go build -ldflags '$(LDFLAGS)' -o tt-sync ./cmd/tt-sync

run-sync: build-sync
	./tt-sync

dev-sync:
	TT_DATA_DIR=./dev-data go run -ldflags '$(LDFLAGS)' ./cmd/tt-sync

docker-build:
	docker build --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) --build-arg DATE=$(DATE) -t tt-sync .

docker-up:
	docker compose up -d

docker-down:
	docker compose down
