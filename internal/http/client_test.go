package http_test

import (
	"fmt"
	internalhttp "github.com/newrelic/nri-discovery-kubernetes/internal/http"
	log "github.com/sirupsen/logrus"
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
	fakeTokenFile          = "./test_data/token"
	retries                = 3
)

func TestClientCalls(t *testing.T) {
	s, requests := testHTTPServerWithEndpoints(t, []string{healthz, prometheusMetric, kubeletMetric})

	k8sClient, cf, inClusterConfig, logger := getTestData(s)

	kubeletClient, err := internalhttp.New(
		internalhttp.DefaultConnector(k8sClient, cf, inClusterConfig, logger),
		internalhttp.WithMaxRetries(retries),
		internalhttp.WithLogger(logger),
	)

	t.Run("creation_succeeds_receiving_200", func(t *testing.T) {
		require.NoError(t, err)
	})

	t.Run("hits_only_local_kubelet", func(t *testing.T) {
		require.NotNil(t, requests)

		_, found := requests[healthz]
		assert.True(t, found)
	})

	t.Run("hits_kubelet_metric", func(t *testing.T) {
		r, err := kubeletClient.Get(kubeletMetric)
		assert.NoError(t, err)
		assert.Equal(t, r.StatusCode, http.StatusOK)

		_, found := requests[kubeletMetric]
		assert.True(t, found)
	})
}

func TestClientCallsViaAPIProxy(t *testing.T) {
	t.Parallel()

	s, requests := testHTTPSServerWithEndpoints(
		t,
		[]string{path.Join(apiProxy, healthz), path.Join(apiProxy, prometheusMetric), path.Join(apiProxy, kubeletMetric)},
	)

	k8sClient, cf, inClusterConfig, logger := getTestData(s)
	cf.Host = "invalid" // disabling local connection

	kubeletClient, err := internalhttp.New(
		internalhttp.DefaultConnector(k8sClient, cf, inClusterConfig, logger),
		internalhttp.WithMaxRetries(retries),
		internalhttp.WithLogger(logger),
	)

	t.Run("creation_succeeds_receiving_200", func(t *testing.T) {
		t.Parallel()

		require.NoError(t, err)
	})

	t.Run("hits_api_server_as_fallback", func(t *testing.T) {
		t.Parallel()

		require.NotNil(t, requests)

		_, found := requests[path.Join(apiProxy, healthz)]
		assert.True(t, found)
	})

	t.Run("hits_kubelet_metric_through_proxy", func(t *testing.T) {
		t.Parallel()

		r, err := kubeletClient.Get(kubeletMetric)
		assert.NoError(t, err)
		assert.Equal(t, r.StatusCode, http.StatusOK)

		_, found := requests[path.Join(apiProxy, kubeletMetric)]
		assert.True(t, found)
	})
}

func TestConfigPrecedence(t *testing.T) {
	t.Parallel()

	t.Run("connector_takes_port", func(t *testing.T) {
		t.Parallel()

		s, _ := testHTTPServerWithEndpoints(t, []string{healthz, prometheusMetric, kubeletMetric})
		_, cf, inClusterConfig, logger := getTestData(s)

		// We use an empty client, but the connector is retrieving the port from the config.
		k8sClient := fake.NewSimpleClientset()
		u, _ := url.Parse(s.URL)
		port, _ := strconv.Atoi(u.Port())
		cf.Port = port

		_, err := internalhttp.New(
			internalhttp.DefaultConnector(k8sClient, cf, inClusterConfig, logger),
			internalhttp.WithMaxRetries(retries),
			internalhttp.WithLogger(logger),
		)
		require.NoError(t, err)
	})
}

func TestClientFailingProbingHTTP(t *testing.T) {
	t.Parallel()

	s, requests := testHTTPServerWithEndpoints(t, []string{})

	c, cf, inClusterConfig, logger := getTestData(s)

	_, err := internalhttp.New(
		internalhttp.DefaultConnector(c, cf, inClusterConfig, logger),
		internalhttp.WithMaxRetries(retries),
		internalhttp.WithLogger(logger),
	)

	t.Run("fails_receiving_404", func(t *testing.T) {
		t.Parallel()

		assert.Error(t, err)
	})

	t.Run("hits_both_api_server_and_local_kubelet", func(t *testing.T) {
		t.Parallel()

		require.NotNil(t, requests)

		_, found := requests[path.Join(apiProxy, healthz)]
		assert.True(t, found)

		_, found = requests[healthz]
		assert.True(t, found)
	})

	t.Run("does_not_attach_bearer_token", func(t *testing.T) {
		t.Parallel()

		var expectedEmptySlice []string
		assert.Equal(t, expectedEmptySlice, requests[healthz].Header["Authorization"])
	})
}

