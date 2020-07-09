.PHONY: all fmt clean docker check

all: stargate

stargate: *.go go.mod
	CGO_ENABLED=0 go build -ldflags "-w -s" -trimpath -a -installsuffix cgo -o $@

clean:
	rm stargate

fmt:
	gofmt -s -w -l .

check: | lint check1 check2

check1:
	golangci-lint run

check2:
	staticcheck -f stylish -checks all ./...

lint:
	golint ./...

docker: Dockerfile *.go go.mod
	docker build -t lanrat/stargate .

deps:
	go mod download
