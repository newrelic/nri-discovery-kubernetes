package http

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/newrelic/nri-kubernetes/v2/src/client"
	kubeletclient "github.com/newrelic/nri-kubernetes/v2/src/kubelet/client"
	"github.com/sirupsen/logrus"
)

type HttpClient interface {
	Get(path string) ([]byte, error)
}

type httpClient struct {
	token string
	http  http.Client
	url   url.URL
}

func (c *httpClient) Get(path string) ([]byte, error) {
	endpoint := c.url.String() + path
	req, _ := http.NewRequest(http.MethodGet, endpoint, nil)
	req.Header.Add("Authorization", "Bearer "+c.token)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	buff, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("httpClient buffer: %q, Status %s", string(buff), resp.Status)
	}

	return buff, nil
}

func NewClient(url url.URL, token string) HttpClient {
	client := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	return &httpClient{
		http:  client,
		url:   url,
		token: token,
	}
}

// kubeletClient addapts the nri-kubernetes kubelet client.
type kubeletClient struct {
	client client.HTTPClient
}

// NewKubeletClient creates a new kubeletClient instance.
func NewKubeletClient(nodeName string, timeout time.Duration) (HttpClient, error) {
	logger := logrus.New()
	logger.SetOutput(os.Stderr)
	d, err := kubeletclient.NewDiscoverer(nodeName, logger)
	if err != nil {
		return nil, err
	}
	client, err := d.Discover(timeout)
	if err != nil {
		return nil, err
	}
	return &kubeletClient{
		client: client,
	}, nil
}

func (kc *kubeletClient) Get(path string) ([]byte, error) {
	resp, err := kc.client.Do(http.MethodGet, path)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	buff, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("kubeletClient buffer: %q, Status %s", string(buff), resp.Status)
	}
	return buff, nil
}
