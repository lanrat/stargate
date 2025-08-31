default: stargate

RELEASE_DEPS = fmt lint
include release.mk

SOURCES := $(shell find . -type f -name "*.go")


.PHONY: all fmt clean docker lint

stargate: ${SOURCES} go.mod go.sum
	CGO_ENABLED=0 go build -ldflags "-w -s -X main.version=${VERSION}" -trimpath -o $@

clean:
	rm stargate

fmt:
	go fmt ./...

lint:
	golangci-lint run

docker: Dockerfile *.go go.mod
	docker build -t lanrat/stargate --build-arg VERSION=${VERSION} .

update-deps: go.mod
	GOPROXY=direct go get -u ./...
	go mod tidy

deps: go.mod
	GOPROXY=direct go mod download

.PHONY: goreleaser
goreleaser:
	goreleaser release --snapshot --clean


.PHONY: test
test:
	go test -v ./...