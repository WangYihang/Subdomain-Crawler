package http

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/WangYihang/Subdomain-Crawler/pkg/domain/service"
)

// Fetcher implements service.HTTPFetcher
type Fetcher struct {
	client          *http.Client
	maxResponseSize int64
	userAgent       string
}

// Config holds HTTP fetcher configuration
type Config struct {
	Timeout         time.Duration
	MaxResponseSize int64
	UserAgent       string
}

// NewFetcher creates a new HTTP fetcher
func NewFetcher(config Config) *Fetcher {
	return &Fetcher{
		client: &http.Client{
			Timeout: config.Timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
		maxResponseSize: config.MaxResponseSize,
		userAgent:       config.UserAgent,
	}
}

// Fetch implements service.HTTPFetcher
func (f *Fetcher) Fetch(url string) (*service.HTTPResponse, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return &service.HTTPResponse{URL: url, Error: err.Error()}, err
	}

	req.Header.Set("User-Agent", f.userAgent)

	resp, err := f.client.Do(req)
	if err != nil {
		return &service.HTTPResponse{URL: url, Error: err.Error()}, err
	}
	defer resp.Body.Close()

	// Limit response size
	limitedReader := io.LimitReader(resp.Body, f.maxResponseSize)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return &service.HTTPResponse{
			URL:        url,
			StatusCode: resp.StatusCode,
			Error:      err.Error(),
		}, err
	}

	headers := make(map[string]string)
	for key, values := range resp.Header {
		headers[key] = strings.Join(values, ", ")
	}

	return &service.HTTPResponse{
		URL:           url,
		StatusCode:    resp.StatusCode,
		Headers:       headers,
		Body:          string(body),
		ContentLength: len(body),
	}, nil
}
