package http

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/newrelic/nri-discovery-kubernetes/internal/config"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/transport"
)

const (
	apiProxyPath = "/api/v1/nodes/%s/proxy/"
	httpScheme   = "http"
	httpsScheme  = "https"
)

// Connector provides an interface to retrieve connParams to connect to a Kubelet instance.
type Connector interface {
	Connect() (*connParams, error)
}

type defaultConnector struct {
	logger          *logrus.Logger
	kc              kubernetes.Interface
	inClusterConfig *rest.Config
	config          *config.Config
}

// DefaultConnector returns a defaultConnector that checks connection against local kubelet and api proxy.
func DefaultConnector(kc kubernetes.Interface, config *config.Config, inClusterConfig *rest.Config, logger *logrus.Logger) Connector {
	return &defaultConnector{
		logger:          logger,
		inClusterConfig: inClusterConfig,
		kc:              kc,
		config:          config,
	}
}

// Connect probes the kubelet connection locally and then through the apiServer proxy.
// It tries to infer the scheme and the port found in node status DaemonEndpoints, if needed the user can set them up.
// Locally it adds the bearer token from the file to the request if protocol=https sing transport.NewBearerAuthWithRefreshRoundTripper,
// on the other hand passing through api-proxy, authentication is managed by the kubernetes client itself.
// Notice that we cannot use the as well rest.TransportFor to connect locally since the certificate sent by kubelet,
// cannot be verified in the same way we do for the apiServer.
func (dp *defaultConnector) Connect() (*connParams, error) {
	kubeletPort, err := dp.getPort()
	if err != nil {
		return nil, fmt.Errorf("getting kubelet port: %w", err)
	}

	kubeletScheme := dp.schemeFor(dp.config)
	hostURL := net.JoinHostPort(dp.config.Host, fmt.Sprint(kubeletPort))

	dp.logger.Infof("Trying to connect to kubelet locally with scheme=%q hostURL=%q", kubeletScheme, hostURL)
	trip, err := tripperWithBearerTokenAndRefresh(dp.inClusterConfig.BearerTokenFile)
	if err != nil {
		return nil, fmt.Errorf("creating tripper connecting to kubelet through nodeIP: %w", err)
	}

	conn, err := dp.checkLocalConnection(trip, kubeletScheme, hostURL)
	if err == nil {
		dp.logger.Infof("Connected to Kubelet through nodeIP with scheme=%q hostURL=%q", kubeletScheme, hostURL)
		return conn, nil
	}
	dp.logger.Warnf("Kubelet not reachable locally with scheme=%q hostURL=%q: %v", kubeletScheme, hostURL, err)
	dp.logger.Warnf("this could lead to interrogate the API Server too much to an extent the cluster may suffer. Fix you configuration!")

	dp.logger.Infof("Trying to connect to kubelet through API proxy %q to node %q", dp.inClusterConfig.Host, dp.config.NodeName)
	tripperAPI, err := rest.TransportFor(dp.inClusterConfig)
	if err != nil {
		return nil, fmt.Errorf("creating tripper connecting to kubelet through API server proxy: %w", err)
	}

	conn, err = dp.checkConnectionAPIProxy(dp.inClusterConfig.Host, dp.config.NodeName, tripperAPI)
	if err != nil {
		return nil, fmt.Errorf("creating connection parameters for API proxy: %w", err)
	}

	return conn, nil
}

func (dp *defaultConnector) checkLocalConnection(tripperWithBearerTokenRefreshing http.RoundTripper, scheme string, hostURL string) (*connParams, error) {
	dp.logger.Debugf("connecting to kubelet directly with nodeIP")
	var err error
	var conn *connParams

	switch scheme {
	case httpScheme:
		if conn, err = dp.checkConnectionHTTP(hostURL); err == nil {
			return conn, nil
		}
	case httpsScheme:
		if conn, err = dp.checkConnectionHTTPS(hostURL, tripperWithBearerTokenRefreshing); err == nil {
			return conn, nil
		}
	default:
		dp.logger.Infof("Checking both HTTP and HTTPS since the scheme was not detected automatically, " +
			"you can set set kubelet.scheme to avoid this behaviour")

		if conn, err = dp.checkConnectionHTTPS(hostURL, tripperWithBearerTokenRefreshing); err == nil {
			return conn, nil
		}

		if conn, err = dp.checkConnectionHTTP(hostURL); err == nil {
			return conn, nil
		}
	}

	return nil, fmt.Errorf("no connection succeeded through localhost: %w", err)
}

func (dp *defaultConnector) getPort() (int32, error) {
	if dp.config.Port != 0 {
		dp.logger.Debugf("Setting Port %d as specified by user config", dp.config.Port)
		return int32(dp.config.Port), nil
	}

	// We pay the price of a single call getting a node to avoid asking the user the Kubelet port.
	node, err := dp.kc.CoreV1().Nodes().Get(context.Background(), dp.config.NodeName, metav1.GetOptions{})
	if err != nil {
		return 0, fmt.Errorf("getting node %q: %w", dp.config.NodeName, err)
	}

	port := node.Status.DaemonEndpoints.KubeletEndpoint.Port
	dp.logger.Debugf("Setting Port %d as found in status condition", port)

	return port, nil
}

