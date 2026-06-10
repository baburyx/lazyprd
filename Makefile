GOPATH ?= $(HOME)/.cache/go
GOBIN ?= $(HOME)/.local/bin

export GOPATH
export GOBIN

.PHONY: build clean install test tidy verify

build:
	mkdir -p bin
	go build -o bin/lazyprd .

clean:
	rm -rf bin dist lazyprd coverage.out

install:
	mkdir -p $(GOBIN)
	go install .

test:
	go test ./...

tidy:
	go mod tidy

verify:
	gofmt -w *.go
	go test ./...
	go build ./...
	rm -f lazyprd
