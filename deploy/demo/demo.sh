#!/usr/bin/env bash
# demo.sh — Live service-discovery demo for nri-discovery-kubernetes
#
# Prerequisites:
#   kubectl context pointing to test-ramk-gke-auto-disc
#   jq installed  (brew install jq)
#   nri-discovery-kubernetes Deployment already running (deployed via skaffold -p gcp)
#
# Usage:
#   ./deploy/demo/demo.sh            # full walkthrough
#   ./deploy/demo/demo.sh --cleanup  # remove all demo-services resources
set -euo pipefail

CLUSTER_CTX="gke_k8s-o11y-team_us-west2-a_test-ramk-gke-auto-disc"
DISCOVERY_DEPLOY="nri-discovery-kubernetes"
DEMO_NS="demo-services"
SERVICES_DIR="$(dirname "$0")/services"
POLL_INTERVAL=16   # slightly longer than --watch-interval=15

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m'

# ── helpers ──────────────────────────────────────────────────────────────────

info()    { echo -e "${BLUE}[INFO]${NC} $*"; }
success() { echo -e "${GREEN}[OK]${NC}   $*"; }
warn()    { echo -e "${YELLOW}[WARN]${NC} $*"; }
header()  { echo -e "\n${BOLD}══════════════════════════════════════════════════════${NC}"; echo -e "${BOLD}  $*${NC}"; echo -e "${BOLD}══════════════════════════════════════════════════════${NC}\n"; }

ensure_context() {
  local current
  current=$(kubectl config current-context 2>/dev/null || echo "none")
  if [[ "$current" != "$CLUSTER_CTX" ]]; then
    warn "Current context is '$current'. Switching to $CLUSTER_CTX..."
    kubectl config use-context "$CLUSTER_CTX"
  fi
}

ensure_jq() {
  if ! command -v jq &>/dev/null; then
    echo "jq is required. Install with: brew install jq"; exit 1
  fi
}

get_discovery_pod() {
  kubectl get pod -n default -l app="$DISCOVERY_DEPLOY" \
    -o jsonpath='{.items[0].metadata.name}' 2>/dev/null
}

# Fetch the latest discovery snapshot from the discovery pod logs.
# Parses the most-recent JSON line (watch-mode emits one JSON object per run).
get_latest_snapshot() {
  local pod="$1"
  kubectl logs "$pod" -n default --tail=50 2>/dev/null \
    | grep -E '^\{' | tail -1
}

