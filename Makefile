.PHONY: clean
clean:
	rm -fr build

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
	golangci-lint run

.PHONY: test
test:
	go test ./...