func (dp *defaultConnector) schemeFor(c *config.Config) string {
	if c.TLS {
		dp.logger.Debugf("flag for TLS explicitly set. using schema %s", httpsScheme)
		return httpsScheme
	}

	switch c.Port {
	case defaultHTTPKubeletPort:
		dp.logger.Debugf("Setting Kubelet Endpoint Scheme http since Port is %d", c.Port)
		return httpScheme
	case defaultHTTPSKubeletPort:
		dp.logger.Debugf("Setting Kubelet Endpoint Scheme https since Port is %d", c.Port)
		return httpsScheme
	}

	dp.logger.Warningf("cannot automatically figure out scheme from non-standard port %d, and no TLS flag has been provided. Defaulting to %s", c.Port, httpScheme)
	return httpScheme
}

type connParams struct {
	url    url.URL
	client HTTPDoer
}

func (dp *defaultConnector) checkConnectionAPIProxy(apiServer string, nodeName string, tripperAPIproxy http.RoundTripper) (*connParams, error) {
	apiURL, err := url.Parse(apiServer)
	if err != nil {
		return nil, fmt.Errorf("parsing kubernetes api url from in cluster config: %w", err)
	}

	conn := connParams{
		client: &http.Client{
			Timeout:   time.Duration(dp.config.Timeout) * time.Millisecond,
			Transport: tripperAPIproxy,
		},
		url: url.URL{
			Host:   apiURL.Host,
			Scheme: apiURL.Scheme,
			Path:   path.Join(fmt.Sprintf(apiProxyPath, nodeName)),
		},
	}

	dp.logger.Debugf("Testing kubelet connection through API proxy: %s%s", apiURL.Host, conn.url.Path)

	if err = checkConnection(conn); err != nil {
		return nil, fmt.Errorf("checking connection via API proxy: %w", err)
	}

	return &conn, nil
}

func (dp *defaultConnector) checkConnectionHTTP(hostURL string) (*connParams, error) {
	dp.logger.Debugf("testing kubelet connection over plain http to %s", hostURL)

	conn := dp.defaultConnParamsHTTP(hostURL)
	if err := checkConnection(conn); err != nil {
		return nil, fmt.Errorf("checking connection via API proxy: %w", err)
	}

	return &conn, nil
}

func (dp *defaultConnector) checkConnectionHTTPS(hostURL string, tripperBearerRefreshing http.RoundTripper) (*connParams, error) {
	dp.logger.Debugf("testing kubelet connection over https to %s", hostURL)

	conn := dp.defaultConnParamsHTTPS(hostURL, tripperBearerRefreshing)
	if err := checkConnection(conn); err != nil {
		return nil, fmt.Errorf("checking connection via API proxy: %w", err)
	}

	return &conn, nil
}

func checkConnection(conn connParams) error {
	conn.url.Path = path.Join(conn.url.Path, healthzPath)

	r, err := http.NewRequest(http.MethodGet, conn.url.String(), nil)
	if err != nil {
		return fmt.Errorf("creating request to %q: %v", conn.url.String(), err)
	}

	resp, err := conn.client.Do(r)
	if err != nil {
		return fmt.Errorf("connecting to %q: %w", conn.url.String(), err)
	}
	defer resp.Body.Close() // nolint: errcheck

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("calling %s got non-200 status code: %d", conn.url.String(), resp.StatusCode)
	}

	return nil
}

func tripperWithBearerTokenAndRefresh(tokenFile string) (http.RoundTripper, error) {
	// Here we're using the default http.Transport configuration, but with a modified TLS config.
	// The DefaultTransport is casted to an http.RoundTripper interface, so we need to convert it back.
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.TLSClientConfig.InsecureSkipVerify = true
	t.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	// Use the default kubernetes Bearer token authentication RoundTripper
	tripperWithBearerRefreshing, err := transport.NewBearerAuthWithRefreshRoundTripper("", tokenFile, t)
	if err != nil {
		return nil, fmt.Errorf("creating bearerAuthWithRefreshRoundTripper: %w", err)
	}

	return tripperWithBearerRefreshing, nil
}

func (dp *defaultConnector) defaultConnParamsHTTP(hostURL string) connParams {
	httpClient := &http.Client{
		Timeout: time.Duration(dp.config.Timeout) * time.Millisecond,
	}

	u := url.URL{
		Host:   hostURL,
		Scheme: httpScheme,
	}
	return connParams{u, httpClient}
}

func (dp *defaultConnector) defaultConnParamsHTTPS(hostURL string, tripper http.RoundTripper) connParams {
	httpClient := &http.Client{
		Timeout: time.Duration(dp.config.Timeout) * time.Millisecond,
	}

	httpClient.Transport = tripper

	u := url.URL{
		Host:   hostURL,
		Scheme: httpsScheme,
	}
	return connParams{u, httpClient}
}

type fixedConnector struct {
	URL    url.URL
	Client HTTPDoer
}

// Connect return connParams without probing any endpoint.
func (mc *fixedConnector) Connect() (*connParams, error) {
	return &connParams{
		url:    mc.URL,
		client: mc.Client,
	}, nil
}

// StaticConnector returns a fixed connector that does not check the connection when calling .Connect().
func StaticConnector(client HTTPDoer, u url.URL) Connector {
	return &fixedConnector{
		URL:    u,
		Client: client,
	}
}
