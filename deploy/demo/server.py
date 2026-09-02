#!/usr/bin/env python3
"""
Live demo server for nri-discovery-kubernetes.

GET /              → dashboard HTML
GET /api/services  → run discovery, return live JSON
GET /health        → 200 OK for Tilt readiness probe

Environment variables:
  CLUSTER_NAME   Kubernetes cluster name (default: minikube)
  DEMO_PORT      Port to listen on      (default: 8765)
  DISCOVERY_BIN  Path to pre-built darwin binary (falls back to `go run`)
"""
import http.server, subprocess, json, os, sys, pathlib

PORT         = int(os.environ.get('DEMO_PORT', '8765'))
CLUSTER_NAME = os.environ.get('CLUSTER_NAME', 'minikube')
ROOT         = pathlib.Path(__file__).parent.parent.parent  # repo root
BINARY       = pathlib.Path(os.environ.get('DISCOVERY_BIN',
                   str(ROOT / 'bin' / 'nri-discovery-kubernetes-darwin')))
HTML_FILE    = pathlib.Path(__file__).parent / 'demo-results.html'


RESOURCES = 'services,deployments,statefulsets,daemonsets,pvcs,crds'


def run_discovery():
    if BINARY.exists():
        cmd = [str(BINARY),
               f'--resources={RESOURCES}',
               f'--cluster_name={CLUSTER_NAME}']
        cwd = None
    else:
        cmd = ['go', 'run', './cmd/discovery/',
               f'--resources={RESOURCES}',
               f'--cluster_name={CLUSTER_NAME}']
        cwd = str(ROOT)
    try:
        r = subprocess.run(cmd, capture_output=True, text=True, timeout=60, cwd=cwd)
        return r.stdout.strip() or '[]'
    except Exception as e:
        return json.dumps({'error': str(e)})


class Handler(http.server.BaseHTTPRequestHandler):
    def log_message(self, fmt, *args):
        pass  # keep stdout clean

    def do_GET(self):
        if self.path == '/api/services':
            body = run_discovery().encode()
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.send_header('Access-Control-Allow-Origin', '*')
            self.send_header('Content-Length', str(len(body)))
            self.end_headers()
            self.wfile.write(body)

        elif self.path == '/health':
            self.send_response(200)
            self.end_headers()
            self.wfile.write(b'ok')

        else:
            try:
                body = HTML_FILE.read_bytes()
                self.send_response(200)
                self.send_header('Content-Type', 'text/html; charset=utf-8')
                self.send_header('Content-Length', str(len(body)))
                self.end_headers()
                self.wfile.write(body)
            except FileNotFoundError:
                self.send_response(404)
                self.end_headers()


if __name__ == '__main__':
    server = http.server.HTTPServer(('', PORT), Handler)
    print(f'Demo dashboard → http://localhost:{PORT}', flush=True)
    server.serve_forever()
