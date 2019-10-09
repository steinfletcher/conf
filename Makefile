setup:
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh
	go mod download
.PHONY: setup

test:
	bash -c 'diff -u <(echo -n) <(gofmt -s -d .)'
	go vet ./...
	go test -v ./...
.PHONY: test

lint:
	./bin/golangci-lint run --tests=false --enable-all --disable=lll ./...
.PHONY: lint

ci: lint test
.PHONY: ci

.DEFAULT_GOAL := ci