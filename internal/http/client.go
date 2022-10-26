package http

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"

	"github.com/sethgrid/pester"
	log "github.com/sirupsen/logrus"
)

const (
	healthzPath             = "/healthz"
	defaultHTTPKubeletPort  = 10255
	defaultHTTPSKubeletPort = 10250
)

// Client implements a client for Kubelet, capable of retrieving prometheus metrics from a given endpoint.
type Client struct {
	logger   *log.Logger
	doer     HTTPDoer
	endpoint url.URL
	retries  int
}

type OptionFunc func(kc *Client) error

// WithMaxRetries returns an OptionFunc to change the number of retries used int Pester Client.
func WithMaxRetries(retries int) OptionFunc {
	return func(kubeletClient *Client) error {
		kubeletClient.retries = retries
		return nil
	}
}

// New builds a Client using the given options.
func New(connector Connector, opts ...OptionFunc) (*Client, error) {
	c := &Client{
		logger: log.New(),
	}
	// In case WithLogger option is not used, we discard all the logs
	c.logger.SetOutput(io.Discard)
	// Set level to panic might save a few cycles if we don't even attempt to write to io.Discard.
	c.logger.SetLevel(log.PanicLevel)

	for i, opt := range opts {
		if err := opt(c); err != nil {
			return nil, fmt.Errorf("applying option #%d: %w", i, err)
		}
	}

	if connector == nil {
		return nil, fmt.Errorf("connector should not be nil")
	}

	conn, err := connector.Connect()
	if err != nil {
		return nil, fmt.Errorf("connecting to kubelet using the connector: %w", err)
	}

	if client, ok := conn.client.(*http.Client); ok {
		httpPester := pester.NewExtendedClient(client)
		httpPester.Backoff = pester.LinearBackoff
		httpPester.MaxRetries = c.retries
		httpPester.LogHook = func(e pester.ErrEntry) {
			c.logger.Debugf("getting data from kubelet: %v", e)
		}
		c.doer = httpPester
	} else {
		c.logger.Debugf("running kubelet client without pester")
		c.doer = conn.client
	}

	c.endpoint = conn.url

	return c, nil
}

// Get implements HTTPGetter interface by sending GET request using configured client.
func (c *Client) Get(urlPath string) (*http.Response, error) {
	// Notice that this is the client to interact with kubelet. In case of CAdvisor the MetricFamiliesGetFunc is used
	e := c.endpoint
	e.Path = path.Join(c.endpoint.Path, urlPath)

	r, err := http.NewRequest(http.MethodGet, e.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request to: %s. Got error: %s ", e.String(), err)
	}

	c.logger.Debugf("Calling Kubelet endpoint: %s", r.URL.String())

	return c.doer.Do(r)
}
