# -*- mode: Python -*-

config.define_string('helm_values_file')
config.define_string('kube_context')
cfg = config.parse()

# Settings and defaults.
helm_values_file=cfg.get('helm_values_file', './tilt-chart-values.yaml')
cluster_context = cfg.get('kube_context', 'minikube')

project_name = 'nri-discovery-kubernetes'

# Only use explicitly allowed kubeconfigs as a safety measure.
allow_k8s_contexts(cluster_context)

# Building binary locally
local_resource('Discovery binary', 'GOOS=linux make compile', deps=[
    "./src",
    "./internal",
    "./cmd",
    "./Makefile",
])

# Use custom Dockerfile for Tilt builds, which only takes locally built daemon binary for live reloading.
dockerfile = '''
FROM golang:1.17-alpine AS dlv-builder

RUN apk add gcc musl-dev && \
    go install github.com/go-delve/delve/cmd/dlv@latest

# ########################################################
FROM newrelic/infrastructure-bundle:2.8.32

COPY --from=dlv-builder /go/bin/dlv /usr/local/bin/
COPY ./%s /var/db/%s
''' % (project_name, project_name)

docker_build(
  ref="discovery-devenv",
  context='./bin',
  dockerfile_contents=dockerfile,
  entrypoint=[
      "dlv",
      "--listen=0.0.0.0:2345",
      "--headless=true",
      "--api-version=2",
      "--check-go-version=false",
      "--only-same-user=false",
      "--accept-multiclient",
      "exec",
      '/var/db/%s' % project_name,
      "--"
  ]
)

k8s_yaml(
    helm(
        'charts/discovery-devenv',
        name="discovery-devenv",
        values=[
            'values-dev.yaml'
        ]
    )
)

k8s_resource(
    "discovery-devenv",
    port_forwards = "2345:2345"
)
