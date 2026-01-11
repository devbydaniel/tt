.PHONY: build run dev test clean build-sync run-sync dev-sync

build:
	go build -o tt ./cmd/tt

run: build
	./tt

dev: build
	TT_DATA_DIR=./dev-data ./tt

test:
	go test ./...

clean:
	rm -f tt tt-sync

build-sync:
	go build -o tt-sync ./cmd/tt-sync

run-sync: build-sync
	./tt-sync

dev-sync: build-sync
	TT_DATA_DIR=./dev-data ./tt-sync
