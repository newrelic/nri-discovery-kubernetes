FROM golang:1.15.2-alpine3.13 AS build
RUN apk add --no-cache --update git make

WORKDIR /go/src/github.com/newrelic/nri-discovery-kubernetes
# cache dependencies
COPY go.mod .
COPY go.sum .
COPY Makefile Makefile
RUN make deps
COPY . .
RUN make compile-only
RUN chmod +x ./bin/nri-discovery-kubernetes

FROM alpine:latest
RUN apk add --no-cache ca-certificates

USER nobody
COPY --from=build /go/src/github.com/newrelic/nri-discovery-kubernetes/bin/nri-discovery-kubernetes /bin/
ENTRYPOINT ["/bin/nri-discovery-kubernetes"]