func TestClientFailingProbingHTTPS(t *testing.T) {
	t.Parallel()

	s, requests := testHTTPSServerWithEndpoints(t, []string{})

	c, cf, inClusterConfig, logger := getTestData(s)
	cf.TLS = true

	_, err := internalhttp.New(
		internalhttp.DefaultConnector(c, cf, inClusterConfig, logger),
		internalhttp.WithMaxRetries(retries),
		internalhttp.WithLogger(logger),
	)

	t.Run("fails_receiving_404", func(t *testing.T) {
		t.Parallel()

		assert.Error(t, err)
	})

	t.Run("hits_both_api_server_and_local_kubelet", func(t *testing.T) {
		t.Parallel()

		require.NotNil(t, requests)

		_, found := requests[path.Join(apiProxy, healthz)]
		assert.True(t, found)

		_, found = requests[healthz]
		assert.True(t, found)
	})

	t.Run("does_attach_bearer_token", func(t *testing.T) {
		t.Parallel()

		require.NotNil(t, requests)
		data, err := os.ReadFile(fakeTokenFile)

		require.NoError(t, err)

		for _, v := range requests {
			assert.Equal(t, "Bearer "+string(data), v.Header["Authorization"][0])
		}
	})
}

func TestClientTimeoutAndRetries(t *testing.T) {
	timeout := 200

	l := &sync.Mutex{}
	var requestsReceived int

	s := httptest.NewServer(http.HandlerFunc(
		func(rw http.ResponseWriter, r *http.Request) {
			// We want to lock the first request to the lagging endpoint.
			l.Lock()
			if r.RequestURI == kubeletMetricWithDelay && requestsReceived == 0 {
				requestsReceived++
				l.Unlock()
				time.Sleep(time.Duration(timeout) * 2 * time.Millisecond)
				rw.WriteHeader(200)
				return
			}
			l.Unlock()
			rw.WriteHeader(200)
		},
	))

	c, cf, inClusterConfig, logger := getTestData(s)

	cf.Timeout = timeout

	kubeletClient, err := internalhttp.New(
		internalhttp.DefaultConnector(c, cf, inClusterConfig, logger),
		internalhttp.WithMaxRetries(2),
	)

	require.NoError(t, err)

	t.Run("gets_200_after_retry", func(t *testing.T) {
		r, err := kubeletClient.Get(kubeletMetricWithDelay)
		require.NoError(t, err)
		assert.Equal(t, r.StatusCode, http.StatusOK)

		// 1 because it has to fail the first lagged request
		assert.Equal(t, requestsReceived, 1)
	})
}

func TestClientOptions(t *testing.T) {
	t.Parallel()

	s, _ := testHTTPServerWithEndpoints(t, []string{healthz, prometheusMetric, kubeletMetric})

	k8sClient, cf, inClusterConfig, logger := getTestData(s)

	_, err := internalhttp.New(
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

func testHTTPServerWithEndpoints(t *testing.T, endpoints []string) (*httptest.Server, map[string]*http.Request) {
	t.Helper()

	requestsReceived := map[string]*http.Request{}
	l := sync.Mutex{}

	testServer := httptest.NewServer(handler(&l, "http", requestsReceived, endpoints))

	return testServer, requestsReceived
}

func testHTTPSServerWithEndpoints(t *testing.T, endpoints []string) (*httptest.Server, map[string]*http.Request) {
	t.Helper()

	requestsReceived := map[string]*http.Request{}
	l := sync.Mutex{}

	testServer := httptest.NewTLSServer(handler(&l, "https", requestsReceived, endpoints))

	return testServer, requestsReceived
}

func handler(l sync.Locker, scheme string, requestsReceived map[string]*http.Request, endpoints []string) http.HandlerFunc {
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
