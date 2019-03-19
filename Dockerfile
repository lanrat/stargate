# build stage
FROM golang:alpine AS build-env
RUN apk update && apk add --no-cache make git

WORKDIR /go/app/
COPY . .
ENV CGO_ENABLED=0
RUN make deps
RUN make

# final stage
FROM alpine
COPY --from=build-env /go/app/stargate /usr/local/bin/
USER 1000

ENTRYPOINT ["stargate"]