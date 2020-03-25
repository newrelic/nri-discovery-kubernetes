package http

import (
	"crypto/tls"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
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

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(resp.Status)
	}

	buff, _ := ioutil.ReadAll(resp.Body)
	_ = resp.Body.Close()
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
