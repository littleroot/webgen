.PHONY: build
build:
	go build ./cmd/exhaustive

.PHONY: vet
vet:
	go vet ./...
	exhaustive ./...
