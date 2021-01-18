.PHONY: build
build:
	go build ./cmd/nausicaa

.PHONY: vet
vet:
	go vet ./...
	exhaustive ./...
