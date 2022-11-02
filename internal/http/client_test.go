package http_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"strconv"
	"sync"
	"testing"
	"time"

	internalhttp "github.com/newrelic/nri-discovery-kubernetes/internal/http"
	log "github.com/sirupsen/logrus"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"

	"github.com/newrelic/nri-discovery-kubernetes/internal/config"
)

const (
	healthz                = "/healthz"
	apiProxy               = "/api/v1/nodes/test-node/proxy"
	prometheusMetric       = "/metric"
	kubeletMetric          = "/kubelet-metric"
	kubeletMetricWithDelay = "/kubelet-metric-delay"
	nodeName               = "test-node"
	fakeTokenFile          = "./test_data/token" // nolint: gosec  // testing credentials
	retries                = 3
)

func TestClientCalls(t *testing.T) {
	l := &sync.Mutex{}

	s, requests := testHTTPServerWithEndpoints(t, l, []string{healthz, prometheusMetric, kubeletMetric})

	k8sClient, cf, inClusterConfig, logger := getTestData(s)

	kubeletClient, err := internalhttp.NewClient(
		internalhttp.DefaultConnector(k8sClient, cf, inClusterConfig, logger),
		internalhttp.WithMaxRetries(retries),
		internalhttp.WithLogger(logger),
	)

	require.NoError(t, err, "Client creation succeeded")

	r, err := kubeletClient.Get(kubeletMetric)
	l.Lock()
	_, foundHealthz := requests[healthz]
	_, foundKubelet := requests[kubeletMetric]
	l.Unlock()

	assert.True(t, foundHealthz, "Client did fallback using API Server as proxy")
	assert.True(t, foundKubelet, "Clients fetched metrics")
	assert.NoError(t, err, "Client did the request without errors")
	assert.Equal(t, r.StatusCode, http.StatusOK, "Client received a 200 status")
	assert.Len(t, requests, 2, "Kubelet was hit to get health and metrics")
	assert.Contains(t, requests, healthz, "Client hit kubelet status using local connection")
	assert.Contains(t, requests, kubeletMetric, "Client hit kubelet metrics")
}

func TestClientCallsViaAPIProxy(t *testing.T) {
	l := &sync.Mutex{}

	s, requests := testHTTPSServerWithEndpoints(t, l, []string{path.Join(apiProxy, healthz), path.Join(apiProxy, prometheusMetric), path.Join(apiProxy, kubeletMetric)})

	k8sClient, cf, inClusterConfig, logger := getTestData(s)
	cf.Host = "invalid" // disabling local connection

	kubeletClient, err := internalhttp.NewClient(
		internalhttp.DefaultConnector(k8sClient, cf, inClusterConfig, logger),
		internalhttp.WithMaxRetries(retries),
		internalhttp.WithLogger(logger),
	)

	require.NoError(t, err, "Client creation succeeded")

	r, err := kubeletClient.Get(kubeletMetric)
	l.Lock()
	_, foundHealthz := requests[path.Join(apiProxy, healthz)]
	_, foundKubelet := requests[path.Join(apiProxy, kubeletMetric)]
	l.Unlock()

	assert.True(t, foundHealthz, "Client did fallback using API Server as proxy")
	assert.True(t, foundKubelet, "Clients fetched metrics")
	assert.NoError(t, err, "Client did the request without errors")
	assert.Equal(t, r.StatusCode, http.StatusOK, "Client received a 200 status")
	assert.Len(t, requests, 2, "Kubelet was hit to get health and metrics")
	assert.Contains(t, requests, path.Join(apiProxy, healthz), "Client hit kubelet status using API Server")
	assert.Contains(t, requests, path.Join(apiProxy, kubeletMetric), "Client hit kubelet metrics")
}

func TestConfigPrecedence(t *testing.T) {
	t.Parallel()

	l := &sync.Mutex{}

	s, _ := testHTTPServerWithEndpoints(t, l, []string{healthz, prometheusMetric, kubeletMetric})
	_, cf, inClusterConfig, logger := getTestData(s)

	// We use an empty client, but the connector is retrieving the port from the config.
	k8sClient := fake.NewSimpleClientset()
	u, _ := url.Parse(s.URL)
	port, _ := strconv.Atoi(u.Port())
	cf.Port = port

	_, err := internalhttp.NewClient(
		internalhttp.DefaultConnector(k8sClient, cf, inClusterConfig, logger),
		internalhttp.WithMaxRetries(retries),
		internalhttp.WithLogger(logger),
	)
	require.NoError(t, err)
}

func TestClientFailingProbingHTTP(t *testing.T) {
	t.Parallel()

	l := &sync.Mutex{}

	s, requests := testHTTPServerWithEndpoints(t, l, []string{})

	c, cf, inClusterConfig, logger := getTestData(s)

	_, err := internalhttp.NewClient(
		internalhttp.DefaultConnector(c, cf, inClusterConfig, logger),
		internalhttp.WithMaxRetries(retries),
		internalhttp.WithLogger(logger),
	)

	assert.Error(t, err, "Client fails with error")

	l.Lock()
	_, foundLocalHealth := requests[healthz]
	_, foundAPIServerHealth := requests[path.Join(apiProxy, healthz)]
	l.Unlock()

	assert.True(t, foundLocalHealth, "Client hit Health though Local Connector")
	assert.True(t, foundAPIServerHealth, "Client hit Health though API Server proxy")

	data, err := os.ReadFile(fakeTokenFile)
	require.NoError(t, err)

	assert.Nil(t, requests[healthz].Header["Authorization"], "Client does not leak Bearer token")
	assert.Equal(t, "Bearer "+string(data), requests[path.Join(apiProxy, healthz)].Header["Authorization"][0], "Client send token to API Server")
}

