default: build
.PHONY: build test

build: clean
	go build -o bin/ot
test:
	go test ./...
run: build
	./bin/ot
clean:
	rm -rf bin/*
