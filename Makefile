.PHONY: build test vet race fmt clean

build:
	mkdir -p dist
	go build -o dist/dnsquerypulse ./cmd/dnsquerypulse

test:
	go test ./...

vet:
	go vet ./...

race:
	go test -race ./...

fmt:
	gofmt -w ./cmd ./internal

clean:
	rm -rf dist
