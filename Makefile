.PHONY: all fmt clean docker

all: stargate

stargate: *.go go.mod
	go build -o $@ .

clean:
	rm stargate

fmt:
	gofmt -s -w -l .

docker: Dockerfile *.go go.mod
	docker build -t lanrat/stargate .

deps:
	go mod download