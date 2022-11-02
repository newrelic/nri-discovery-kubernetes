package http

import (
	"net/http"
)

// Getter is an interface for HTTP client with, which should provide
// scheme, port and hostname for the HTTP call.
type Getter interface {
	Get(path string) (*http.Response, error)
}

type Doer interface {
	Do(*http.Request) (*http.Response, error)
}
