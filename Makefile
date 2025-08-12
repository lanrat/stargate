default: stargate

RELEASE_DEPS = fmt lint
include release.mk

.PHONY: all fmt clean docker lint

stargate: *.go go.mod
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