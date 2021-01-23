.PHONY: build
build:
	go build ./cmd/webgen

.PHONY: vet
vet:
	go vet ./...
	exhaustive ./...
