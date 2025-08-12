# build stage
FROM golang:alpine AS build-env
RUN apk update && apk add --no-cache make git

# Accept VERSION as a build argument
ARG VERSION
ENV VERSION=${VERSION}
ENV CGO_ENABLED=0

WORKDIR /go/app/
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN make

# final stage
FROM alpine
COPY --from=build-env /go/app/stargate /usr/local/bin/
USER 1000

ENTRYPOINT ["stargate"]