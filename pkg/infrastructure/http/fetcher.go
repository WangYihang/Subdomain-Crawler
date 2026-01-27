package http

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/WangYihang/Subdomain-Crawler/pkg/domain/entity"
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

	// Read request body (usually empty for GET)
	var reqBody string
	if req.Body != nil {
		bodyBytes, _ := io.ReadAll(req.Body)
		reqBody = string(bodyBytes)
		req.Body = io.NopCloser(strings.NewReader(reqBody))
	}

	// Prepare HTTPMessage
	httpMsg := &entity.HTTPMessage{
		Request: &entity.HTTPRequest{
			Method:        req.Method,
			URL:           req.URL.String(),
			Proto:         req.Proto,
			Header:        make(map[string]string),
			Body:          reqBody,
			ContentLength: req.ContentLength,
		},
	}
	for k, v := range req.Header {
		httpMsg.Request.Header[k] = strings.Join(v, ", ")
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return &service.HTTPResponse{URL: url, Error: err.Error(), Message: httpMsg}, err
	}
	defer resp.Body.Close()

	// Limit response size
	limitedReader := io.LimitReader(resp.Body, f.maxResponseSize)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		// Even if reading body fails, we might want to return what we have
		return &service.HTTPResponse{
			URL:        url,
			StatusCode: resp.StatusCode,
			Error:      err.Error(),
			Message:    httpMsg,
		}, err
	}
	bodyStr := string(body)

	headers := make(map[string]string)
	for key, values := range resp.Header {
		headers[key] = strings.Join(values, ", ")
	}

	// Populate response part of HTTPMessage
	httpMsg.Response = &entity.HTTPResponse{
		Proto:         resp.Proto,
		StatusCode:    resp.StatusCode,
		Status:        resp.Status,
		Header:        headers,
		Body:          bodyStr,
		ContentLength: resp.ContentLength,
	}

	return &service.HTTPResponse{
		URL:           url,
		StatusCode:    resp.StatusCode,
		Headers:       headers,
		Body:          bodyStr,
		ContentLength: len(body),
		Message:       httpMsg,
	}, nil
}
