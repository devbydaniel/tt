.PHONY: build run test clean

build:
	go build -o t ./cmd/t

run: build
	./t

test:
	go test ./...

clean:
	rm -f t
