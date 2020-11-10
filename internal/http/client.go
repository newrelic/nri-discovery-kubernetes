package http

import (
	"crypto/tls"
	"errors"
	nriK8sClient "github.com/newrelic/nri-kubernetes/src/client"
	nriKubeletClient "github.com/newrelic/nri-kubernetes/src/kubelet/client"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	netHttp "net/http"
	"net/url"
	"os"
	"time"
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
		return nil, errors.New(resp.Status)
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
	client nriK8sClient.HTTPClient
}

// NewKubeletClient creates a new kubeletClient instance.
func NewKubeletClient(nodeName string, timeout time.Duration) (HttpClient, error) {
	logger := logrus.New()
	logger.SetOutput(os.Stderr)
	d, err := nriKubeletClient.NewDiscoverer(nodeName, logger)
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
	resp, err := kc.client.Do(netHttp.MethodGet, path)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	buff, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != netHttp.StatusOK {
		return nil, errors.New(resp.Status)
	}
	return buff, nil
}
