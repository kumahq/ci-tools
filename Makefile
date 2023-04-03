.PHONY: build
build: build/release-tool

.PHONY: build/release-tool
build/release-tool:
	mkdir -p build
	go build -o build ./cmd/release-tool/...

GOPATH?=$(shell go env GOPATH)

.PHONY: check
check:
	go fmt ./...
	go mod tidy
	test -n "$$CI" || $(GOPATH)/bin/golangci-lint run -v

.PHONY: test
test:
	go test ./...

.PHONY: clean
clean:
	rm -fr build
