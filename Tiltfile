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
FROM golang:1.26.6-alpine AS dlv-builder

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

# ── Live demo dashboard ────────────────────────────────────────────────────────
# Builds a native macOS binary for local use, then starts a Python HTTP server
# that re-runs discovery on every page load. Auto-refreshes every 30 s.
local_resource(
    'demo-dashboard',
    serve_cmd = 'CLUSTER_NAME=minikube DEMO_PORT=8765 python3 deploy/demo/server.py',
    readiness_probe = probe(
        http_get = http_get_action(8765, path='/health'),
        period_secs = 3,
    ),
    deps   = ['deploy/demo/server.py', 'deploy/demo/demo-results.html'],
    labels = ['demo'],
    links  = [link('http://localhost:8765', 'Demo Dashboard')],
)
