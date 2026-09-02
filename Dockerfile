FROM golang:1.26.6-alpine AS build
RUN apk add --no-cache --update git

WORKDIR /go/src/github.com/newrelic/nri-discovery-kubernetes
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o bin/nri-discovery-kubernetes ./cmd/discovery/

FROM alpine:latest
RUN apk add --no-cache ca-certificates

USER nobody
COPY --from=build /go/src/github.com/newrelic/nri-discovery-kubernetes/bin/nri-discovery-kubernetes /bin/
ENTRYPOINT ["/bin/nri-discovery-kubernetes"]
