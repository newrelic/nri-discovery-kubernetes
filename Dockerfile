
FROM golang:1.24.4-bookworm AS build
RUN apk add --no-cache --update git make

WORKDIR /go/src/github.com/newrelic/nri-discovery-kubernetes
# cache dependencies
COPY go.mod go.sum Makefile ./
RUN make deps
COPY . .
RUN make compile-only
RUN chmod +x ./bin/nri-discovery-kubernetes

FROM alpine:latest
RUN apk add --no-cache ca-certificates

USER nobody
COPY --from=build /go/src/github.com/newrelic/nri-discovery-kubernetes/bin/nri-discovery-kubernetes /bin/
ENTRYPOINT ["/bin/nri-discovery-kubernetes"]