# Pretty-print a discovery snapshot highlighting instrumentation status.
show_snapshot() {
  local snapshot="$1"
  local ts
  ts=$(echo "$snapshot" | jq -r '.timestamp // "unknown"')
  local count
  count=$(echo "$snapshot" | jq '.items | length')
  echo -e "\n${BOLD}Snapshot at ${ts} — ${count} services discovered${NC}"

  echo "$snapshot" | jq -r '
    .items[] |
    [
      .variables.serviceName // .variables.name // "(unnamed)",
      .variables.namespace,
      .variables.serviceType // "pod",
      (if .instrumented then "✅ " + (.instrumentationType // "instrumented") else "❌ not instrumented" end)
    ] | @tsv
  ' | column -t -s $'\t' \
    | while IFS= read -r line; do
        if echo "$line" | grep -q "✅"; then
          echo -e "  ${GREEN}${line}${NC}"
        else
          echo -e "  ${RED}${line}${NC}"
        fi
      done
  echo ""
}

wait_for_discovery() {
  info "Waiting ${POLL_INTERVAL}s for discovery to pick up new service..."
  sleep "$POLL_INTERVAL"
}

cleanup() {
  header "Cleanup: removing demo-services namespace"
  kubectl delete namespace "$DEMO_NS" --ignore-not-found
  success "demo-services namespace deleted."
  exit 0
}

# ── main ──────────────────────────────────────────────────────────────────────

[[ "${1:-}" == "--cleanup" ]] && cleanup

ensure_context
ensure_jq

header "nri-discovery-kubernetes — Service Discovery Demo"
echo "  Cluster : $CLUSTER_CTX"
echo "  Namespace: $DEMO_NS (mock services)"
echo ""
echo "  Instrumentation legend:"
echo -e "  ${GREEN}✅ instrumented${NC}  — NR or OTel signals detected"
echo -e "  ${RED}✗  not instrumented${NC} — no signals detected"
echo ""

# ── Step 0: verify discovery pod is running ───────────────────────────────────
header "Step 0: Verify discovery deployment"
POD=$(get_discovery_pod)
if [[ -z "$POD" ]]; then
  warn "Discovery deployment not found. Deploy it first:"
  echo "  make test/skaffold/gcp"
  exit 1
fi
success "Discovery pod: $POD"

# Show baseline (probably just kube-system services)
SNAP=$(get_latest_snapshot "$POD")
if [[ -n "$SNAP" ]]; then
  info "Baseline snapshot (before any demo services):"
  show_snapshot "$SNAP"
fi

read -rp "Press Enter to begin deploying demo services..."

# ── Step 1: Redis (instrumented - OTel) ──────────────────────────────────────
header "Step 1: Deploy Redis — INSTRUMENTED (OpenTelemetry)"
echo "  Annotation: instrumentation.opentelemetry.io/inject-sdk=true"
echo "  Annotation: prometheus.io/scrape=true"
echo ""
kubectl apply -f "$SERVICES_DIR/redis.yaml"
success "Redis deployed."
wait_for_discovery
show_snapshot "$(get_latest_snapshot "$POD")"

read -rp "Press Enter to deploy next service..."

# ── Step 2: MongoDB (not instrumented) ────────────────────────────────────────
header "Step 2: Deploy MongoDB — NOT INSTRUMENTED"
echo "  No NR or OTel annotations — will appear as instrumented=false"
echo ""
kubectl apply -f "$SERVICES_DIR/mongodb.yaml"
success "MongoDB deployed."
wait_for_discovery
show_snapshot "$(get_latest_snapshot "$POD")"

read -rp "Press Enter to deploy next service..."

# ── Step 3: Kafka (instrumented - NR integration) ─────────────────────────────
header "Step 3: Deploy Kafka — INSTRUMENTED (New Relic on-host integration)"
echo "  Annotation: newrelic.com/integrations-config (nri-kafka config)"
echo ""
kubectl apply -f "$SERVICES_DIR/kafka.yaml"
success "Kafka deployed."
wait_for_discovery
show_snapshot "$(get_latest_snapshot "$POD")"

read -rp "Press Enter to deploy next service..."

# ── Step 4: Cassandra (not instrumented) ─────────────────────────────────────
header "Step 4: Deploy Cassandra — NOT INSTRUMENTED"
echo "  No NR or OTel annotations — will appear as instrumented=false"
echo ""
kubectl apply -f "$SERVICES_DIR/cassandra.yaml"
success "Cassandra deployed."
wait_for_discovery
show_snapshot "$(get_latest_snapshot "$POD")"

read -rp "Press Enter to deploy next service..."

# ── Step 5: RabbitMQ (not instrumented) ──────────────────────────────────────
header "Step 5: Deploy RabbitMQ — NOT INSTRUMENTED"
echo "  No NR or OTel annotations — will appear as instrumented=false"
echo ""
kubectl apply -f "$SERVICES_DIR/rabbitmq.yaml"
success "RabbitMQ deployed."
wait_for_discovery
show_snapshot "$(get_latest_snapshot "$POD")"

# ── Final summary ─────────────────────────────────────────────────────────────
header "Final Summary: All 5 services discovered"
FINAL=$(get_latest_snapshot "$POD")
show_snapshot "$FINAL"

echo ""
echo -e "${BOLD}Services NOT instrumented (need attention):${NC}"
echo "$FINAL" | jq -r '
  .items[] | select(.instrumented == false) |
  "  • " + (.variables.serviceName // .variables.name // "(unnamed)") + " [" + .variables.namespace + "]"
'

echo ""
echo -e "${BOLD}Services instrumented:${NC}"
echo "$FINAL" | jq -r '
  .items[] | select(.instrumented == true) |
  "  • " + (.variables.serviceName // .variables.name // "(unnamed)") + " [" + .variables.namespace + "]  → " + .instrumentationType
'

echo ""
info "To watch discovery live:  kubectl logs -f $POD -n default | grep -E '^\{' | jq '.'"
info "To clean up demo:         $0 --cleanup"
