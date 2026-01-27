package fetcher

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/WangYihang/Subdomain-Crawler/pkg/extract"
	"github.com/WangYihang/Subdomain-Crawler/pkg/httpclient"
)

// Result represents fetch result
type Result struct {
	Domain             string
	Root               string
	Subdomains         []string
	Title              string
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
}

// Config holds fetcher config
type Config struct {
	Client          *httpclient.Client
	Filter          *extract.Filter
	MaxResponseSize int64
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

// fetchURL fetches URL
func (f *Fetcher) fetchURL(url string, result *Result) error {
	startTime := time.Now()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		result.Error = err.Error()
		return err
	}

	resp, err := f.client.Do(req)
	if err != nil {
		result.Error = err.Error()
		return err
	}
	defer resp.Body.Close()

	result.ResponseStatusCode = resp.StatusCode
	result.ResponseTime = time.Since(startTime).Milliseconds()

	bodyReader := io.LimitReader(resp.Body, f.maxResponseSize)
	bodyCloser := io.NopCloser(bodyReader)

	domains, err := f.extractor.FromBody(bodyCloser)
	if err != nil && err != io.EOF {
		result.Error = fmt.Sprintf("extraction error: %v", err)
		return err
	}

	filtered := extract.FilterBySuffix(domains, result.Root)
	sanitized := make([]string, len(filtered))
	for i, d := range filtered {
		sanitized[i] = f.sanitizer.Sanitize(d)
	}
	result.Subdomains = f.deduplicator.Deduplicate(sanitized)

	return nil
}
