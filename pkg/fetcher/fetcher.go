package fetcher

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/WangYihang/Subdomain-Crawler/pkg/extract"
	"github.com/WangYihang/Subdomain-Crawler/pkg/httpclient"
	"github.com/WangYihang/Subdomain-Crawler/pkg/output"
)

const httpLogBodyPreviewBytes = 4096

// Result represents fetch result
type Result struct {
	Domain             string
	Root               string
	Subdomains         []string
	Title              string
	ContentLength      int64 // from HTTP header, -1 when unknown
	ResponseStatusCode int
	ResponseTime       int64
	Error              string
	Timestamp          int64
}

// Fetcher handles fetching
type Fetcher struct {
	client          *httpclient.Client
	extractor       *extract.DomainExtractor
	filter          *extract.Filter
	deduplicator    *extract.Deduplicator
	sanitizer       *extract.Sanitizer
	maxResponseSize int64
	httpLog         *output.JsonlWriter
}

// Config holds fetcher config
type Config struct {
	Client          *httpclient.Client
	Filter          *extract.Filter
	MaxResponseSize int64
	HttpLog         *output.JsonlWriter
}

// NewFetcher creates fetcher
func NewFetcher(config *Config) *Fetcher {
	if config.MaxResponseSize == 0 {
		config.MaxResponseSize = 10 * 1024 * 1024
	}

	return &Fetcher{
		client:          config.Client,
		extractor:       extract.NewDomainExtractor(),
		filter:          config.Filter,
		deduplicator:    extract.NewDeduplicator(),
		sanitizer:       extract.NewSanitizer(),
		maxResponseSize: config.MaxResponseSize,
		httpLog:         config.HttpLog,
	}
}

// Fetch fetches domain
func (f *Fetcher) Fetch(domain, root string, protocols []string) *Result {
	result := &Result{
		Domain:    domain,
		Root:      root,
		Timestamp: time.Now().UnixMilli(),
	}

	for _, protocol := range protocols {
		url := fmt.Sprintf("%s://%s/", protocol, domain)
		if err := f.fetchURL(url, result); err == nil {
			return result
		}
	}

	if result.Error == "" {
		result.Error = "failed to fetch"
	}

	return result
}

// headersToMap converts http.Header to map[string][]string for JSON.
func headersToMap(h http.Header) map[string][]string {
	if h == nil {
		return nil
	}
	m := make(map[string][]string, len(h))
	for k, v := range h {
		m[k] = v
	}
	return m
}

func (f *Fetcher) writeHTTPLog(req *http.Request, url string, resp *http.Response, body []byte, result *Result, startTime time.Time) {
	if f.httpLog == nil {
		return
	}
	requestAt := startTime.UnixMilli()
	responseAt := time.Now().UnixMilli()
	durationMs := result.ResponseTime

	reqLog := map[string]interface{}{
		"method":     "GET",
		"url":        url,
		"request_at": requestAt,
		"timestamp":  startTime.Format(time.RFC3339Nano),
	}
	if req != nil {
		reqLog["method"] = req.Method
		reqLog["url"] = req.URL.String()
		reqLog["host"] = req.Host
		reqLog["headers"] = headersToMap(req.Header)
	} else {
		reqLog["headers"] = map[string][]string(nil)
	}

	respLog := map[string]interface{}{
		"error":            result.Error,
		"response_time_ms": durationMs,
		"response_at":      responseAt,
		"timestamp":        time.Now().Format(time.RFC3339Nano),
	}
	if resp != nil {
		respLog["status"] = resp.Status
		respLog["status_code"] = resp.StatusCode
		respLog["headers"] = headersToMap(resp.Header)
		respLog["content_length"] = resp.ContentLength
		respLog["title"] = result.Title
	}
	if body != nil {
		respLog["body_size"] = len(body)
		respLog["body_truncated"] = int64(len(body)) >= f.maxResponseSize
		if len(body) > 0 {
			preview := body
			if len(preview) > httpLogBodyPreviewBytes {
				preview = preview[:httpLogBodyPreviewBytes]
			}
			respLog["body_preview"] = string(preview)
		}
	}

	_ = f.httpLog.Log(map[string]interface{}{
		"request":  reqLog,
		"response": respLog,
	})
}

// fetchURL fetches URL
func (f *Fetcher) fetchURL(url string, result *Result) error {
	startTime := time.Now()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		result.Error = err.Error()
		result.ResponseTime = time.Since(startTime).Milliseconds()
		f.writeHTTPLog(nil, url, nil, nil, result, startTime)
		return err
	}

	resp, err := f.client.Do(req)
	if err != nil {
		result.Error = err.Error()
		result.ResponseTime = time.Since(startTime).Milliseconds()
		f.writeHTTPLog(req, url, nil, nil, result, startTime)
		return err
	}
	defer resp.Body.Close()

	result.ResponseStatusCode = resp.StatusCode
	result.ResponseTime = time.Since(startTime).Milliseconds()
	result.ContentLength = resp.ContentLength

	body, err := io.ReadAll(io.LimitReader(resp.Body, f.maxResponseSize))
	if err != nil {
		result.Error = fmt.Sprintf("read body: %v", err)
		f.writeHTTPLog(req, url, resp, nil, result, startTime)
		return err
	}
	bodyStr := string(body)
	result.Title = extract.ExtractTitle(bodyStr)

	domains := f.extractor.FromString(bodyStr)
	filtered := extract.FilterBySuffix(domains, result.Root)
	sanitized := make([]string, len(filtered))
	for i, d := range filtered {
		sanitized[i] = f.sanitizer.Sanitize(d)
	}
	result.Subdomains = f.deduplicator.Deduplicate(sanitized)

	f.writeHTTPLog(req, url, resp, body, result, startTime)
	return nil
}
