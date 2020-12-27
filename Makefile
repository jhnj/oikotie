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
migrate: build
	migrate -database "postgres://johan:password@localhost:5432/oikotie?sslmode=disable" -path migrations up
