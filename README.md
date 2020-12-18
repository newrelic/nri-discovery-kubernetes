[![Community Plus header](https://github.com/newrelic/opensource-website/raw/master/src/images/categories/Community_Plus.png)](https://opensource.newrelic.com/oss-category/#community-plus)

# New Relic Kubernetes Auto-Discovery [![Build Status](https://travis-ci.org/newrelic/nri-discovery-kubernetes.svg?branch=master)](https://travis-ci.org/newrelic/nri-prometheus.svg?branch=master)[![CLA assistant](https://cla-assistant.io/readme/badge/newrelic/nri-discovery-kubernetes)](https://cla-assistant.io/newrelic/nri-discovery-kubernetes)

Automatically discovers containers running inside Kubernetes and returns a list of containers and their metadata in all namespaces by default. It can be configured to discover on specific namespaces only.

This application is meant to be run alongside the Infrastructure agent to automatically configure integrations based on the discovered containers.

## Building

This integration requires a Kubernetes cluster available to deploy and run. For development, we recommend using [Docker](https://docs.docker.com/install/), [Minikube](https://minikube.sigs.k8s.io/docs/start/), and [skaffold](https://skaffold.dev/docs/getting-started/#installing-skaffold).

Currently, tests are totally isolated and you don't need a cluster to run them.

### Requirements

1. **Go 1.13**. This project uses [go modules](https://github.com/golang/go/wiki/Modules), which makes it incompatible with previous Go versions.

   It can work with **Go 1.11** but you will need to set **GO111MODULE=on**.

2. Ensure you've added `$GOPATH/bin` to your `$PATH`, otherwise builds won't be possible.

If you want to learn more about GOPATH, check the [official Go docs](https://golang.org/doc/code.html#GOPATH).

### Testing

You can run the linters with `make validate` and the tests with `make test`.

### Building

To build the project run: `make build`. This will output the binary release at `bin/nri-discovery-kubernetes`.

### Build the Docker image

In case you wish to push your own version of the image to a Docker registry, you can build it with:

```bash
`docker build . -t <YOUR_IMAGE_NAME>`
```

or

```bash
`docker build . -f Dockerfile.release -t <YOUR_IMAGE_NAME>`
```

to build the image only, after building the executable. Then push it with `docker push <YOUR_IMAGE_NAME>`.

### Executing discovery in a development cluster

You can execute the discovery in a local development cluster (for example, Minikube) or in a cloud cluster (like GCP).

- You need to configure how to deploy the integration in the cluster. For minikube you can use the provided YAML in `deploy/minikube.yaml`. Copy `deploy/skaffold.yaml.template` to `deploy/skaffold.yaml` and replace the placeholders. Run `make test/skaffold` and you should be good to go.

- To deploy into a GCP K8s cluster, Copy `deploy/skaffold.yaml.template` to `deploy/skaffold.yaml`  and `deploy/gcp.yaml.template` to `deploy/gcp.yaml` and replace the placeholders. Once you have it configured, deploy it in your Kubernetes cluster with `make test/skaffold/gcp`.

## Support

Should you need assistance with New Relic products, you are in good hands with several support diagnostic tools and support channels.

> This [troubleshooting framework](https://discuss.newrelic.com/t/troubleshooting-frameworks/108787) steps you through common troubleshooting questions.

> New Relic offers NRDiag, [a client-side diagnostic utility](https://docs.newrelic.com/docs/using-new-relic/cross-product-functions/troubleshooting/new-relic-diagnostics) that automatically detects common problems with New Relic agents. If NRDiag detects a problem, it suggests troubleshooting steps. NRDiag can also automatically attach troubleshooting data to a New Relic Support ticket.

If the issue has been confirmed as a bug or is a Feature request, please file a Github issue.

**Support channels**

* [New Relic Documentation](https://docs.newrelic.com): Comprehensive guidance for using our platform
* [New Relic Community](https://discuss.newrelic.com): The best place to engage in troubleshooting questions
* [New Relic Developer](https://developer.newrelic.com/): Resources for building a custom observability applications
* [New Relic University](https://learn.newrelic.com/): A range of online training for New Relic users of every level

## Privacy

At New Relic we take your privacy and the security of your information seriously, and are committed to protecting your information. We must emphasize the importance of not sharing personal data in public forums, and ask all users to scrub logs and diagnostic information for sensitive information, whether personal, proprietary, or otherwise.

We define “Personal Data” as any information relating to an identified or identifiable individual, including, for example, your name, phone number, post code or zip code, Device ID, IP address and email address.

Review [New Relic’s General Data Privacy Notice](https://newrelic.com/termsandconditions/privacy) for more information.

## Contributing

We encourage your contributions to improve Kubernetes Auto-Discovery! Keep in mind when you submit your pull request, you'll need to sign the CLA via the click-through using CLA-Assistant. You only have to sign the CLA one time per project.

If you have any questions, or to execute our corporate CLA, required if your contribution is on behalf of a company,  please drop us an email at opensource@newrelic.com.

**A note about vulnerabilities**

As noted in our [security policy](/SECURITY.md), New Relic is committed to the privacy and security of our customers and their data. We believe that providing coordinated disclosure by security researchers and engaging with the security community are important means to achieve our security goals.

If you believe you have found a security vulnerability in this project or any of New Relic's products or websites, we welcome and greatly appreciate you reporting it to New Relic through [HackerOne](https://hackerone.com/newrelic).

If you would like to contribute to this project, please review [these guidelines](./CONTRIBUTING.md).

To all contributors, we thank you!  Without your contribution, this project would not be what it is today.

## License
nri-discovery-kubernetes is licensed under the [Apache 2.0](http://apache.org/licenses/LICENSE-2.0.txt) License.
