.PHONY: build run dev test clean

build:
	go build -o tt ./cmd/tt

run: build
	./tt

dev: build
	TT_DATA_DIR=./dev-data ./tt

test:
	go test ./...

clean:
	rm -f tt
