.PHONY: clean
clean:
	rm -r build

.PHONY: build
build: build/release-tool

.PHONY: build/release-tool
build/release-tool:
	mkdir -p build
	go build -o build ./cmd/release-tool/...
