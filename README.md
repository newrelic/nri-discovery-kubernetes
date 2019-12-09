# New Relic Kubernetes Auto-Discovery

[![Build Status](https://travis-ci.org/newrelic/nri-discovery-kubernetes.svg?branch=master)](https://travis-ci.org/newrelic/nri-prometheus.svg?branch=master)
[![CLA assistant](https://cla-assistant.io/readme/badge/newrelic/nri-discovery-kubernetes)](https://cla-assistant.io/newrelic/nri-discovery-kubernetes)

Automatically discover containers running inside Kubernetes.
Returns a list of containers and their metadata in all namespaces by default, but can be configured to discover on specific namespaces only.
The application is meant to be run in conjunction with the Infrastructure agent to automatically configure integrations based on the discovered containers.

## Development

This integration requires having a Kubernetes cluster available to deploy & run
it. For development, we recommend using [Docker](https://docs.docker.com/install/), [Minikube](https://minikube.sigs.k8s.io/docs/start/) & [skaffold](https://skaffold.dev/docs/getting-started/#installing-skaffold).

However, at the moment the tests are totally isolated and you don't need a cluster to run them.

### Prerequisites

1. **Go 1.11**. This project uses the [go modules](https://github.com/golang/go/wiki/Modules) support, which makes
   it incompatible with previous Go versions. If using **Go 1.11** set **GO111MODULE=on**.
2. Ensure you added `$GOPATH/bin` to your `$PATH`, otherwise builds won't be possible.

If you want to learn more about the GOPATH, check the [official Go docs](https://golang.org/doc/code.html#GOPATH).

### Running the tests & linters

You can run the linters with `make validate` and the tests with `make test`.

### Build the binary

To build the project run: `make build`. This will output the binary release at `bin/nri-discovery-kubernetes`.

### Build the docker image

In case you wish to push your own version of the image to a Docker registry, you can build it with:

```bash
IMAGE_NAME=<YOUR_IMAGE_NAME> `make docker-build`
```

And push it later with `docker push`

### Executing the discovery in a development cluster

You can execute the discovery in a local development cluster (example minikube) or in a cloud cluster (example: GCP)

- You need to configure how to deploy the integration in the cluster. For minikube you can use the provided yaml in 
 `deploy/minikube.yaml`. Copy `deploy/skaffold.yaml.template` to `deploy/skaffold.yaml` and replace the placeholders, 
 and run `make test/skaffold` and you should be good to go.

- To deploy into a GCP K8s cluster, Copy `deploy/skaffold.yaml.template` to `deploy/skaffold.yaml` 
 and `deploy/gcp.yaml.template` to `deploy/gcp.yaml` and replace the placeholders.
Once you have it configured, deploy it in your Kubernetes cluster with `make test/skaffold-gcp`