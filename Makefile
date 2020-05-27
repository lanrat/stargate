.PHONY: all fmt clean docker check

all: stargate

stargate: *.go go.mod
	CGO_ENABLED=0 go build -ldflags "-w -s" -a -installsuffix cgo -o $@

clean:
	rm stargate

fmt:
	gofmt -s -w -l .

check:
	golangci-lint run || true
	staticcheck -unused.whole-program -checks all ./...

docker: Dockerfile *.go go.mod
	docker build -t lanrat/stargate .

deps:
	go mod download