func TestClientFailingProbingHTTPS(t *testing.T) {
	t.Parallel()

	l := &sync.Mutex{}

	s, requests := testHTTPSServerWithEndpoints(t, l, []string{})

	c, cf, inClusterConfig, logger := getTestData(s)
	cf.TLS = true

	_, err := internalhttp.NewClient(
		internalhttp.DefaultConnector(c, cf, inClusterConfig, logger),
		internalhttp.WithMaxRetries(retries),
		internalhttp.WithLogger(logger),
	)

	assert.Error(t, err, "Client fails with error")

	l.Lock()
	_, foundLocalHealth := requests[healthz]
	_, foundAPIServerHealth := requests[path.Join(apiProxy, healthz)]
	l.Unlock()

	assert.True(t, foundLocalHealth, "Client hit Health though Local Connector")
	assert.True(t, foundAPIServerHealth, "Client hit Health though API Server proxy")

	data, err := os.ReadFile(fakeTokenFile)
	require.NoError(t, err)

	// All request have the token as they are sent through HTTPS
	for _, v := range requests {
		assert.Equal(t, "Bearer "+string(data), v.Header["Authorization"][0], "Client sent Bearer token")
	}
}

func TestClientTimeoutAndRetries(t *testing.T) {
	t.Parallel()

	timeout := 200

	l := &sync.Mutex{}
	var delayedRequests int

	s := httptest.NewServer(http.HandlerFunc(
		func(rw http.ResponseWriter, r *http.Request) {
			l.Lock()
			delayRequest := false
			if r.RequestURI == kubeletMetricWithDelay {
				delayedRequests++
				delayRequest = delayedRequests == 1 // We want to lock only the first request to the lagging endpoint.
			}
			l.Unlock()

			if delayRequest {
				time.Sleep(time.Duration(timeout) * 2 * time.Millisecond)
			}
			rw.WriteHeader(200)
		},
	))

	c, cf, inClusterConfig, logger := getTestData(s)

	cf.Timeout = timeout

	kubeletClient, err := internalhttp.NewClient(
		internalhttp.DefaultConnector(c, cf, inClusterConfig, logger),
		internalhttp.WithMaxRetries(2),
	)

	require.NoError(t, err)

	r, err := kubeletClient.Get(kubeletMetricWithDelay)
	require.NoError(t, err, "Client created correctly")
	assert.Equal(t, r.StatusCode, http.StatusOK, "Client request answered successfully")

	assert.Equal(t, 2, delayedRequests, "Client did a successful second retry")
}

func TestClientOptions(t *testing.T) {
	t.Parallel()

	l := &sync.Mutex{}

	s, _ := testHTTPServerWithEndpoints(t, l, []string{healthz, prometheusMetric, kubeletMetric})

	k8sClient, cf, inClusterConfig, logger := getTestData(s)

	_, err := internalhttp.NewClient(
		internalhttp.DefaultConnector(k8sClient, cf, inClusterConfig, logger),
		internalhttp.WithMaxRetries(retries),
		internalhttp.WithLogger(logger),
	)

	assert.NoError(t, err)
}

func getTestData(s *httptest.Server) (*fake.Clientset, *config.Config, *rest.Config, *log.Logger) {
	u, _ := url.Parse(s.URL)
	port, _ := strconv.Atoi(u.Port())

	c := fake.NewSimpleClientset(getTestNode(port))

	cf := &config.Config{
		NodeName: nodeName,
		Host:     u.Hostname(),
	}

	inClusterConfig := &rest.Config{
		Host:            fmt.Sprintf("%s://%s", u.Scheme, u.Host),
		BearerTokenFile: fakeTokenFile,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: true,
		},
	}

	logger := log.New()
	logger.SetOutput(io.Discard)
	// Set level to panic might save a few cycles if we don't even attempt to write to io.Discard.
	logger.SetLevel(log.PanicLevel)

	return c, cf, inClusterConfig, logger
}

func getTestNode(port int) *v1.Node {
	return &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
		},
		Status: v1.NodeStatus{
			DaemonEndpoints: v1.NodeDaemonEndpoints{
				KubeletEndpoint: v1.DaemonEndpoint{
					Port: int32(port),
				},
			},
		},
	}
}

func testHTTPServerWithEndpoints(t *testing.T, l *sync.Mutex, endpoints []string) (*httptest.Server, map[string]*http.Request) {
	t.Helper()

	requestsReceived := map[string]*http.Request{}

	testServer := httptest.NewServer(handler(l, requestsReceived, endpoints))

	return testServer, requestsReceived
}

func testHTTPSServerWithEndpoints(t *testing.T, l *sync.Mutex, endpoints []string) (*httptest.Server, map[string]*http.Request) {
	t.Helper()

	requestsReceived := map[string]*http.Request{}

	testServer := httptest.NewTLSServer(handler(l, requestsReceived, endpoints))

	return testServer, requestsReceived
}

func handler(l sync.Locker, requestsReceived map[string]*http.Request, endpoints []string) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		l.Lock()
		requestsReceived[r.RequestURI] = r
		l.Unlock()

		for _, e := range endpoints {
			if e == r.RequestURI {
				rw.WriteHeader(200)
				return
			}
		}
		rw.WriteHeader(404)
	}
}
