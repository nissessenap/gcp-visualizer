.PHONY: build test lint clean

build:
	go build -o gcp-visualizer cmd/gcp-visualizer/main.go

test:
	go test -v ./...

test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

lint:
	golangci-lint run

clean:
	rm -f gcp-visualizer
	rm -f coverage.out
	rm -rf /tmp/gcp-visualizer-*

install: build
	cp gcp-visualizer $(GOPATH)/bin/
