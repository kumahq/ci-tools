.PHONY: all
all: build check test

.PHONY: build
build: build/release-tool

.PHONY: build/release-tool
build/release-tool:
	mkdir -p build
	go build -o build ./cmd/release-tool/...

.PHONY: check
check:
	go fmt ./...
	go mod tidy
	test -n "$$CI" || mise exec -- golangci-lint run -v

.PHONY: test
test:
	go test ./...

.PHONY: clean
clean:
	rm -fr build
